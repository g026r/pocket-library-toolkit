package ui

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/g026r/pocket-library-editor/pkg/io"
	"github.com/g026r/pocket-library-editor/pkg/models"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

type errMsg struct {
	err   error
	fatal bool
}

type initDoneMsg struct{}

type updateMsg struct{}

// tickMsg is just a generic message fired by tickCmd
// The time.Time could conceivably be used to make certain messages aren't processed out of order but isn't
type tickMsg time.Time

// tickCmd is used by the progress bar. It fires 1/5 of a second after it's started, and it is necessary to call it
// again if another tick is desired
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type Model struct {
	updates      chan Model // updates is used for passing updates to the Model rather than using pointers
	rootDir      fs.FS
	entries      []models.Entry
	playTimes    map[uint32]models.PlayTime
	thumbnails   map[models.System]models.Thumbnails
	internal     map[models.System][]models.Entry // internal is a map of all known possible entries, grouped by system. For eventual use with add & adv. editing, maybe.
	*io.Config                                    // io.Config is a pointer as we need to be able to read this value in the configDelegate, which doesn't have access to Model
	*stack                                        // stack contains the stack of screens. Useful for when we go up a screen, as a few have multiple possible parents.
	spinner      spinner.Model                    // spinner is used for calls where we don't know the percentage. Mostly this means the initial loading screen
	progress     *progress.Model                  // progress is used for calls where we do know the percentage; has to be a pointer as the screen size event calls before we've finished initializing the Model
	percent      *float64                         // the percent of the progress bar
	err          error                            // err is used to print out an error if the program has to exit early
	wait         string                           // wait is the message to display while waiting
	anyKey       bool                             // anyKey tells View whether we're waiting for a key input or not
	mainMenu     *list.Model
	subMenu      *list.Model // subMenu covers the library & thumbnail options: menus where esc goes up a screen but filtering is disabled
	configMenu   *list.Model // configMenu is a special case as it needs a different delegate renderer from subMenu
	gameList     *list.Model // gameList covers anything that lists all the games in the library
	gameInput    []textinput.Model
	focusedInput int
}

func NewModel() tea.Model {
	prog := progress.New(progress.WithScaledGradient("#006699", "#00ccff"))
	config := io.Config{}

	return Model{
		updates:    make(chan Model, 1),
		stack:      &stack{[]screen{Initializing}},
		spinner:    spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		progress:   &prog,
		Config:     &config,
		mainMenu:   NewMainMenu(),
		configMenu: NewConfigMenu(&config),
		subMenu:    NewSubMenu(),  // subMenu needs to be set even without items to avoid a nil pointer with the initial WindowSizeMsg
		gameList:   NewGameMenu(), // same for gameList
		gameInput:  NewTextInput(),
	}
}

const (
	name = iota
	system
	crc
	sig
	magic
	added
	play
	submit
	cancel
)

