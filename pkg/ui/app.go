package ui

import (
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/g026r/pocket-toolkit/pkg/io"
	"github.com/g026r/pocket-toolkit/pkg/models"
	"github.com/g026r/pocket-toolkit/pkg/util"
)

//go:embed version.txt
var version string

type errMsg struct {
	err   error
	fatal bool
}

// TODO: Do I really need initDoneMsg & updateMsg?
// TODO: Could what they're used to trigger be done in their functions instead?
// initDoneMsg signals the initial loading is done.
// it's used to stop the spinner, clear the screen, etc
type initDoneMsg struct {
	badPlaytimes bool
}

// updateMsg tells the program that an update requiring a progress bar is complete
// it's used to trigger the "press any key" text
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
	rootDir      string
	entries      []models.Entry
	playTimes    map[uint32]models.PlayTime
	thumbnails   map[models.System]models.Thumbnails
	internal     map[models.System][]models.Entry // internal is a map of all known possible entries, grouped by system. For eventual use with add & adv. editing, maybe.
	*io.Config                                    // io.Config is a pointer as we need to be able to read this value in the configDelegate, which doesn't have access to Model
	stack                                         // stack contains the stack of screens. Useful for when we go up a screen, as a few have multiple possible parents.
	spinner      spinner.Model                    // spinner is used for calls where we don't know the percentage. Mostly this means the initial loading screen
	progress     progress.Model                   // progress is used for calls where we do know the percentage; has to be a pointer as the screen size event calls before we've finished initializing the Model
	percent      float64                          // the percent of the progress bar; it's a pointer as I'm using it to signal that initialization is done
	err          error                            // err is used to print out an error if the program has to exit early
	wait         string                           // wait is the message to display while waiting
	anyKey       bool                             // anyKey tells View whether we're waiting for a key input or not
	mainMenu     list.Model
	subMenu      list.Model        // subMenu covers the library & thumbnail options: menus where esc goes up a screen but filtering is disabled
	configMenu   list.Model        // configMenu is a special case as it needs a different delegate renderer from subMenu
	gameList     list.Model        // gameList covers anything that lists all the games in the library
	gameInput    []FocusBlurViewer // "Widgets" on the edit & add screens
	focusedInput int               // Which widget has focus on the edit/add screens
	initialized  bool
}

