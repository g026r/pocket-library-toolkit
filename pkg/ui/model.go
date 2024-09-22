package ui

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/g026r/pocket-library-editor/pkg/io"
	model2 "github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

type errMsg struct {
	err   error
	fatal bool
}

type initDoneMsg struct{}

type updateMsg struct{}

type tickMsg time.Time

type model struct {
	updates   chan model // updates is used for passing updates to the model rather than using pointers
	RootDir   fs.FS
	Entries   []model2.Entry
	PlayTimes map[uint32]model2.PlayTime
	Thumbs    map[util.System]model2.Thumbnails
	io.Config
	Internal     map[util.System][]model2.Entry // Internal is a map of all known possible entries, grouped by system. For eventual use with add, maybe.
	*stack                                      // stack contains the stack of screens. Useful for when we go up a screen, as a few have multiple possible parents.
	spinner      spinner.Model                  // spinner is used for calls where we don't know the percentage. Mostly this means the initial loading screen
	progress     *progress.Model                // progress is used for calls where we do know the percentage; has to be a pointer as the screen size event calls before we've finished initializing the model
	percent      *float64                       // the percent of the progress bar
	err          error                          // err is used to print out an error if the program has to exit early
	wait         string                         // wait is the message to display while waiting
	anyKey       bool                           // anyKey tells View whether we're waiting for a key input or not
	page         int                            // page stores the page of game entries we are currently on
	mainMenu     *list.Model
	libMenu      *list.Model
	thumbMenu    *list.Model
	configMenu   *list.Model
	removeList   *list.Model
	generateList *list.Model
	editList     *list.Model
}

func NewModel() tea.Model {
	prog := progress.New(progress.WithScaledGradient("#006699", "#00ccff"), progress.WithWidth(100))
	return model{
		updates:      make(chan model, 1),
		stack:        &stack{[]screen{Initializing}},
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		progress:     &prog,
		mainMenu:     NewMainMenu(),
		libMenu:      NewLibraryMenu(),
		thumbMenu:    NewThumbMenu(),
		configMenu:   NewConfigMenu(),
		editList:     NewGameMenu("Main > Library > Edit Game"),
		removeList:   NewGameMenu("Main > Library > Remove Game"),
		generateList: NewGameMenu("Main > Thumbnails > Regenerate Thumbnail"),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.initSystem)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.libMenu.SetHeight(msg.Height)
		m.libMenu.SetWidth(msg.Width)
		m.thumbMenu.SetHeight(msg.Height)
		m.thumbMenu.SetWidth(msg.Width)
		m.configMenu.SetHeight(msg.Height)
		m.configMenu.SetWidth(msg.Width)
		m.generateList.SetHeight(msg.Height)
		m.generateList.SetWidth(msg.Width)
		m.removeList.SetHeight(msg.Height)
		m.removeList.SetWidth(msg.Width)
		m.editList.SetHeight(msg.Height)
		m.editList.SetWidth(msg.Width)
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

	return menuHandler(m, msg)
}

func processMenuItem(m model, key menuKey) (model, tea.Cmd) {
	switch key {
	case lib:
		m.libMenu.ResetSelected()
		if !m.ShowAdd {
			m.libMenu.SetItems(libraryOptions[1:])
		} else {
			m.libMenu.SetItems(libraryOptions)
		}
		m.Push(LibraryMenu)
	case thumbs:
		m.thumbMenu.ResetSelected()
		m.Push(ThumbMenu)
	case config:
		m.configMenu.ResetSelected()
		//m.configMenu.SetItems(m.generateConfigOptions())
		m.Push(ConfigMenu)
	case quit:
		return m, tea.Quit
	case save:
		m.Push(Saving)
		return m, tea.Batch(m.save, tickCmd())
	case back:
		m.Pop()
	case add:
		// TODO: Add menu?
	case edit:
		m.editList.ResetSelected() // TODO: Reset the actual menu items
		m.editList.ResetFilter()
		m.editList.SetItems(m.generateGameListView())
		m.Push(EditList)
	case rm:
		m.removeList.ResetSelected() // TODO: Reset the actual menu items
		m.removeList.ResetFilter()
		m.removeList.SetItems(m.generateGameListView())
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
		m.generateList.ResetSelected() // TODO: Reset the actual menu items
		m.generateList.ResetFilter()
		m.generateList.SetItems(m.generateGameListView())
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
		return configMenu(m, key)
	}

	return m, nil
}

