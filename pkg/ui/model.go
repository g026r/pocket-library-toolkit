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

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/g026r/pocket-library-editor/pkg/io"
	model2 "github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

type screen int

const (
	MainMenu    screen = iota
	LibraryMenu screen = iota
	ThumbMenu   screen = iota
	ConfigMenu  screen = iota
	GameList    screen = iota
	AddScreen   screen = iota
	EditScreen  screen = iota
	Saving      screen = iota
	Waiting     screen = iota
	FatalError  screen = iota
)

var (
	selected = lipgloss.NewStyle().Bold(true)

	mainMenuOptions = []string{"Library", "Thumbnails", "Settings", "Save & Quit", "Quit"}
	libraryOptions  = []string{"Add entry", "Edit entry", "Remove entry", "Fix played times", "Back"}
	thumbOptions    = []string{"Generate missing thumbnails", "Regenerate game thumbnail", "Regenerate complete library", "Prune orphaned thumbnails", "Generate complete system thumbnails", "Back"}

	// updates is used for passing changes to the internal objects (library, played times, thumbnails))
	// It could probably have been made simpler — a struct containing only the lists & maps — except we're also using it
	// on the init call, which is when the config & root dir also get updated
	updates = make(chan model, 1)
)

type errMsg struct {
	err   error
	fatal bool
}

type initDoneMsg struct{}

type updateMsg struct{}

type tickMsg time.Time

type model struct {
	RootDir   fs.FS
	Entries   []model2.Entry
	PlayTimes map[uint32]model2.PlayTime
	Thumbs    map[util.System]model2.Thumbnails
	io.Config
	Internal map[util.System][]model2.Entry // Internal is a map of all known possible entries, grouped by system. For eventual use with add, maybe.
	*stack                                  // stack contains the stack of screens. Useful for when we go up a screen, as a few have multiple possible parents.
	spinner  spinner.Model                  // spinner is used for calls where we don't know the percentage. Mostly this means the initial loading screen
	progress progress.Model                 // progress is used for calls where we do know the percentage
	percent  *float64
	err      error  // err is used to print out an error if the program has to exit early
	wait     string // wait is the message to display while waiting
	anyKey   bool   // anyKey tells View whether we're waiting for a key input or not
	pos      int    // pos stores the generic cursor position
	main     int    // main stores the position of the cursor on the main menu for when we go back up to it
	lib      int    // lib stores the position of the cursor on the library menu for when we go back up to it
	thumb    int    // thumb stores the position of the cursor on the thumbnails menu for when we go back up to it
}