func NewTextInput() []textinput.Model {
	inputs := make([]textinput.Model, play+1)

	inputs[system] = textinput.New()
	// TODO: Should we go with full suggestions instead?
	inputs[system].SetSuggestions([]string{models.GB.String(), models.GBC.String(), models.GBA.String(), models.GG.String(), models.SMS.String(), models.NGP.String(), models.NGPC.String(), models.PCE.String(), models.Lynx.String()})
	inputs[system].Prompt = "System: "
	inputs[system].Placeholder = models.GB.String()
	inputs[system].Validate = sysValidate
	inputs[system].ShowSuggestions = true

	inputs[name] = textinput.New()
	inputs[name].Prompt = "Name: "
	inputs[name].Validate = notBlank

	inputs[crc] = textinput.New()
	inputs[crc].Prompt = "CRC32: "
	inputs[crc].Placeholder = "0x00000000"
	inputs[crc].Validate = hexValidate

	inputs[sig] = textinput.New()
	inputs[sig].Prompt = "Signature: "
	inputs[sig].Placeholder = "0x00000000"
	inputs[sig].Validate = hexValidate

	inputs[magic] = textinput.New()
	inputs[magic].Prompt = "Magic Number: "
	inputs[magic].Placeholder = "0x0000"
	inputs[magic].Validate = hexValidate

	inputs[added] = textinput.New()
	inputs[added].Prompt = "Date Added: "
	inputs[added].Placeholder = "2024-01-15 13:24" // Will be replaced eventually
	inputs[added].Validate = dateValidate

	inputs[play] = textinput.New()
	inputs[play].Prompt = "Played: "
	inputs[play].Placeholder = "0h 0m 0s"
	inputs[play].Validate = playValidate

	for i := range inputs {
		inputs[i].PromptStyle = itemStyle
		inputs[i].Cursor.Style = selectedItemStyle.PaddingLeft(0)
		inputs[i].TextStyle = itemStyle.PaddingLeft(2)
	}

	return inputs
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.initSystem)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	switch msg := msg.(type) {
	case updateMsg:
		m = <-m.updates
		if m.Peek() == Waiting {
			m.anyKey = true
		}
	case tea.KeyMsg:
		if m.anyKey {
			m.anyKey = false
			m.Pop()
			*m.percent = 0.0
			return m, nil
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		if m.percent != nil {
			break // percent gets set as the last step of initialization. If it's not nil, we can stop the spinner.
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tickMsg:
		if *m.percent < 1.0 {
			return m, tickCmd()
		}
		return m, m.progress.SetPercent(1.0)
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		*m.progress = progressModel.(progress.Model)
		return m, cmd
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8
		m.mainMenu.SetHeight(msg.Height)
		m.mainMenu.SetWidth(msg.Width)
		m.subMenu.SetHeight(msg.Height)
		m.subMenu.SetWidth(msg.Width)
		m.configMenu.SetHeight(msg.Height)
		m.configMenu.SetWidth(msg.Width)
		m.gameList.SetHeight(msg.Height)
		m.gameList.SetWidth(msg.Width)
		return m, nil
	case initDoneMsg:
		m = <-m.updates // Replace the ui we have with the new, initialized one. Fine in this case as we return m further down the method.
		m.Clear()
		m.Push(MainMenu) // Finished initializing. Replace the stack with a new one containing only the main menu
		return m, tea.ClearScreen
	case errMsg:
		m.err = msg.err
		if msg.fatal {
			m.Push(FatalError)
			return m, tea.Sequence(tea.ExitAltScreen, tea.Quit) // Need to exit alt screen first or the error message doesn't appear for long enough
		}
	}

	if s := m.Peek(); s == MainMenu || s == LibraryMenu || s == ThumbMenu || s == ConfigMenu ||
		s == EditList || s == RemoveList || s == GenerateList {
		return m.menuHandler(msg)
	} else if s == AddScreen || s == EditScreen {
		return m.inputHandler(msg)
	}

	return m, nil
}

func (m Model) View() (s string) {
	switch m.Peek() {
	case Initializing:
		s = fmt.Sprintf("  %s Loading your Pocket library. Please wait.", m.spinner.View())
		//s = m.menu.View()
	case Waiting:
		s = fmt.Sprintf("\n  %s\n\n  %s", m.wait, m.progress.ViewAs(*m.percent))
		if m.anyKey {
			s = fmt.Sprintf("%s\n\n  Press any key to continue.", s)
		}
	case Saving:
		s = fmt.Sprintf("\n  Saving your Pocket library\n\n  %s", m.progress.ViewAs(*m.percent))
	case FatalError:
		s = fmt.Sprintf("FATAL ERROR: %v\n", m.err)
	case MainMenu:
		s = m.mainMenu.View()
	case LibraryMenu, ThumbMenu:
		s = m.subMenu.View()
	case ConfigMenu:
		s = m.configMenu.View()
	case RemoveList, EditList, GenerateList:
		s = m.gameList.View()
	case EditScreen:
		// TODO: Add a title to the edit & add screens
		s = m.editScreenView()
	case AddScreen:
		s = m.addScreenView()
	default:
		panic("Panic! At the View() call")
	}

	return
}

// initSystem loads all our data from disk
func (m Model) initSystem() tea.Msg {
	var d string
	var err error

	switch len(os.Args) {
	case 1:
		if d, err = os.Executable(); err != nil {
			return errMsg{err, true}
		}
		d = filepath.Dir(d)
	case 2:
		d = os.Args[1]
	default:
	}

	d, err = filepath.Abs(d)
	if err != nil {
		return errMsg{err, true}
	}

	fi, err := os.Stat(d)
	if err != nil {
		return errMsg{err, true}
	} else if !fi.IsDir() {
		return errMsg{fmt.Errorf("%s is not a directory", d), true}
	}
	m.rootDir = os.DirFS(d)

	c, err := io.LoadConfig()
	if err != nil {
		return errMsg{err, true}
	}
	*m.Config = c

	e, err := io.LoadEntries(m.rootDir)
	if err != nil {
		return errMsg{err, true}
	}
	m.entries = e

	p, err := io.LoadPlaytimes(m.rootDir)
	if err != nil {
		return errMsg{err, true}
	}
	m.playTimes = p

	if len(m.entries) != len(m.playTimes) {
		return errMsg{fmt.Errorf("entry count mismatch between list.bin [%d] & playtimes.bin [%d]", len(m.entries), len(m.playTimes)), true}
	}

	t, err := io.LoadThumbs(m.rootDir)
	if err != nil {
		return errMsg{err, true}
	}
	m.thumbnails = t

	if m.ShowAdd { // Only need to load these if we're showing the add option
		i, err := io.LoadInternal()
		if err != nil {
			return errMsg{err, true}
		}
		m.internal = i
	}

	per := 0.0
	m.percent = &per // Setting this value both prevents nil pointer dereferences & is used as the signal to stop the spinner

	m.updates <- m
	return initDoneMsg{}
}

// save is the opposite of init: save our data to disk
func (m Model) save() tea.Msg {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = os.Mkdir(fmt.Sprintf("%s/library-editor", wd), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return errMsg{err, true}
	}

	ctr := 0.0
	tick := make(chan any)
	total := float64(len(m.entries))
	for _, v := range m.thumbnails {
		if v.Modified {
			total = total + float64(len(v.Images)) // Only increase the total if they've been modified since we don't write them out otherwise.
		}
	}
	total = total + 1 // Add 1 for the config

	go func() { // Run these in a goroutine to avoid having to pass around the pointer to the progress value as that would require knowing the total as well
		defer close(tick)
		if err := io.SaveLibrary(m.entries, m.playTimes, tick); err != nil {
			tick <- err
			return
		}
		if err := io.SaveThumbs(m.thumbnails, tick); err != nil {
			tick <- err
			return
		}
		if err := io.SaveConfig(*m.Config); err != nil {
			tick <- err
			return
		}
		tick <- true
	}()

	for i := range tick {
		switch msg := i.(type) {
		case error:
			return errMsg{msg, true}
		default:
			ctr++
			*m.percent = ctr / total
		}
	}

	return tea.QuitMsg{}
}

// playfix turns the most significant bit in the played time integer & sets them to 0.
// This fixes a known bug in the library via the assumption that nobody has played 4660+ hours of something.
func (m Model) playfix() tea.Msg {
	ctr := 0.0
	for k, v := range m.playTimes {
		p := v.Played &^ 0xFF000000
		v.Played = p
		m.playTimes[k] = v
		ctr++
		*m.percent = ctr / float64(len(m.playTimes))
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// prune removes all thumbnail entries that don't have a corresponding entry in the library
func (m Model) prune() tea.Msg {
	ctr := 0.0
	total := 0.0
	for _, v := range m.thumbnails {
		total = total + float64(len(v.Images))
	}

	for k, v := range m.thumbnails {
		t := m.thumbnails[k]
		t.Images = slices.DeleteFunc(v.Images, func(image models.Image) bool {
			return !slices.ContainsFunc(m.entries, func(entry models.Entry) bool {
				return entry.System.ThumbFile() == k && entry.Crc32 == image.Crc32
			})
		})
		if len(t.Images) != len(m.thumbnails[k].Images) {
			t.Modified = true
		}
		m.thumbnails[k] = t
		ctr++
		*m.percent = ctr / total
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// genFull generates thumbnail images for all files in the Images/<system>/ directories. It can take a while.
// TODO: Should some of this be moved into the io package? We'd lose the progress bar though
func (m Model) genFull() tea.Msg {
	ctr := 0.0
	total := 0.0
	for _, sys := range models.ValidThumbsFiles {
		de, err := os.ReadDir(fmt.Sprintf("%s/System/Library/Images/%s", m.rootDir, strings.ToLower(sys.String())))
		if os.IsNotExist(err) {
			// Directory doesn't exist. Just continue
			continue
		} else if err != nil {
			return errMsg{err, true}
		}
		for _, e := range de {
			if !e.IsDir() && len(e.Name()) == 12 /* 8 characters + 4 char extension */ && strings.HasSuffix(e.Name(), ".bin") {
				total++ // Only increment this if it's a file we're going to try processing
			}
		}
	}

	for _, sys := range models.ValidThumbsFiles {
		de, err := os.ReadDir(fmt.Sprintf("%s/System/Library/Images/%s", m.rootDir, strings.ToLower(sys.String())))
		if os.IsNotExist(err) {
			// Directory doesn't exist. Just continue
			continue
		} else if err != nil {
			return errMsg{err, true}
		}

		thumbs := models.Thumbnails{Modified: true}
		for _, e := range de {
			if e.IsDir() || len(e.Name()) != 12 || !strings.HasSuffix(e.Name(), ".bin") {
				continue // Definitely not an image file. Continue.
			}

			ctr++
			hash, _, _ := strings.Cut(e.Name(), ".") // Don't need to check found as the above if stmt guarantees its presence
			b, err := hex.DecodeString(hash)
			if err != nil {
				// Not a valid file name. Skip
				continue
			}
			i, err := models.GenerateThumbnail(m.rootDir, sys, binary.BigEndian.Uint32(b))
			if err != nil { // This one is based off of existing files, so don't check for os.ErrNotExist
				return errMsg{err, true}
			}

			thumbs.Images = append(thumbs.Images, i)
			*m.percent = ctr / total
		}
		m.thumbnails[sys] = thumbs
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// genMissing generates thumbnails for only those games in the user's library that don't have entries.
// genMissing and regenLib are the same except for the slices.ContainsFunc call. Can we do something about that?
func (m Model) genMissing() tea.Msg {
	ctr := 0.0
	for _, e := range m.entries {
		sys := e.System.ThumbFile()

		if !slices.ContainsFunc(m.thumbnails[sys].Images, func(image models.Image) bool {
			return image.Crc32 == e.Crc32
		}) {
			img, err := models.GenerateThumbnail(m.rootDir, sys, e.Crc32) // TODO: move this func elsewhere?
			if err != nil && !os.IsNotExist(err) {                        // We only care if it was something other than a not existing error
				return errMsg{err, true}
			} else {
				i := m.thumbnails[sys]
				i.Images = append(i.Images, img)
				i.Modified = true
				m.thumbnails[sys] = i
			}
		}
		ctr++
		*m.percent = ctr / float64(len(m.entries))
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// regenLib generates new thumbnails for all games in the user's library
func (m Model) regenLib() tea.Msg {
	ctr := 0.0
	for _, e := range m.entries {
		sys := e.System.ThumbFile()

		img, err := models.GenerateThumbnail(m.rootDir, sys, e.Crc32) // TODO: move this func elsewhere?
		if err != nil && !os.IsNotExist(err) {                        // We only care if it was something other than a not existing error
			return errMsg{err, true}
		} else {
			i := m.thumbnails[sys]
			i.Images = append(i.Images, img)
			i.Modified = true
			m.thumbnails[sys] = i
		}
		ctr++
		*m.percent = ctr / float64(len(m.entries))
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// genSingle generates a single thumbnail entry & then either updates or inserts it into the list of thumbnails
func (m Model) genSingle(e models.Entry) tea.Cmd {
	return func() tea.Msg {
		*m.percent = 0.0
		sys := e.System.ThumbFile()
		img, err := models.GenerateThumbnail(m.rootDir, sys, e.Crc32)
		*m.percent = .50        // These percentages are just made up.
		if os.IsNotExist(err) { // Doesn't exist. That's fine.
			*m.percent = 1.0
			m.updates <- m
			return updateMsg{}
		} else if err != nil {
			return errMsg{err, true}
		}
		t := m.thumbnails[sys]
		for i, c := range t.Images {
			if c.Crc32 == e.Crc32 {
				t.Images[i] = img
				t.Modified = true
			}
		}
		*m.percent = .75
		if !t.Modified { // We didn't find the image for that game
			t.Images = append(t.Images, img)
			t.Modified = true
		}

		m.thumbnails[sys] = t

		*m.percent = 1.0
		m.updates <- m

		return updateMsg{}
	}
}

func (m Model) editScreenView() string {
	s := fmt.Sprintf("%s\n\n", m.gameInput[name].View())
	if m.AdvancedEditing {
		s = fmt.Sprintf("%s%s\n\n", s, m.gameInput[system].View())
	}
	s = fmt.Sprintf("%s%s\n\n", s, m.gameInput[crc].View())
	if m.AdvancedEditing {
		s = fmt.Sprintf("%s%s\n\n%s\n\n", s, m.gameInput[sig].View(), m.gameInput[magic].View())
	}
	s = fmt.Sprintf("%s%s\n\n%s\n\n", s, m.gameInput[added].View(), m.gameInput[play].View())

	return s
}

func (m Model) addScreenView() string {
	s := fmt.Sprintf("  %s\n\n", m.gameInput[name].View())
	s = fmt.Sprintf("%s  %s\n\n", s, m.gameInput[system].View())
	s = fmt.Sprintf("%s  %s\n\n", s, m.gameInput[crc].View())
	s = fmt.Sprintf("%s  %s\n\n  %s\n\n", s, m.gameInput[sig].View(), m.gameInput[magic].View())
	s = fmt.Sprintf("%s  %s\n\n  %s\n\n", s, m.gameInput[added].View(), m.gameInput[play].View())

	return s
}

// removeEntry removes an entry from the library. If RemoveImages is set to true, it also removes any thumbnails
// associated with the system + CRC combination. idx is the element's index in the entries slice; this has to be used, rather
// than CRC or sig, as it's possible with editting/adding to have duplicates.
//
// Unlike many of the other operations, this one should not be performed in a separate tea.Cmd, as we don't want to display
// the list with the incorrect items.
func (m Model) removeEntry(idx int) Model {
	rm := m.entries[idx]
	m.entries = slices.Delete(m.entries, idx, idx+1)

	if !m.RemoveImages { // If they don't have this flagged, leave the thumbnails alone
		return m
	}

	// Delete the thumbnail entry if it exists
	sys := rm.System.ThumbFile()
	t := m.thumbnails[sys]
	for j, img := range t.Images {
		if rm.Crc32 == img.Crc32 {
			t.Images = slices.Delete(t.Images, j, j+1)
			t.Modified = true
		}
	}
	m.thumbnails[sys] = t

	return m
}

func (m Model) menuHandler(msg tea.Msg) (tea.Model, tea.Cmd) {
	scr := m.Peek()
	switch scr {
	case MainMenu, LibraryMenu, ThumbMenu, ConfigMenu: // Menus without filtering
		if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "enter" || k.String() == " ") {
			return enter[scr](m, msg)
		} else if ok && k.String() == "esc" {
			return esc[scr](m, msg)
		}
	case EditList, GenerateList, RemoveList: // Menus with filtering
		if !m.gameList.SettingFilter() {
			if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "enter" || k.String() == " ") {
				return enter[scr](m, msg)
			} else if ok && k.String() == "esc" {
				return esc[scr](m, msg)
			}
		}
	}

	// ok should be never false, but check just to make certain
	if fn, ok := def[scr]; ok {
		return fn(m, msg)
	}

	return m, nil
}

func (m Model) inputHandler(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "tab", "down":
			return m.shiftInput(1)
		case "shift+tab", "up":
			return m.shiftInput(-1)
		case "enter":
			switch m.focusedInput {
			case submit:
				// TODO: Handle submitting
			case cancel:
				m.Pop()
			default:
				return m.shiftInput(1)
			}
		case "esc":
			m.Pop()
		}
	}

	return m.updateInputs(msg)
}

func (m Model) shiftInput(i int) (tea.Model, tea.Cmd) {
	// TODO: Blur buttons as necessary
	m.gameInput[m.focusedInput].Blur()
	m.gameInput[m.focusedInput].PromptStyle = itemStyle
	m.focusedInput = m.focusedInput + i
	if !m.AdvancedEditing && m.Peek() != AddScreen {
		if i > 0 {
			if m.focusedInput == system {
				m.focusedInput = crc
			} else if m.focusedInput == sig || m.focusedInput == magic {
				m.focusedInput = added
			}
		} else if i < 0 {
			if m.focusedInput == system {
				m.focusedInput = name
			} else if m.focusedInput == sig || m.focusedInput == magic {
				m.focusedInput = crc
			}
		}
	}
	if m.focusedInput >= len(m.gameInput) {
		m.focusedInput = len(m.gameInput) - 1
	} else if m.focusedInput < 0 {
		m.focusedInput = 0
	}
	m.gameInput[m.focusedInput].PromptStyle = selectedItemStyle.PaddingLeft(4)
	return m, m.gameInput[m.focusedInput].Focus()
}

func (m Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.gameInput))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.gameInput {
		m.gameInput[i], cmds[i] = m.gameInput[i].Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) processMenuItem(key menuKey) (Model, tea.Cmd) {
	switch key {
	case lib:
		*m.subMenu = generateSubMenu(*m.subMenu, libraryOptions, "Main > Library", m.mainMenu.Width(), m.mainMenu.Height())
		if !m.ShowAdd {
			m.subMenu.SetItems(m.subMenu.Items()[1:])
		}
		m.Push(LibraryMenu)
	case thumbs:
		*m.subMenu = generateSubMenu(*m.subMenu, thumbOptions, "Main > Thumbnails", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(ThumbMenu)
	case config:
		m.configMenu.ResetSelected()
		m.Push(ConfigMenu)
	case quit:
		return m, tea.Quit
	case save:
		m.Push(Saving)
		return m, tea.Batch(m.save, tickCmd())
	case back:
		return pop(m, nil)
	case add:
		m.focusedInput = 0
		for i := range m.gameInput {
			m.gameInput[i].SetValue("")
			m.gameInput[i].Blur()
			m.gameInput[i].PromptStyle = itemStyle
		}
		// TODO: Remove focus from buttons
		m.gameInput[added].Placeholder = time.Now().Format("2006-01-02 15:04") // Reset the placeholder to whatever
		m.gameInput[m.focusedInput].PromptStyle = selectedItemStyle.PaddingLeft(4)
		m.Push(AddScreen)
		return m, m.gameInput[m.focusedInput].Focus()
	case edit:
		*m.gameList = generateGameList(*m.gameList, m.entries, "Main > Library > Edit Game", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(EditList)
	case rm:
		*m.gameList = generateGameList(*m.gameList, m.entries, "Main > Library > Remove Game", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(RemoveList)
	case fix:
		m.Push(Waiting)
		m.wait = "Fixing played times"
		return m, tea.Batch(m.playfix, tickCmd())
	case missing:
		m.Push(Waiting)
		m.wait = "Generating missing thumbnails for library"
		return m, tea.Batch(m.genMissing, tickCmd())
	case single:
		*m.gameList = generateGameList(*m.gameList, m.entries, "Main > Library > Generate Thumbnail", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(GenerateList)
	case genlib:
		m.Push(Waiting)
		m.wait = "Regenerating all thumbnails for library"
		return m, tea.Batch(m.regenLib, tickCmd())
	case prune:
		m.Push(Waiting)
		m.wait = "Removing orphaned thumbs.bin entries"
		return m, tea.Batch(m.prune, tickCmd())
	case all:
		m.Push(Waiting)
		m.wait = "Generating thumbnails for all games in the Images folder. This may take a while."
		return m, tea.Batch(m.genFull, tickCmd())
	case showAdd, advEdit, rmThumbs:
		return m.configChange(key)
	}

	return m, nil
}

// configMenu handles item selection on the settings menu
func (m Model) configChange(key menuKey) (Model, tea.Cmd) {
	switch key {
	case showAdd:
		m.ShowAdd = !m.ShowAdd
	case rmThumbs:
		m.RemoveImages = !m.RemoveImages
	case advEdit:
		m.AdvancedEditing = !m.AdvancedEditing
	}

	return m, nil
}

func sysValidate(s string) error {
	_, err := models.Parse(s)
	return err
}

func notBlank(s string) error {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		fmt.Errorf("Name cannot be blank")
	}
	return nil
}

func hexValidate(s string) error {
	_, err := util.HexStringTransform(s)
	return err
}

func dateValidate(s string) error {
	// TODO: Make this more robust (e.g. 12 vs 24 hour, leading 0s, etc. Seconds?)
	_, err := time.Parse("2006-01-02 15:04", s)
	return err
}

func playValidate(s string) error {
	// TODO: Should actually be in the form 00h 00m 00s
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	if i < 0 {
		return fmt.Errorf("Playtime cannot be a negative value")
	}

	return nil
}