func (m model) View() (s string) {
	switch m.Peek() {
	case Initializing:
		s = fmt.Sprintf(" %s Loading your library. Please wait.", m.spinner.View())
		//s = m.menu.View()
	case Waiting:
		s = fmt.Sprintf("%s\n\n%s", m.wait, m.progress.ViewAs(*m.percent))
		if m.anyKey {
			s = fmt.Sprintf("%s\n\nPress any key to continue.", s)
		}
	case Saving:
		s = fmt.Sprintf("Saving your Pocket library\n\n%s", m.progress.ViewAs(*m.percent))
	case FatalError:
		s = fmt.Sprintf("FATAL ERROR: %v\n", m.err)
	case MainMenu:
		//s = menuView(m, "Welcome to the Analogue Pocket library editor", mainMenuOptions)
		s = m.mainMenu.View()
	case LibraryMenu:
		opt := libraryOptions // TODO: Show do this when we swap screens. Not here.
		if !m.ShowAdd {
			opt = libraryOptions[1:]
		}
		m.libMenu.SetItems(opt)
		s = m.libMenu.View()
		//s = menuView(m, "Main > Library", opt)
	case ThumbMenu:
		//s = menuView(m, "Main > Thumbnails", thumbOptions)
		s = m.thumbMenu.View()
	case ConfigMenu:
		//s = settingsView(m, "Main > Settings")
		m.configMenu.SetItems(m.generateConfigView())
		s = m.configMenu.View()
	case RemoveList:
		s = m.removeList.View()
	case EditList:
		s = m.editList.View()
	case GenerateList:
		s = m.generateList.View()
	case AddScreen, EditScreen:
		fallthrough
	default:
		panic("Panic! At the switch statement")
	}

	return
}

func (m model) generateGameListView() []list.Item {
	items := make([]list.Item, 0)
	for _, e := range m.Entries {
		items = append(items, e)
	}

	return items
}

// initSystem loads all our data from disk
func (m model) initSystem() tea.Msg {
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
	m.RootDir = os.DirFS(d)

	c, err := io.LoadConfig()
	if err != nil {
		return errMsg{err, true}
	}
	m.Config = c

	e, err := io.LoadEntries(m.RootDir)
	if err != nil {
		return errMsg{err, true}
	}
	m.Entries = e

	p, err := io.LoadPlaytimes(m.RootDir)
	if err != nil {
		return errMsg{err, true}
	}
	m.PlayTimes = p

	if len(m.Entries) != len(m.PlayTimes) {
		return errMsg{fmt.Errorf("entry count mismatch between list.bin [%d] & playtimes.bin [%d]", len(m.Entries), len(m.PlayTimes)), true}
	}

	t, err := io.LoadThumbs(m.RootDir)
	if err != nil {
		return errMsg{err, true}
	}
	m.Thumbs = t

	if m.ShowAdd { // Only need to load these if we're showing the add option
		i, err := io.LoadInternal()
		if err != nil {
			return errMsg{err, true}
		}
		m.Internal = i
	}

	per := 0.0
	m.percent = &per // Setting this value both prevents nil pointer dereferences & is used as the signal to stop the spinner

	m.updates <- m
	return initDoneMsg{}
}

