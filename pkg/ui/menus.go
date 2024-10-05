package ui

import (
	"fmt"
	goio "io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/g026r/pocket-toolkit/pkg/io"
	"github.com/g026r/pocket-toolkit/pkg/models"
)

// menuKey consists of all the possible menu actions
type menuKey int

const (
	lib menuKey = iota
	thumbs
	config
	about
	save
	quit
	libAdd
	libEdit
	libRm
	libFix
	back
	tmMissing
	tmSingle
	tmGenlib
	tmAll
	tmPrune
	cfgShowAdd
	cfgAdvEdit
	cfgRmThumbs
	cfgGenNew
	cfgUnmodified
	cfgOverwrite
	cfgBackup
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).PaddingLeft(2).PaddingRight(2).Background(blue).Foreground(lipgloss.AdaptiveColor{Light: "#aaaaaa", Dark: "#111111"})
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(blue)
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	italic            = lipgloss.NewStyle().Italic(true)
	blue              = lipgloss.AdaptiveColor{Light: "#006699", Dark: "#00ccff"}
)

// menuItem is used for each menu that isn't a game list
// text represents the text to display
// key is the result that should be checked to determine the item selected
type menuItem struct {
	text string
	key  menuKey
}

func (m menuItem) FilterValue() string {
	return m.text
}

func (m menuItem) String() string {
	return m.text
}

var (
	mainMenuOptions = []list.Item{
		menuItem{"Library", lib},
		menuItem{"Thumbnails", thumbs},
		menuItem{"Settings", config},
		menuItem{"About", about},
		menuItem{"Save & Quit", save},
		menuItem{"Quit", quit}}
	libraryOptions = []list.Item{
		menuItem{"Add entry", libAdd},
		menuItem{"Edit entry", libEdit},
		menuItem{"Remove entry", libRm},
		menuItem{"Fix played times", libFix},
		menuItem{"Back", back}}
	thumbOptions = []list.Item{
		menuItem{"Generate missing thumbnails", tmMissing},
		menuItem{"Regenerate single game", tmSingle},
		menuItem{"Regenerate full library", tmGenlib},
		menuItem{"Prune orphaned thumbnails", tmPrune},
		menuItem{"Generate complete system thumbnails", tmAll},
		menuItem{"Back", back}}
	configOptions = []list.Item{
		menuItem{"Remove thumbnail when removing game", cfgRmThumbs},
		menuItem{"Generate new thumbnail when editing game", cfgGenNew},
		// menuItem{"Overwrite original files on save", cfgOverwrite},
		menuItem{"Backup files before overwriting", cfgBackup},
		menuItem{"Always save _thumbs.bin files, even if unmodified", cfgUnmodified},
		// menuItem{"Show advanced library editing fields " + italic.Render("(Experimental)"), cfgAdvEdit},
		// menuItem{"Show 'Add to Library' " + italic.Render("(Experimental)"), cfgShowAdd},
		menuItem{"Back", back}}

	// esc consists of the items to be performed if esc is typed
	esc = map[screen]func(m *Model, msg tea.Msg) (*Model, tea.Cmd){
		MainMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return m, nil // No op
		},
		LibraryMenu:  pop,
		ThumbMenu:    pop,
		ConfigMenu:   pop,
		EditList:     pop,
		GenerateList: pop,
		RemoveList:   pop,
	}

	// enter consists of the actions to be performed when an item is selected
	enter = map[screen]func(m *Model, msg tea.Msg) (*Model, tea.Cmd){
		MainMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return m.processMenuItem(m.mainMenu.SelectedItem().(menuItem).key)
		},
		LibraryMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return m.processMenuItem(m.subMenu.SelectedItem().(menuItem).key)
		},
		ThumbMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return m.processMenuItem(m.subMenu.SelectedItem().(menuItem).key)
		},
		ConfigMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return m.processMenuItem(m.configMenu.SelectedItem().(menuItem).key)
		},
		EditList: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			if len(m.gameList.Items()) == 0 {
				return m, nil
			}
			entry := m.gameList.SelectedItem().(models.Entry)
			m.focusedInput = 0
			m.gameInput[name].(*Input).SetValue(entry.Name)
			m.gameInput[system].(*Input).SetValue(entry.System.String())
			m.gameInput[crc].(*Input).SetValue(fmt.Sprintf("0x%08x", entry.Crc32))
			m.gameInput[sig].(*Input).SetValue(fmt.Sprintf("0x%08x", entry.Sig))
			m.gameInput[magic].(*Input).SetValue(fmt.Sprintf("0x%04x", entry.Magic))
			if p, ok := m.playTimes[entry.Sig]; ok {
				m.gameInput[added].(*Input).SetValue(time.Unix(int64(p.Added), 0).UTC().Format("2006-01-02 15:04:05"))
				m.gameInput[play].(*Input).SetValue(p.FormatPlayTime())
			} else {
				m.gameInput[added].(*Input).SetValue(time.Now().Format("2006-01-02 15:04"))
				m.gameInput[play].(*Input).SetValue("0h 0m 0s")
			}

			for i := range m.gameInput {
				if i != cancel && i != submit {
					// Don't need to reset Err here like we do with the add screen as SetValue triggers ValidateFunc
					m.gameInput[i].(*Input).CursorEnd()
				}
				m.gameInput[i].Style(itemStyle)
				m.gameInput[i].Blur()
			}

			// Name is always the first value selected
			m.gameInput[name].Style(selectedItemStyle.PaddingLeft(4))

			m.Push(EditScreen)
			return m, m.gameInput[name].Focus()
		},
		GenerateList: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			entry := m.gameList.SelectedItem().(models.Entry)
			m.Push(Waiting)
			m.percent = 0.0
			m.wait = fmt.Sprintf("Generating thumbnail for %s (%s)", entry.Name, entry.System)
			return m, tea.Batch(m.genSingle(entry), tickCmd())
		},
		RemoveList: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			if len(m.gameList.Items()) == 0 {
				return m, nil
			}
			idx := m.gameList.Index()
			m = m.removeEntry(idx)
			m.gameList.RemoveItem(idx)
			return m, func() tea.Msg {
				return updateMsg{}
			}
		},
	}

	// def consists of the default actions when nothing else is to be done
	def = map[screen]func(m *Model, msg tea.Msg) (*Model, tea.Cmd){
		MainMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return defaultAction(MainMenu, &m.mainMenu, m, msg)
		},
		LibraryMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return defaultAction(LibraryMenu, &m.subMenu, m, msg)
		},
		ThumbMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return defaultAction(ThumbMenu, &m.subMenu, m, msg)
		},
		ConfigMenu: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			return defaultAction(ConfigMenu, &m.configMenu, m, msg)
		},
		EditList: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			var cmd tea.Cmd
			m.gameList, cmd = m.gameList.Update(msg)
			return m, cmd
		},
		GenerateList: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			var cmd tea.Cmd
			m.gameList, cmd = m.gameList.Update(msg)
			return m, cmd
		},
		RemoveList: func(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
			var cmd tea.Cmd
			m.gameList, cmd = m.gameList.Update(msg)
			return m, cmd
		},
	}
)