func NewModel() tea.Model {
	var opt progress.Option
	if lipgloss.DefaultRenderer().HasDarkBackground() {
		opt = progress.WithScaledGradient(blue.Light, blue.Dark)
	} else {
		opt = progress.WithScaledGradient(blue.Dark, blue.Light)
	}

	config := io.Config{}

	return &Model{
		stack:      stack{[]screen{Initializing}},
		spinner:    spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(itemStyle.Foreground(blue))),
		progress:   progress.New(opt),
		Config:     &config,
		mainMenu:   *NewMainMenu(),
		configMenu: *NewConfigMenu(&config),
		subMenu:    *NewSubMenu(),  // subMenu needs to be set even without items to avoid a nil pointer with the initial WindowSizeMsg
		gameList:   *NewGameMenu(), // same for gameList
		gameInput:  NewInputs(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.initSystem)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	switch msg := msg.(type) {
	case updateMsg:
		m.anyKey = true
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.anyKey {
			m.anyKey = false
			m.percent = 0.0
			return pop(m, msg)
		}
		if m.Peek() == PlaytimePrompt {
			if strings.ToLower(msg.String()) == "y" {
				var cmd, cmd2 tea.Cmd
				m, cmd = pop(m, msg) // Need to pop first to get the prompt screen out of there
				m, cmd2 = m.processMenuItem(libFix)
				return m, tea.Batch(cmd, cmd2)
			} else if strings.ToLower(msg.String()) == "n" || msg.String() == " " || msg.String() == "enter" {
				return pop(m, msg)
			}
		}
	case spinner.TickMsg:
		if m.initialized {
			break // initialized gets set as the last step of initialization. If it's true, we can stop the spinner.
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tickMsg:
		if m.percent < 1.0 {
			return m, tickCmd()
		}
		return m, m.progress.SetPercent(1.0)
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8
		m.mainMenu.SetHeight(msg.Height - 1)
		m.mainMenu.SetWidth(msg.Width - 1)
		m.subMenu.SetHeight(msg.Height - 1)
		m.subMenu.SetWidth(msg.Width - 1)
		m.configMenu.SetHeight(msg.Height - 1)
		m.configMenu.SetWidth(msg.Width - 1)
		m.gameList.SetHeight(msg.Height - 1)
		m.gameList.SetWidth(msg.Width - 1)
		return m, nil
	case list.FilterMatchesMsg:
		// Only m.gameList has filtering enabled, so don't need to check the current screen
		listModel, cmd := m.gameList.Update(msg)
		m.gameList = listModel
		return m, cmd
	case initDoneMsg:
		m.initialized = true
		m.Clear()
		if msg.badPlaytimes {
			m.Push(PlaytimePrompt)
		} else {
			m.Push(MainMenu) // Finished initializing. Replace the stack with a new one containing only the main menu
		}
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

func (m *Model) View() (s string) {
	switch m.Peek() {
	case Initializing:
		s = fmt.Sprintf("%s %s", m.spinner.View(), "Loading your Pocket library. Please wait.")
	case Waiting:
		s = fmt.Sprintf("\n%s\n\n%s", itemStyle.Render(m.wait), itemStyle.Render(m.progress.ViewAs(m.percent)))
	case Saving:
		s = fmt.Sprintf("\n  Saving your Pocket library\n\n  %s", m.progress.ViewAs(m.percent))
	case FatalError:
		s = errorStyle.Render(fmt.Sprintf("FATAL ERROR: %v\n", m.err))
	case MainMenu:
		s = m.mainMenu.View()
	case LibraryMenu, ThumbMenu:
		s = m.subMenu.View()
	case ConfigMenu:
		s = m.configMenu.View()
	case RemoveList, EditList, GenerateList:
		s = m.gameList.View()
	case EditScreen:
		s = m.inputView(fmt.Sprintf("Edit Game > %s", m.gameList.SelectedItem().(models.Entry).Name))
	case AddScreen:
		s = m.inputView("Add Game")
	case AboutScreen:
		s = aboutView()
	case PlaytimePrompt:
		s = playTimePrompt()
	default:
		panic("Panic! At the View() call")
	}

	if m.anyKey {
		s = fmt.Sprintf("%s\n\n%s", s, itemStyle.Render("Press any key to continue."))
	}

	return
}

func playTimePrompt() string {
	s := fmt.Sprintf("  %s", titleStyle.Render("Unofficial Analogue Pocket library toolkit"))

	s = fmt.Sprintf("%s\n\n%s", s, itemStyle.Render("Potentially invalid played times (> 4096 hours) detected. Do you wish to adjust them (y/N)?"))
	s = fmt.Sprintf("%s\n\n%s", s, subduedStyle.Render("You can disable this automated check in the settings."))

	return s
}

func aboutView() string {
	s := fmt.Sprintf("  %s", titleStyle.Render("Unofficial Analogue Pocket library toolkit"))

	v := strings.Split(version, ",")
	s = fmt.Sprintf("%s\n\n%s", s, itemStyle.Render("Version:", v[0]))
	s = fmt.Sprintf("%s\n%s", s, itemStyle.Render("Built:", v[1]))
	s = fmt.Sprintf("%s\n\n%s", s, itemStyle.Render("Bug reports, feature requests, & new versions:"))
	s = fmt.Sprintf("%s\n%s", s, selectedItemStyle.PaddingLeft(6).Render("https://github.com/g026r/pocket-library-toolkit"))

	return s
}

// initSystem loads all our data from disk
func (m *Model) initSystem() tea.Msg {
	d, err := util.GetRoot()
	if err != nil {
		return errMsg{err, true}
	}
	m.rootDir = d
	rootFs := os.DirFS(d)

	c, err := io.LoadConfig()
	if err != nil {
		return errMsg{err, true}
	}
	*m.Config = c

	e, err := io.LoadEntries(rootFs)
	if err != nil {
		return errMsg{err, true}
	}
	m.entries = e

	p, err := io.LoadPlaytimes(rootFs)
	if err != nil {
		return errMsg{err, true}
	}
	m.playTimes = p

	if len(m.entries) < len(m.playTimes) { // We can have more entries than play times, but not the other way around.
		return errMsg{fmt.Errorf("entry count mismatch between list.bin [%d] & playtimes.bin [%d]", len(m.entries), len(m.playTimes)), true}
	}
	for _, e := range m.entries { // Check to make certain that every entry has a play time entry
		if _, ok := m.playTimes[e.Sig]; !ok { // entry without a play time
			return errMsg{fmt.Errorf("entry mismatch between list.bin & playtimes.bin"), true}
		}
	}

	t, err := io.LoadThumbs(rootFs)
	if err != nil {
		return errMsg{err, true}
	}
	m.thumbnails = t

	// TODO: Uncomment when we're closer to having a complete dataset & can use this with the add functionality
	//	i, err := io.LoadInternal()
	//	if err != nil {
	//		return errMsg{err, true}
	//	}
	//	m.internal = i

	if m.CheckPlaytimes {
		for _, p := range m.playTimes {
			if p.Played >= 0x01000000 {
				return initDoneMsg{true}
			}
		}
	}

	return initDoneMsg{false}
}

// save is the opposite of init: save our data to disk
func (m *Model) save() tea.Msg {
	var err error

	success := false
	tmpList, err := os.CreateTemp(fmt.Sprintf("%s/System/Played Games", m.rootDir), "list_*.tmp")
	if err != nil {
		return errMsg{err, true}
	}
	tmpPlaytimes, err := os.CreateTemp(fmt.Sprintf("%s/System/Played Games", m.rootDir), "playtimes_*.tmp")
	if err != nil {
		return errMsg{err, true}
	}
	tmpThumbs := make(map[models.System]*os.File)
	for k, v := range m.thumbnails {
		if v.Modified || m.SaveUnmodified {
			tmpThumbs[k], err = os.CreateTemp(fmt.Sprintf("%s/System/Library/Images/", m.rootDir), fmt.Sprintf("%s_thumbs_*.tmp", strings.ToLower(k.String())))
			if err != nil {
				return errMsg{err, true}
			}
		}
	}
	defer func() {
		// An absolute mess of a function that:
		// 1. Closes all the file handles
		// 2. Backs up the old library entries if that's enabled & moves the temporary files over to the correct spot if successful, or
		// 3. Deletes them if we weren't
		// TODO: Clean this up to make it all more manageable

		// Clean everything up
		_ = tmpList.Close()
		_ = tmpPlaytimes.Close()
		for _, v := range tmpThumbs {
			_ = v.Close()
		}
		if success {
			t := time.Now().Format("20060102_150405")
			if err := m.backupAndRename(m.rootDir, "/System/Played Games/list.bin", tmpList, t); err != nil {
				log.Fatal(errorStyle.Render(err.Error()))
			}
			if err := m.backupAndRename(m.rootDir, "/System/Played Games/playtimes.bin", tmpPlaytimes, t); err != nil {
				log.Fatal(errorStyle.Render(err.Error()))
			}
			for k, v := range tmpThumbs {
				if err := m.backupAndRename(m.rootDir, fmt.Sprintf("/System/Library/Images/%s_thumbs.bin", strings.ToLower(k.String())), v, t); err != nil {
					log.Fatal(errorStyle.Render(err.Error()))
				}
			}
		} else {
			// Remove all the files as we weren't successful
			_ = os.Remove(tmpList.Name())
			_ = os.Remove(tmpPlaytimes.Name())
			for _, v := range tmpThumbs {
				_ = os.Remove(v.Name())
			}
		}
		m.percent = 1.0
	}()

	ctr := 0.0
	tick := make(chan any)
	total := float64(len(m.entries))
	for _, v := range m.thumbnails {
		if v.Modified {
			total = total + float64(len(v.Images)) // Only increase the total if they've been modified since we don't write them out otherwise.
		}
	}

	go func() { // Run these in a goroutine to avoid having to pass around the pointer to the progress value as that would require knowing the total as well
		defer close(tick)
		if err := io.SaveLibrary(tmpList, m.entries, tmpPlaytimes, m.playTimes, tick); err != nil {
			tick <- err
			return
		}
		for k, v := range tmpThumbs {
			if err := io.SaveThumbsFile(v, m.thumbnails[k].Images, tick); err != nil {
				tick <- err
				return
			}
		}
		tick <- true
	}()

	for i := range tick {
		switch msg := i.(type) {
		case error:
			return errMsg{msg, true}
		default:
			ctr++
			m.percent = ctr / total
		}
	}

	success = true
	return tea.QuitMsg{}
}

// playfix resets the most significant byte in the played time integer to 0.
// This fixes a known bug in the library via the assumption that nobody has played 4660+ hours of something.
func (m *Model) playfix() tea.Msg {
	ctr := 0.0
	for k, v := range m.playTimes {
		v.Played = v.Played &^ 0xFF000000
		m.playTimes[k] = v
		ctr++
		m.percent = ctr / float64(len(m.playTimes))
	}
	m.percent = 1.0
	return updateMsg{}
}

// prune removes all thumbnail entries that don't have a corresponding entry in the library
func (m *Model) prune() tea.Msg {
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
		m.percent = ctr / total
		m.dedupe(k)
	}
	m.percent = 1.0
	return updateMsg{}
}

// genFull generates thumbnail images for all files in the Images/<system>/ directories. It can take a while.
// TODO: Should some of this be moved into the io package? We'd lose the progress bar though. Though could maybe use a channel to pass messages
func (m *Model) genFull() tea.Msg {
	ctr := 0.0
	total := 0.0
	// Calculate what our total percentage is so we can show the progress bar
	for _, sys := range models.ValidThumbsFiles {
		de, err := os.ReadDir(fmt.Sprintf("%s/System/Library/Images/%s", m.rootDir, strings.ToLower(sys.String())))
		if errors.Is(err, fs.ErrNotExist) {
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
		if errors.Is(err, fs.ErrNotExist) {
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
			i, err := io.GenerateThumbnail(os.DirFS(m.rootDir), sys, binary.BigEndian.Uint32(b))
			if errors.Is(err, io.ErrSixteenBitImage) {
				continue // Can't handle 16-bit images at the moment due to a lack of documentation & examples
			} else if err != nil { // This one is based off of existing files, so don't check for os.ErrNotExist
				return errMsg{err, true}
			}

			thumbs.Images = append(thumbs.Images, i)
			m.percent = ctr / total
		}
		m.thumbnails[sys] = thumbs
	}
	m.percent = 1.0
	return updateMsg{}
}

// genMissing generates thumbnails for only those games in the user's library that don't have entries.
// genMissing and regenLib are the same except for the slices.ContainsFunc call. Can we do something about that?
func (m *Model) genMissing() tea.Msg {
	ctr := 0.0
	for _, e := range m.entries {
		sys := e.System.ThumbFile()

		if !slices.ContainsFunc(m.thumbnails[sys].Images, func(image models.Image) bool {
			return image.Crc32 == e.Crc32
		}) {
			img, err := io.GenerateThumbnail(os.DirFS(m.rootDir), sys, e.Crc32)
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, io.ErrSixteenBitImage) {
				continue
			} else if err != nil { // We only care if it was something other than a not existing error
				return errMsg{err, true}
			} else {
				i := m.thumbnails[sys]
				i.Images = append(i.Images, img)
				i.Modified = true
				m.thumbnails[sys] = i
			}
		}
		ctr++
		m.percent = ctr / float64(len(m.entries))
	}
	m.percent = 1.0
	return updateMsg{}
}

// regenLib generates new thumbnails for all games in the user's library
func (m *Model) regenLib() tea.Msg {
	ctr := 0.0
	for _, e := range m.entries {
		sys := e.System.ThumbFile()

		img, err := io.GenerateThumbnail(os.DirFS(m.rootDir), sys, e.Crc32)
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, io.ErrSixteenBitImage) {
			continue
		} else if err != nil { // We only care if it was something other than a not existing error
			return errMsg{err, true}
		} else {
			i := m.thumbnails[sys]
			i.Images = append(i.Images, img)
			i.Modified = true
			m.thumbnails[sys] = i
		}
		ctr++
		m.percent = ctr / float64(len(m.entries))
	}

	for _, sys := range models.ValidThumbsFiles {
		m.dedupe(sys)
	}
	m.percent = 1.0
	return updateMsg{}
}

// genSingle generates a Single thumbnail entry & then either updates or inserts it into the list of thumbnails
func (m *Model) genSingle(e models.Entry) tea.Cmd {
	return func() tea.Msg {
		m.percent = 0.0
		sys := e.System.ThumbFile()
		img, err := io.GenerateThumbnail(os.DirFS(m.rootDir), sys, e.Crc32)
		m.percent = .50                                                              // These percentages are just made up.
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, io.ErrSixteenBitImage) { // Doesn't exist. That's fine.
			m.percent = 1.0
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
		m.percent = .75
		if !t.Modified { // We didn't find the image for that game
			t.Images = append(t.Images, img)
			t.Modified = true
		}

		m.thumbnails[sys] = t

		m.percent = 1.0
		return updateMsg{}
	}
}

// dedupe iterates through the thumbnails for a given System & removes any duplicates it finds,
// keeping only the first image in the list.
// It's not run on save, as the images can get quite large. Instead, it's only run after prune and regenLib are called.
func (m *Model) dedupe(sys models.System) {
	found := map[uint32]bool{}
	remove := make([]int, 0)
	thumbs := m.thumbnails[sys]
	for i := range thumbs.Images {
		img := thumbs.Images[i]
		if _, ok := found[img.Crc32]; ok {
			remove = append(remove, i)
		}
		found[img.Crc32] = true
	}
	for i := len(remove) - 1; i >= 0; i-- { // Go backwards through the list, removing duplicates
		thumbs.Images = slices.Delete(thumbs.Images, i, i+1)
		thumbs.Modified = true
	}
	m.thumbnails[sys] = thumbs
}

func (m *Model) inputView(title string) string {
	s := fmt.Sprintf("  %s", titleStyle.Render(fmt.Sprintf("Main > Library > %s", title)))
	s = fmt.Sprintf("%s\n\n%s\n%s\n", s, m.gameInput[name].View(), errorStyle.Render(m.gameInput[name].error()))
	if m.AdvancedEditing || m.Peek() == AddScreen {
		s = fmt.Sprintf("%s%s\n%s\n", s, m.gameInput[system].View(), errorStyle.Render(m.gameInput[system].error()))
	}
	s = fmt.Sprintf("%s%s\n%s\n", s, m.gameInput[crc].View(), errorStyle.Render(m.gameInput[crc].error()))
	if m.AdvancedEditing || m.Peek() == AddScreen {
		s = fmt.Sprintf("%s%s\n%s\n", s, m.gameInput[sig].View(), errorStyle.Render(m.gameInput[sig].error()))
		s = fmt.Sprintf("%s%s\n%s\n", s, m.gameInput[magic].View(), errorStyle.Render(m.gameInput[magic].error()))
	}
	s = fmt.Sprintf("%s%s\n%s\n", s, m.gameInput[added].View(), errorStyle.Render(m.gameInput[added].error()))
	s = fmt.Sprintf("%s%s\n%s\n", s, m.gameInput[play].View(), errorStyle.Render(m.gameInput[play].error()))
	s = fmt.Sprintf("%s%s%s", s, m.gameInput[submit].View(), m.gameInput[cancel].View())
	return s
}

// removeEntry removes an entry from the library. If RemoveImages is set to true, it also removes any thumbnails
// associated with the system + CRC combination. idx is the element's index in the entries slice; this has to be used, rather
// than CRC or sig, as it's possible with editting/adding to have duplicates.
//
// Unlike many of the other operations, this one should not be performed in a separate tea.Cmd, as we don't want to display
// the list with the incorrect items.
func (m *Model) removeEntry(idx int) *Model {
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

func (m *Model) menuHandler(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			} else if ok && k.String() == "esc" && !m.gameList.IsFiltered() { // Only exit if we're not filtering. Otherwise, let the default actions reset the filter
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

func (m *Model) inputHandler(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "tab", "down":
			return m.shiftInput(1)
		case "shift+tab", "up":
			return m.shiftInput(-1)
		case "left":
			if m.focusedInput == cancel {
				return m.shiftInput(-1)
			}
		case "right":
			if m.focusedInput == submit {
				return m.shiftInput(1)
			}
		case "enter":
			switch m.focusedInput {
			case submit:
				return m.saveEntry()
			case cancel:
				return pop(m, msg)
			default:
				return m.shiftInput(1)
			}
		case " ": // Space only submits on the buttons
			switch m.focusedInput {
			case submit:
				return m.saveEntry()
			case cancel:
				return pop(m, msg)
			}
		case "esc":
			return pop(m, msg)
		}
	}

	return m.updateInputs(msg)
}

func (m *Model) shiftInput(i int) (tea.Model, tea.Cmd) {
	m.gameInput[m.focusedInput].Blur()
	m.gameInput[m.focusedInput].Style(itemStyle)
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
	m.gameInput[m.focusedInput].Style(focusedStyle)

	if t, ok := m.gameInput[m.focusedInput].(*Input); ok {
		t.SetCursor(len(t.Value()))
	}

	return m, m.gameInput[m.focusedInput].Focus()
}

func (m *Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.gameInput))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range len(m.gameInput) {
		u, c := m.gameInput[i].Update(msg)
		m.gameInput[i] = u
		cmds[i] = c
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) saveEntry() (tea.Model, tea.Cmd) {
	foundError := -1
	for i := range play + 1 { // Check all the ones that are inputs.
		if t, ok := m.gameInput[i].(*Input); ok {
			var err error
			if err = t.Validate(t.Value()); err == nil {
				continue
			}
			// Only set the cursor back to the first error found
			if foundError == -1 {
				m.gameInput[i].Style(focusedStyle)
				m.gameInput[m.focusedInput].Blur()
				m.gameInput[m.focusedInput].Style(itemStyle)
				m.focusedInput = i
				t.SetCursor(len(t.Value()))
				foundError = i
			}
			// But set all the error statuses
			t.Err = err
		}
	}
	if foundError != -1 {
		return m, m.gameInput[foundError].Focus()
	}

	e := models.Entry{}
	if m.Peek() == EditScreen {
		e = m.gameList.SelectedItem().(models.Entry)
	}
	e.Name = m.gameInput[name].Value()
	sys := strings.TrimSpace(m.gameInput[system].Value())
	if sys == "" {
		sys = models.GB.String()
	}
	e.System, _ = models.Parse(sys)
	e.Crc32, _ = util.HexStringTransform(m.gameInput[crc].Value())
	e.Sig, _ = util.HexStringTransform(m.gameInput[sig].Value())
	e.Magic, _ = util.HexStringTransform(m.gameInput[magic].Value())

	// We only want to edit the played time *if* we're editing or we're adding & a played time doesn't yet exist
	if p, ok := m.playTimes[e.Sig]; !ok || m.Peek() == EditScreen {
		t, _ := parseDate(m.gameInput[added].Value())
		p.Added = uint32(t.Unix())
		p.Played = parsePlayTime(m.gameInput[play].Value())
		p.System = e.System
		m.playTimes[e.Sig] = p
	}

	if m.Peek() == EditScreen {
		if m.gameList.IsFiltered() {
			// Need to do this as Index() returns position based on the filtered list of items, despite what the godoc says
			// See: https://github.com/charmbracelet/bubbles/issues/550
			selected := m.gameList.SelectedItem().(models.Entry)
			for i := range m.entries {
				if models.EntrySort(m.entries[i], selected) == 0 {
					m.entries[i] = e
					break
				}
			}
		} else {
			// Don't use SetItem on gameList as we're going to resort the items
			// So we'll have to reset the value of Items anyway
			m.entries[m.gameList.Index()] = e
		}
	} else {
		m.entries = append(m.entries, e)
	}

	slices.SortFunc(m.entries, models.EntrySort)
	var cmd tea.Cmd
	if m.Peek() == EditScreen {
		// Only reset the items rather than re-initialize the whole list so that we can keep our position in the list + the active filter
		tmp := make([]list.Item, len(m.entries))
		for i := range m.entries {
			tmp[i] = m.entries[i]
		}
		cmd = m.gameList.SetItems(tmp) // TODO: This is really kind of ugly. Can we make it neater?
	}

	if m.GenerateNew {
		// TODO: Figure out a way to clean up any orphaned ones as well
		m.percent = 0.0
		m.wait = fmt.Sprintf("Generating thumbnail for %s (%s)", e.Name, e.System)
		m.Replace(Waiting)
		return m, tea.Sequence(cmd, tea.Batch(m.genSingle(e), tickCmd())) // Probably doesn't need to be done in sequence tbh
	}

	m, c := pop(m, nil) // c is always nil here, but capture it just in case I ever change that
	return m, tea.Sequence(cmd, c)
}

