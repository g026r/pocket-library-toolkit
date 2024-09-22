package ui

import (
	"fmt"
	goio "io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/g026r/pocket-library-editor/pkg/io"
	model2 "github.com/g026r/pocket-library-editor/pkg/model"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).PaddingLeft(2).PaddingRight(2).Background(lipgloss.AdaptiveColor{Light: "#006699", Dark: "#00ccff"}).Foreground(lipgloss.AdaptiveColor{Light: "#aaaaaa", Dark: "#111111"})
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.AdaptiveColor{Light: "#006699", Dark: "#00ccff"})
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	italicStyle       = lipgloss.NewStyle().Italic(true)
)

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
		menuItem{"Save & Quit", save},
		menuItem{"Quit", quit}}
	libraryOptions = []list.Item{
		menuItem{"Add entry", add},
		menuItem{"Edit entry", edit},
		menuItem{"Remove entry", rm},
		menuItem{"Fix played times", fix},
		menuItem{"Back", back}}
	thumbOptions = []list.Item{
		menuItem{"Generate missing thumbnails", missing},
		menuItem{"Regenerate game thumbnail", single},
		menuItem{"Regenerate complete library", genlib},
		menuItem{"Prune orphaned thumbnails", prune},
		menuItem{"Generate complete system thumbnails", all},
		menuItem{"Back", back}}
	configOptions = []list.Item{
		menuItem{"Remove thumbnail when removing game", rmThumbs},
		menuItem{"Show advanced library editing fields " + italicStyle.Render("(Experimental)"), advEdit},
		menuItem{"Show 'Add to Library' " + italicStyle.Render("(Experimental)"), showAdd},
		menuItem{"Back", back}}

	// pop is the ESC action for basically everything but main menu
	// It removes the latest item from the stack, allowing the rendering to go up one level
	pop = func(m model, msg tea.Msg) (model, tea.Cmd) {
		m.Pop()
		return m, nil
	}

	// esc consists of the items to be performed if esc is typed
	esc = map[screen]func(m model, msg tea.Msg) (model, tea.Cmd){
		MainMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
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
	enter = map[screen]func(m model, msg tea.Msg) (model, tea.Cmd){
		MainMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			return processMenuItem(m, m.mainMenu.SelectedItem().(menuItem).key)
		},
		LibraryMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			return processMenuItem(m, m.subMenu.SelectedItem().(menuItem).key)
		},
		ThumbMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			return processMenuItem(m, m.subMenu.SelectedItem().(menuItem).key)
		},
		ConfigMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			return processMenuItem(m, m.configMenu.SelectedItem().(menuItem).key)
		},
		EditList: func(m model, msg tea.Msg) (model, tea.Cmd) {
			entry := m.gameList.SelectedItem().(model2.Entry)
			m.Push(EditScreen)
			return m, func() tea.Msg { // TODO: Need to find a way to pass this back. Or can I just use the menu idx instead?
				return entry
			}
		},
		GenerateList: func(m model, msg tea.Msg) (model, tea.Cmd) {
			entry := m.gameList.SelectedItem().(model2.Entry)
			m.Push(Waiting)
			m.wait = fmt.Sprintf("Generating thumbnail for %s (%s)", entry.Name, entry.System)
			return m, tea.Batch(m.genSingle(entry), tickCmd())
		},
		RemoveList: func(m model, msg tea.Msg) (model, tea.Cmd) {
			idx := m.gameList.Index()
			m = m.removeEntry(idx)
			m.gameList.RemoveItem(idx)
			m.updates <- m
			return m, func() tea.Msg {
				return updateMsg{}
			}
		},
	}

	// def consists of the default actions when nothing else is to be done
	def = map[screen]func(m model, msg tea.Msg) (model, tea.Cmd){
		MainMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.mainMenu, cmd = m.mainMenu.Update(msg)
			return m, cmd
		},
		LibraryMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.subMenu, cmd = m.subMenu.Update(msg)
			return m, cmd
		},
		ThumbMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.subMenu, cmd = m.subMenu.Update(msg)
			return m, cmd
		},
		ConfigMenu: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.configMenu, cmd = m.configMenu.Update(msg)
			return m, cmd
		},
		EditList: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.gameList, cmd = m.gameList.Update(msg)
			return m, cmd
		},
		GenerateList: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.gameList, cmd = m.gameList.Update(msg)
			return m, cmd
		},
		RemoveList: func(m model, msg tea.Msg) (model, tea.Cmd) {
			var cmd tea.Cmd
			*m.gameList, cmd = m.gameList.Update(msg)
			return m, cmd
		},
	}
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
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

	fmt.Fprint(w, fn(str))
}

type entryDelegate struct{}

func (d entryDelegate) Height() int                             { return 1 }
func (d entryDelegate) Spacing() int                            { return 0 }
func (d entryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d entryDelegate) Render(w goio.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(model2.Entry)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s (%s)", index+1, i.Name, i.System) //italicStyle.Render(fmt.Sprintf("(%s)", i.System)))

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type configDelegate struct {
	*io.Config
}

func (d configDelegate) Height() int                             { return 1 }
func (d configDelegate) Spacing() int                            { return 0 }
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
		case advEdit:
			b = d.AdvancedEditing
		case showAdd:
			b = d.ShowAdd
		case rmThumbs:
			b = d.RemoveImages
		default:
			// Don't know what this is. Return
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

	fmt.Fprint(w, fn(str))
}

func NewMainMenu() *list.Model {
	mm := list.New(mainMenuOptions, itemDelegate{}, 0, 5)
	mm.Title = "Welcome to the unofficial Analogue Pocket library editor"
	mm.SetShowStatusBar(false)
	mm.Styles.Title = titleStyle
	mm.Styles.HelpStyle = helpStyle
	mm.Styles.PaginationStyle = paginationStyle
	mm.SetFilteringEnabled(false)
	mm.KeyMap.Quit.SetKeys("q") // Don't want "ESC" as a quit key here

	return &mm
}

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

func menuHandler(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
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

func generateGameList(l list.Model, entries []model2.Entry, title string, width, height int) list.Model {
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