func NewModel() tea.Model {
	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	prog.Width = 100
	return model{
		stack:    &stack{make([]screen, 0)},
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		progress: prog,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.initSystem)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	switch msg := msg.(type) {
	case updateMsg:
		m = <-updates
		m.anyKey = true
	case tea.KeyMsg:
		return keyMsg(m, msg)
	case spinner.TickMsg:
		if m.percent != nil {
			break // percent gets set as the last step of initialization. If it's not nil, we can stop the spinner.
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tickMsg:
		if *m.percent < 1.0 {
			return m, tea.Batch(m.progress.SetPercent(*m.percent), tickCmd())
		}
		return m, m.progress.SetPercent(1.0)
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8 // FIXME: This doesn't seem to fire unless I actually resize the window?
		return m, nil
	case initDoneMsg:
		m = <-updates    // Replace the ui we have with the new, initialized one. Fine in this case as we return m further down the method.
		m.Push(MainMenu) // Finished initializing. Replace the stack with a new one containing only the main menu
		return m, tea.ClearScreen
	case errMsg:
		m.err = msg.err
		if msg.fatal {
			m.Push(FatalError)
			return m, tea.Sequence(tea.ExitAltScreen, tea.Quit) // Need to exit alt screen first or the error message doesn't appear for long enough
		}
		tea.Println()
	}

	return m, nil
}

func (m model) View() (s string) {
	switch m.Peek() {
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
		s = menuView(m, "Welcome to the Analogue Pocket library editor", mainMenuOptions)
	case LibraryMenu:
		opt := libraryOptions
		if !m.ShowAdd {
			opt = libraryOptions[1:]
		}
		s = menuView(m, "Main > Library", opt)
	case ThumbMenu:
		s = menuView(m, "Main > Thumbs", thumbOptions)
	case ConfigMenu:
		s = settingsView(m, "Main > Settings")

	case GameList, AddScreen, EditScreen:
		fallthrough
	default:
		s = fmt.Sprintf("Welcome to the default zone!")
	}

	return
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

	updates <- m
	return initDoneMsg{}
}

// save is the opposite of init: save our data to disk
func (m model) save() tea.Msg {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = os.Mkdir(fmt.Sprintf("%s/library-editor", wd), 755)
	if !os.IsExist(err) {
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
	*m.percent = 100.0
	updates <- m
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
	*m.percent = 100.0
	updates <- m
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
	*m.percent = 100.0
	updates <- m
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
	*m.percent = 100.0
	updates <- m
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
	*m.percent = 100.0
	updates <- m
	return updateMsg{}
}

// keyMsg handles all the Update logic for key presses
func keyMsg(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k := msg.String(); k {
	case "ctrl+c":
		return m, tea.Quit // Ctrl-C always quits, even while saving.
	}
	if m.anyKey {
		m.anyKey = false
		m.Pop()
		*m.percent = 0.0
		return m, nil
	}
	if m.Peek() == Saving || m.Peek() == Waiting {
		return m, nil
	}
	switch k := msg.String(); k {
	case "q":
		if m.Peek() == MainMenu {
			return m, tea.Quit
		}
	case "esc", "backspace": // ESC or backspace always lets us go up a screen. Unless we're at the top screen. Then it quits without saving
		return back(m)
	case "up", "w", "i":
		m.pos = adjustCursor(m, -1)
	case "down", "s", "k":
		m.pos = adjustCursor(m, 1)
	case "left", "a", "j": // left functions the same as backspace
		return back(m)
	case "right", "d", "l":
	case "enter", " ":
		//fmt.Println(m.pos) // FIXME: Debug statement
		switch m.Peek() {
		case MainMenu:
			return mainMenu(m)
		case LibraryMenu:
			return libMenu(m)
		case ThumbMenu:
			return thumbMenu(m)
		case ConfigMenu:
			return configMenu(m)
		}
	}
	return m, nil
}

// back handles moving the UI back up one screen
func back(m model) (tea.Model, tea.Cmd) {
	switch m.Pop() {
	//case MainMenu:
	//	return m, tea.Quit
	case ConfigMenu, LibraryMenu, ThumbMenu:
		m.pos = m.main
	case AddScreen, EditScreen:
		m.pos = m.lib
	case GameList:
		if m.Peek() == LibraryMenu {
			m.pos = m.lib
		} else {
			m.pos = m.thumb
		}
	}
	return m, nil
}

// mainMenu handles enter / space on the main menu
func mainMenu(m model) (tea.Model, tea.Cmd) {
	m.main = m.pos
	switch m.pos {
	case 0: // Library
		m.pos = 0
		m.Push(LibraryMenu)
	case 1: // Thumbnails
		m.pos = 0
		m.Push(ThumbMenu)
	case 2: // Config
		m.pos = 0
		m.Push(ConfigMenu)
	case 3: // Save & Quit
		m.Push(Saving)
		return m, tea.Batch(m.save, tickCmd())
	case 4: // Quit
		return m, tea.Quit
	}
	return m, nil
}

// libMenu handles enter / space on the library menu
func libMenu(m model) (tea.Model, tea.Cmd) {
	m.lib = m.pos
	pos := m.pos
	if !m.ShowAdd { // If the add menu is hidden, bump everything up by 1
		pos = pos + 1
	}
	switch pos {
	case 0: // Add
		// TODO
	case 1: // Edit
		m.pos = 0
		// TODO
		//m.Push(ThumbMenu)
	case 2: // Remove
		m.pos = 0
		// TODO
		//m.Push(ConfigMenu)
	case 3: // Fix played times
		m.Push(Waiting)
		m.wait = "Fixing played times"
		return m, tea.Batch(m.playfix, tickCmd())
	case 4: // back
		return back(m)
	}
	return m, nil
}

// thumbMenu handles enter / space on the thumbnails menu
func thumbMenu(m model) (tea.Model, tea.Cmd) {
	m.thumb = m.pos
	switch m.pos {
	case 0: // Generate missing
		m.Push(Waiting)
		m.wait = "Generating missing thumbnails for library"
		return m, tea.Batch(m.genMissing, tickCmd())
	case 1: // Generate single
		m.pos = 0
		//m.Push(ThumbMenu)
	case 2: // Regenerate library
		m.Push(Waiting)
		m.wait = "Regenerating all thumbnails for library"
		return m, tea.Batch(m.regenLib, tickCmd())
	case 3: // Prune orphaned
		m.Push(Waiting)
		m.wait = "Removing orphaned thumbs.bin entries"
		return m, tea.Batch(m.prune, tickCmd())
	case 4: // Generate full library
		m.Push(Waiting)
		m.wait = "Generating thumbnails for all games in the Images folder. This may take a while."
		return m, tea.Batch(m.genFull, tickCmd())
	case 5: // Back
		return back(m)
	}
	return m, nil
}

// configMenu handles enter / space on the settings menu
func configMenu(m model) (tea.Model, tea.Cmd) {
	switch m.pos {
	case 0:
		m.RemoveImages = !m.RemoveImages
	case 1:
		m.AdvancedEditing = !m.AdvancedEditing
	case 2:
		m.ShowAdd = !m.ShowAdd
	case 3:
		return back(m)
	}
	return m, nil
}

// adjustCursor moves the cursor up or down in a menu, with code to prevent it from going beyond the boundaries
func adjustCursor(m model, i int) int {
	pos := m.pos + i
	if pos <= 0 {
		pos = 0
	} else {
		switch m.Peek() {
		case MainMenu:
			if pos > len(mainMenuOptions)-1 {
				pos = len(mainMenuOptions) - 1
			}
		case ThumbMenu:
			if pos > len(thumbOptions)-1 {
				pos = len(thumbOptions) - 1
			}
		case LibraryMenu:
			if m.ShowAdd {
				if pos > len(libraryOptions)-1 {
					pos = len(libraryOptions) - 1
				}
			} else {
				if pos > len(libraryOptions)-2 {
					pos = len(libraryOptions) - 2
				}
			}
		case ConfigMenu:
			if pos > 3 {
				pos = 3
			}
		}
	}
	return pos
}

// menuView is code for rendering a generic menu; used by the main, library, and thumbnail menus but not the settings
func menuView(m model, title string, options []string) string {
	tpl := "%s\n\n%s\n"

	var choices string
	for i, s := range options {
		choices = choices + fmt.Sprintf("%s\n", pointer(s, m.pos == i))
	}

	return fmt.Sprintf(tpl, title, choices)
}

// settingsView is used for rendering the settings menu. It is slightly different in that it needs to have visual checkboxes
func settingsView(m model, title string) string {
	configOptions := []string{
		"[%s] Remove thumbnail when removing game",
		"[%s] Show advanced library editing fields (Experimental)",
		"[%s] Show add library entry (Experimental)",
		"Back"}
	if m.RemoveImages {
		configOptions[0] = fmt.Sprintf(configOptions[0], "X")
	} else {
		configOptions[0] = fmt.Sprintf(configOptions[0], " ")
	}
	if m.AdvancedEditing {
		configOptions[1] = fmt.Sprintf(configOptions[1], "X")
	} else {
		configOptions[1] = fmt.Sprintf(configOptions[1], " ")
	}
	if m.ShowAdd {
		configOptions[2] = fmt.Sprintf(configOptions[2], "X")
	} else {
		configOptions[2] = fmt.Sprintf(configOptions[2], " ")
	}

	return menuView(m, title, configOptions)
}

// pointer renders the carat that points to the currently selected menu option.
func pointer(label string, checked bool) string {
	if checked {
		return selected.Render("> " + label)
	}
	return fmt.Sprintf("  %s", label)
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