func (m *Model) processMenuItem(key menuKey) (*Model, tea.Cmd) {
	switch key {
	case lib:
		m.subMenu = generateSubMenu(m.subMenu, libraryOptions, "Main > Library", m.mainMenu.Width(), m.mainMenu.Height())
		if !m.ShowAdd {
			m.subMenu.SetItems(m.subMenu.Items()[1:])
		}
		m.Push(LibraryMenu)
	case thumbs:
		m.subMenu = generateSubMenu(m.subMenu, thumbOptions, "Main > Thumbnails", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(ThumbMenu)
	case config:
		m.configMenu.ResetSelected()
		m.Push(ConfigMenu)
	case about:
		m.Push(AboutScreen)
		m.anyKey = true
	case quit:
		return m, tea.Quit
	case save:
		m.Push(Saving)
		m.percent = 0.0
		return m, tea.Batch(m.save, tickCmd())
	case back:
		if m.Peek() == ConfigMenu {
			if err := m.SaveConfig(); err != nil {
				return m, func() tea.Msg { return errMsg{err, true} }
			}
		}
		return pop(m, nil)
	case libAdd:
		m.focusedInput = 0
		for i := range len(m.gameInput) {
			m.gameInput[i].Style(itemStyle)
			m.gameInput[i].Reset()
			m.gameInput[i].Blur()
			if i != submit && i != cancel {
				// Weirdly Reset() doesn't reset the value of Err
				// Set it directly rather than using SetValue to reset it as "" is a validation error for name
				m.gameInput[i].(*Input).Err = nil
			}
		}

		m.gameInput[added].(*Input).Placeholder = time.Now().Format("2006-01-02 15:04") // Reset the placeholder to whatever
		m.gameInput[m.focusedInput].Style(focusedStyle)
		m.Push(AddScreen)
		return m, m.gameInput[m.focusedInput].Focus()
	case libEdit:
		m.gameList = generateGameList(m.gameList, m.entries, "Main > Library > Edit Game", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(EditList)
	case libRm:
		m.gameList = generateGameList(m.gameList, m.entries, "Main > Library > Remove Game", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(RemoveList)
	case libFix:
		m.percent = 0.0
		m.wait = "Fixing played times"
		m.Push(Waiting)
		return m, tea.Batch(m.playfix, tickCmd())
	case tmMissing:
		m.percent = 0.0
		m.wait = "Generating missing thumbnails for library"
		m.Push(Waiting)
		return m, tea.Batch(m.genMissing, tickCmd())
	case tmSingle:
		m.gameList = generateGameList(m.gameList, m.entries, "Main > Library > Generate Thumbnail", m.mainMenu.Width(), m.mainMenu.Height())
		m.Push(GenerateList)
	case tmGenLib:
		m.percent = 0.0
		m.wait = "Regenerating all thumbnails for library"
		m.Push(Waiting)
		return m, tea.Batch(m.regenLib, tickCmd())
	case tmPrune:
		m.percent = 0.0
		m.wait = "Removing orphaned thumbs.bin entries"
		m.Push(Waiting)
		return m, tea.Batch(m.prune, tickCmd())
	case tmAll:
		m.percent = 0.0
		m.wait = "Generating thumbnails for all games in the Images folder. This may take a while."
		m.Push(Waiting)
		return m, tea.Batch(m.genFull, tickCmd())
	case cfgShowAdd, cfgAdvEdit, cfgRmThumbs, cfgGenNew, cfgUnmodified, cfgBackup, cfgPlaytimeCheck:
		return m.configChange(key)
	}

	return m, nil
}

// configMenu handles item selection on the settings menu
func (m *Model) configChange(key menuKey) (*Model, tea.Cmd) {
	switch key {
	case cfgShowAdd:
		m.ShowAdd = !m.ShowAdd
	case cfgRmThumbs:
		m.RemoveImages = !m.RemoveImages
	case cfgAdvEdit:
		m.AdvancedEditing = !m.AdvancedEditing
	case cfgGenNew:
		m.GenerateNew = !m.GenerateNew
	case cfgUnmodified:
		m.SaveUnmodified = !m.SaveUnmodified
	case cfgBackup:
		m.Backup = !m.Backup
	case cfgPlaytimeCheck:
		m.CheckPlaytimes = !m.CheckPlaytimes
	}

	return m, nil
}

func (m *Model) backupAndRename(root string, path string, tempFile *os.File, time string) error {
	file := fmt.Sprintf("%s%s", root, path)
	if m.Backup {
		if err := os.Rename(file, fmt.Sprintf("%s_%s.bak", strings.TrimSuffix(file, ".bin"), time)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return os.Rename(tempFile.Name(), file)
}

// pop is the ESC action for basically everything but main menu
// It removes the latest item from the stack, allowing the rendering to go up one level, and runs a GC just because.
// It's doable largely because we never have a submenu on top of another submenu
func pop(m *Model, _ tea.Msg) (*Model, tea.Cmd) {
	m.Pop()
	runtime.GC() // Not ideal. Probably also not necessary.
	return m, nil
}