// save is the opposite of init: save our data to disk
func (m model) save() tea.Msg {
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
	total := float64(len(m.Entries))
	for _, v := range m.Thumbs {
		if v.Modified {
			total = total + float64(len(v.Images)) // Only increase the total if they've been modified since we don't write them out otherwise.
		}
	}
	total = total + 1 // Add 1 for the config

	go func() { // Run these in a goroutine to avoid having to pass around the pointer to the progress value as that would require knowing the total as well
		defer close(tick)
		if err := io.SaveLibrary(m.Entries, m.PlayTimes, tick); err != nil {
			tick <- err
			return
		}
		if err := io.SaveThumbs(m.Thumbs, tick); err != nil {
			tick <- err
			return
		}
		if err := io.SaveConfig(m.Config); err != nil {
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
func (m model) playfix() tea.Msg {
	ctr := 0.0
	for k, v := range m.PlayTimes {
		p := v.Played &^ 0xFF000000
		v.Played = p
		m.PlayTimes[k] = v
		ctr++
		*m.percent = ctr / float64(len(m.PlayTimes))
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

func (m model) prune() tea.Msg {
	ctr := 0.0
	total := 0.0
	for _, v := range m.Thumbs {
		total = total + float64(len(v.Images))
	}

	for k, v := range m.Thumbs {
		t := m.Thumbs[k]
		t.Images = slices.DeleteFunc(v.Images, func(image model2.Image) bool {
			return !slices.ContainsFunc(m.Entries, func(entry model2.Entry) bool {
				return entry.System.ThumbFile() == k && entry.Crc32 == image.Crc32
			})
		})
		if len(t.Images) != len(m.Thumbs[k].Images) {
			t.Modified = true
		}
		m.Thumbs[k] = t
		ctr++
		*m.percent = ctr / total
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// genFull generates thumbnail images for all files in the Images/<system>/ directories. It can take a while.
// TODO: Should some of this be moved into the io package?
func (m model) genFull() tea.Msg {
	ctr := 0.0
	total := 0.0
	for _, sys := range util.ValidThumbsFiles {
		de, err := os.ReadDir(fmt.Sprintf("%s/System/Library/Images/%s", m.RootDir, strings.ToLower(sys.String())))
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

	for _, sys := range util.ValidThumbsFiles {
		de, err := os.ReadDir(fmt.Sprintf("%s/System/Library/Images/%s", m.RootDir, strings.ToLower(sys.String())))
		if os.IsNotExist(err) {
			// Directory doesn't exist. Just continue
			continue
		} else if err != nil {
			return errMsg{err, true}
		}

		thumbs := model2.Thumbnails{Modified: true}
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
			i, err := model2.GenerateThumbnail(m.RootDir, sys, binary.BigEndian.Uint32(b))
			if err != nil { // This one is based off of existing files, so don't check for os.ErrNotExist
				return errMsg{err, true}
			}

			thumbs.Images = append(thumbs.Images, i)
			*m.percent = ctr / total
		}
		m.Thumbs[sys] = thumbs
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// genMissing generates thumbnails for only those games in the user's library that don't have entries.
// genMissing and regenLib are the same except for the slices.ContainsFunc call. Can we do something about that?
func (m model) genMissing() tea.Msg {
	ctr := 0.0
	for _, e := range m.Entries {
		sys := e.System.ThumbFile()

		if !slices.ContainsFunc(m.Thumbs[sys].Images, func(image model2.Image) bool {
			return image.Crc32 == e.Crc32
		}) {
			img, err := model2.GenerateThumbnail(m.RootDir, sys, e.Crc32) // TODO: move this func elsewhere?
			if err != nil && !os.IsNotExist(err) {                        // We only care if it was something other than a not existing error
				return errMsg{err, true}
			} else {
				i := m.Thumbs[sys]
				i.Images = append(i.Images, img)
				i.Modified = true
				m.Thumbs[sys] = i
			}
		}
		ctr++
		*m.percent = ctr / float64(len(m.Entries))
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// regenLib generates new thumbnails for all games in the user's library
func (m model) regenLib() tea.Msg {
	ctr := 0.0
	for _, e := range m.Entries {
		sys := e.System.ThumbFile()

		img, err := model2.GenerateThumbnail(m.RootDir, sys, e.Crc32) // TODO: move this func elsewhere?
		if err != nil && !os.IsNotExist(err) {                        // We only care if it was something other than a not existing error
			return errMsg{err, true}
		} else {
			i := m.Thumbs[sys]
			i.Images = append(i.Images, img)
			i.Modified = true
			m.Thumbs[sys] = i
		}
		ctr++
		*m.percent = ctr / float64(len(m.Entries))
	}
	*m.percent = 1.0
	m.updates <- m
	return updateMsg{}
}

// genSingle generates a single thumbnail entry & then either updates or inserts it into the list of thumbnails
func (m model) genSingle(e model2.Entry) tea.Cmd {
	return func() tea.Msg {
		*m.percent = 0.0
		img, err := model2.GenerateThumbnail(m.RootDir, e.System, e.Crc32)
		*m.percent = .50 // These percentages are just made up.
		if err != nil && !os.IsNotExist(err) {
			return errMsg{err, true}
		}
		t := m.Thumbs[e.System]
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

		m.Thumbs[e.System] = t

		*m.percent = 1.0
		m.updates <- m

		return updateMsg{}
	}
}

// configMenu handles item selection on the settings menu
func configMenu(m model, key menuKey) (model, tea.Cmd) {
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

func (m model) generateConfigView() []list.Item {
	configOptions := []list.Item{
		menuItem{"[%s] Remove thumbnail when removing game", rmThumbs},
		menuItem{"[%s] Show advanced library editing fields (Experimental)", advEdit},
		menuItem{"[%s] Show add library entry (Experimental)", showAdd},
		menuItem{"Back", back}}
	var item menuItem
	if m.RemoveImages {
		item = configOptions[0].(menuItem)
		item.text = fmt.Sprintf(item.text, "X")
		configOptions[0] = item
	} else {
		item = configOptions[0].(menuItem)
		item.text = fmt.Sprintf(item.text, " ")
		configOptions[0] = item
	}
	if m.AdvancedEditing {
		item = configOptions[1].(menuItem)
		item.text = fmt.Sprintf(item.text, "X")
		configOptions[1] = item
	} else {
		item = configOptions[1].(menuItem)
		item.text = fmt.Sprintf(item.text, " ")
		configOptions[1] = item
	}
	if m.ShowAdd {
		item = configOptions[2].(menuItem)
		item.text = fmt.Sprintf(item.text, "X")
		configOptions[2] = item
	} else {
		item = configOptions[2].(menuItem)
		item.text = fmt.Sprintf(item.text, " ")
		configOptions[2] = item
	}

	return configOptions
}

// removeEntry removes an entry from the library. if Config.RemoveImages is set to true, it also removes any thumbnails
// associated with the system + CRC combination. idx is the element's index in the Entries slice; this has to be used, rather
// than CRC or sig, as it's possible with editting/adding to have duplicates.
//
// Unlike many of the other operations, this one should not be performed in a separate tea.Cmd, as we don't want to display
// the list with the incorrect items.
func (m model) removeEntry(idx int) model {
	rm := m.Entries[idx]
	m.Entries = slices.Delete(m.Entries, idx, idx+1)

	if !m.RemoveImages { // If they don't have this flagged, leave the thumbnails alone
		return m
	}

	// Delete the thumbnail entry if it exists
	sys := rm.System.ThumbFile()
	t := m.Thumbs[sys]
	for j, img := range t.Images {
		if rm.Crc32 == img.Crc32 {
			t.Images = slices.Delete(t.Images, j, j+1)
			t.Modified = true
		}
	}
	m.Thumbs[sys] = t

	return m
}

// tickCmd is used by the progress bar. It fires 1/5 of a second after it's started, and is necessary to call again if
// another tick is desired
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// stack is a simple struct for keeping track of the screen we are currently on & the ones that came before it.
// It's probably overkill. Could have just used a different constant for the edit game, remove game, & regen thumbnail screens.
type stack struct {
	s []screen
}

func (s *stack) Peek() screen {
	if len(s.s) == 0 {
		return MainMenu
	}
	return s.s[len(s.s)-1]
}

func (s *stack) Pop() screen {
	if len(s.s) == 0 {
		return MainMenu
	}
	rm := s.s[len(s.s)-1]
	s.s = s.s[:len(s.s)-1]
	return rm
}

func (s *stack) Push(v screen) {
	s.s = append(s.s, v)
}

func (s *stack) Clear() {
	s.s = make([]screen, 0)
}

// Parent returns the parent of the current screen without modifying the stack.
// There are a few cases where we need to know what the previous screen was, without leaving the current screen.
func (s *stack) Parent() screen {
	if len(s.s) < 2 {
		return MainMenu
	}

	return s.s[len(s.s)-2]
}
