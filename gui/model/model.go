package model

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/g026r/pocket-library-editor/gui/io"
	model2 "github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

type screen int

const (
	Loading     screen = iota
	MainMenu    screen = iota
	LibraryMenu screen = iota
	ThumbMenu   screen = iota
	ConfigMenu  screen = iota
	GameList    screen = iota
	AddScreen   screen = iota
	EditScreen  screen = iota
	GenerateAll screen = iota
	Saving      screen = iota
	FatalError  screen = iota
)

var (
	selected = lipgloss.NewStyle().Bold(true)

	mainMenuOptions = []string{"Library", "Thumbnails", "Settings", "Save & Quit", "Quit"}
	libraryOptions  = []string{"Add entry", "Edit entry", "Remove entry", "Fix played times"}
	thumbOptions    = []string{"Generate missing thumbnails", "Regenerate game thumbnail", "Regenerate complete library", "Prune orphaned thumbnails", "Generate complete system thumbnails"}
)

type errMsg struct {
	err   error
	fatal bool
}

type initMsg struct {
	model
}

type stack struct {
	s []screen
}

type model struct {
	RootDir   fs.FS
	Entries   []model2.Entry
	PlayTimes map[uint32]model2.PlayTime
	Thumbs    map[util.System]model2.Thumbnails
	io.Config
	Internal map[util.System][]model2.Entry // Internal is a map of all known possible entries, grouped by system
	stack
	spinner spinner.Model
	err     error
	pos     int // pos stores the generic cursor position
	main    int // main stores the position of the cursor on the main menu for when we go back up to it
	lib     int // lib stores the position of the cursor on the library menu for when we go back up to it
	thumb   int // thumb stores the position of the cursor on the thumbnails menu for when we go back up to it
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.initSystem)
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

	return initMsg{m}
}

func (m model) save() tea.Msg {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = os.Mkdir(fmt.Sprintf("%s/library-editor", wd), 755)
	if !os.IsExist(err) {
		return errMsg{err, true}
	}

	if err := io.SaveLibrary(m.Entries, m.PlayTimes); err != nil {
		return errMsg{err, true}
	}
	if err := io.SaveThumbs(m.Thumbs); err != nil {
		return errMsg{err, true}
	}
	if err := io.SaveConfig(m.Config); err != nil {
		return errMsg{err, true}
	}

	return tea.QuitMsg{}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch k := msg.String(); k {
		case "ctrl+c":
			return m, tea.Quit // Ctrl-C always quits, even while saving.
		}
		if m.Peek() == Saving {
			break // Disable all other keys while saving.
		}
		switch k := msg.String(); k {
		case "q", "esc": // ESC always lets us go up a screen. Unless we're at the top screen. Then it quits without saving
			switch m.Pop() {
			case Loading, MainMenu:
				return m, tea.Quit
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
		case "up", "w", "i":
			m.pos = adjustCursor(m, -1)
		case "down", "s", "k":
			m.pos = adjustCursor(m, 1)
		case "left", "a", "j":
		case "right", "d", "l":
		case "enter", "space":
			//fmt.Println(m.pos) // FIXME: Debug statement
			switch m.Peek() {
			case MainMenu:
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
					//return m, tea.ClearScreen
				case 3: // Save & Quit
					m.spinner = spinner.New(spinner.WithSpinner(spinner.MiniDot))
					m.Push(Saving)
					return m, tea.Batch(m.save, m.spinner.Tick)
				case 4: // Quit
					return m, tea.Quit
				}
			case LibraryMenu:
			case ThumbMenu:
			case ConfigMenu:
				switch m.pos {
				case 0:
					m.RemoveImages = !m.RemoveImages
				case 1:
					m.AdvancedEditing = !m.AdvancedEditing
				case 2:
					m.ShowAdd = !m.ShowAdd
				}
				//return m, tea.ClearScreen
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case initMsg:
		m = msg.model    // Replace the model we have with the new, initialized one. Fine in this case as we return m further down the method.
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
			if pos > 2 {
				pos = 2
			}
		}
	}

	return pos
}

func (m model) View() (s string) {
	switch m.Peek() {
	case Loading:
		s = fmt.Sprintf("%s Loading your Pocket library...", m.spinner.View())
	case Saving:
		s = fmt.Sprintf("%s Saving your Pocket library...", m.spinner.View())
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

	case GameList, AddScreen, EditScreen, GenerateAll:
		fallthrough
	default:
		s = fmt.Sprintf("Welcome to the default zone!")
	}

	return
}

func menuView(m model, title string, options []string) string {
	tpl := "%s\n\n%s\n"

	var choices string
	for i, s := range options {
		choices = choices + fmt.Sprintf("%s\n", pointer(s, m.pos == i))
	}

	return fmt.Sprintf(tpl, title, choices)
}

func settingsView(m model, title string) string {
	configOptions := []string{
		"[%s] Remove thumbnail when removing game",
		"[%s] Show advanced library editing fields (Experimental)",
		"[%s] Show add library entry (Experimental)"}
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

func pointer(label string, checked bool) string {
	if checked {
		return selected.Render("> " + label)
	}
	return fmt.Sprintf("  %s", label)
}

func (s *stack) Peek() screen {
	if len(s.s) == 0 {
		return Loading
	}
	return s.s[len(s.s)-1]
}

func (s *stack) Pop() screen {
	if len(s.s) == 0 {
		return Loading
	}
	rm := s.s[len(s.s)-1]
	s.s = s.s[:len(s.s)-1]
	return rm
}

func (s *stack) Push(v screen) {
	s.s = append(s.s, v)
}

func NewModel() tea.Model {
	return model{spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
}