// defaultAction is a default action for sub menus allowing numeric navigation.
// It's not easily doable for game list menus as there may be too many items to handle key-presses without storing the previous press & waiting to process it.
func defaultAction(scr screen, menu *list.Model, m *Model, msg tea.Msg) (*Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if i, err := strconv.Atoi(k.String()); err == nil && i >= 1 && i <= len(menu.Items()) {
			menu.Select(i - 1)
			return enter[scr](m, msg)
		}
	}
	var cmd tea.Cmd
	*menu, cmd = menu.Update(msg)
	return m, cmd
}

// itemDelegate is the default rendered for menuItem instances that aren't io.Config values.
// Though it can take anything that implements fmt.Stringer if need be.
type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }

// We're not using Update to process key presses as we need to update the Model in some cases & we don't have access to it.
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w goio.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(fmt.Stringer)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

// entryDelegate is the default renderer for models.Entry instances
type entryDelegate struct{}

func (d entryDelegate) Height() int                             { return 1 }
func (d entryDelegate) Spacing() int                            { return 0 }
func (d entryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d entryDelegate) Render(w goio.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(models.Entry)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s (%s)", index+1, i.Name, i.System)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

// configDelegate is the default renderer for menuItem instances that represent io.Config values.
type configDelegate struct {
	*io.Config
}

func (d configDelegate) Height() int  { return 1 }
func (d configDelegate) Spacing() int { return 0 }

// While we could conceivably use Update for the config menu, as we have a pointer to the object being updated, we're not
// just to keep the menu processing code all in one spot
func (d configDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d configDelegate) Render(w goio.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(menuItem)
	if !ok {
		return
	}

	var str string
	if i.key == back {
		str = fmt.Sprintf("%d. %s", index+1, i)
	} else {
		str = fmt.Sprintf("%d. [%%s] %s", index+1, i)
		var b bool
		switch i.key {
		case cfgRmThumbs:
			b = d.RemoveImages
		case cfgGenNew:
			b = d.GenerateNew
		case cfgUnmodified:
			b = d.SaveUnmodified
		case cfgOverwrite:
			b = d.Overwrite
		case cfgAdvEdit:
			b = d.AdvancedEditing
		case cfgShowAdd:
			b = d.ShowAdd
		case cfgBackup:
			b = d.Backup
		default:
			// If we don't know what this value is, return
			return
		}

		if b {
			str = fmt.Sprintf(str, "X")
		} else {
			str = fmt.Sprintf(str, " ")
		}
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

func NewMainMenu() *list.Model {
	mm := list.New(mainMenuOptions, itemDelegate{}, 0, 5)
	mm.Title = "Welcome to the unofficial Analogue Pocket library toolkit"
	mm.SetShowStatusBar(false)
	mm.Styles.Title = titleStyle
	mm.Styles.HelpStyle = helpStyle
	mm.Styles.PaginationStyle = paginationStyle
	mm.SetFilteringEnabled(false)
	mm.KeyMap.Quit.SetKeys("q") // Don't want "ESC" as a quit key here

	return &mm
}

// NewSubMenu exists to set the default values that we will use for all actual sub menus.
// It's empty to start with as we'll be modifying it as necessary depending on the actions the user has taken
func NewSubMenu() *list.Model {
	sm := list.New([]list.Item{}, itemDelegate{}, 0, 0) // Empty to start with
	sm.Title = ""                                       // Blank to start with
	sm.SetShowStatusBar(false)
	sm.Styles.Title = titleStyle
	sm.Styles.HelpStyle = helpStyle
	sm.Styles.PaginationStyle = paginationStyle
	sm.SetFilteringEnabled(false)
	sm.KeyMap.Quit.SetEnabled(false)
	sm.KeyMap.Quit.Unbind()

	return &sm
}

// NewGameMenu exists to set the default values that we will use for all actual game menus.
// It's empty to start with as we'll be modifying it as necessary depending on the actions the user has taken
// It differs from NewSubMenu in that it uses the entryDelegate & has filtering enabled.
func NewGameMenu() *list.Model {
	gm := list.New([]list.Item{}, entryDelegate{}, 0, 0) // Empty to start with
	gm.Title = ""                                        // Blank to start with
	gm.SetShowStatusBar(false)
	gm.Styles.Title = titleStyle
	gm.Styles.HelpStyle = helpStyle
	gm.Styles.PaginationStyle = paginationStyle
	gm.SetFilteringEnabled(true)
	gm.SetStatusBarItemName("game", "games")
	gm.KeyMap.Quit.SetEnabled(false)
	gm.KeyMap.Quit.Unbind()

	return &gm
}

func NewConfigMenu(config *io.Config) *list.Model {
	cm := list.New(configOptions, configDelegate{Config: config}, 0, 0) // Empty to start with; replaced when called
	cm.Title = "Main > Settings"
	cm.SetShowStatusBar(false)
	cm.Styles.Title = titleStyle
	cm.Styles.HelpStyle = helpStyle
	cm.Styles.PaginationStyle = paginationStyle
	cm.SetFilteringEnabled(false)
	cm.KeyMap.Quit.SetEnabled(false)
	cm.KeyMap.Quit.Unbind()

	return &cm
}

func generateGameList(l list.Model, entries []models.Entry, title string, width, height int) list.Model {
	// TODO: Could I save memory if I modified this in place? (And is it worth it?)
	items := make([]list.Item, 0)
	for _, e := range entries {
		items = append(items, e)
	}

	return generateSubMenu(l, items, title, width, height)
}

func generateSubMenu(l list.Model, items []list.Item, title string, width, height int) list.Model {
	l.Title = title
	l.ResetSelected()
	l.ResetFilter()
	l.SetItems(items)
	// Need to reset height & width or else it doesn't display right the first time since the number of items in the list
	// has changed from the initial WindowSizeMsg that's fired on startup
	// It's overkill after it's displayed once, but an issue until then.
	l.SetWidth(width)
	l.SetHeight(height)

	return l
}
