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

	italicStyle = lipgloss.NewStyle().Italic(true)
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
	mainMenuOptions = []list.Item{menuItem{"Library", lib}, menuItem{"Thumbnails", thumbs}, menuItem{"Settings", config}, menuItem{"Save & Quit", save}, menuItem{"Quit", quit}}
	libraryOptions  = []list.Item{menuItem{"Add entry", add}, menuItem{"Edit entry", edit}, menuItem{"Remove entry", rm}, menuItem{"Fix played times", fix}, menuItem{"Back", back}}
	thumbOptions    = []list.Item{menuItem{"Generate missing thumbnails", missing}, menuItem{"Regenerate game thumbnail", single}, menuItem{"Regenerate complete library", genlib}, menuItem{"Prune orphaned thumbnails", prune}, menuItem{"Generate complete system thumbnails", all}, menuItem{"Back", back}}
	configOptions   = []list.Item{
		menuItem{"Remove thumbnail when removing game", rmThumbs},
		menuItem{"Show advanced library editing fields " + lipgloss.NewStyle().Italic(true).Render("(Experimental)"), advEdit},
		menuItem{"Show add library entry " + lipgloss.NewStyle().Italic(true).Render("(Experimental)"), showAdd},
		menuItem{"Back", back}}
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

	str := fmt.Sprintf("%d. %s %s", index+1, i.Name, italicStyle.Render(fmt.Sprintf("(%s)", i.System)))

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

func NewLibraryMenu() *list.Model {
	lm := list.New(libraryOptions, itemDelegate{}, 0, 0)
	lm.Title = "Main > Library"
	lm.SetShowStatusBar(false)
	lm.Styles.Title = titleStyle
	lm.Styles.HelpStyle = helpStyle
	lm.Styles.PaginationStyle = paginationStyle
	lm.SetFilteringEnabled(false)
	lm.KeyMap.Quit.SetEnabled(false)
	lm.KeyMap.Quit.SetKeys()

	return &lm
}

func NewThumbMenu() *list.Model {
	tm := list.New(thumbOptions, itemDelegate{}, 0, 0)
	tm.Title = "Main > Thumbnails"
	tm.SetShowStatusBar(false)
	tm.Styles.Title = titleStyle
	tm.Styles.HelpStyle = helpStyle
	tm.Styles.PaginationStyle = paginationStyle
	tm.SetFilteringEnabled(false)
	tm.KeyMap.Quit.SetEnabled(false)
	tm.KeyMap.Quit.SetKeys()

	return &tm
}

func NewGameMenu(title string) *list.Model {
	gm := list.New([]list.Item{}, entryDelegate{}, 0, 0) // Empty to start with
	gm.Title = title
	gm.SetShowStatusBar(false)
	gm.Styles.Title = titleStyle
	gm.Styles.HelpStyle = helpStyle
	gm.Styles.PaginationStyle = paginationStyle
	gm.SetFilteringEnabled(true)
	gm.SetStatusBarItemName("game", "games")
	gm.KeyMap.Quit.SetEnabled(false)
	gm.KeyMap.Quit.SetKeys()

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
	cm.KeyMap.Quit.SetKeys()

	return &cm
}

func menuHandler(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var opt menuKey
	switch m.Peek() {
	case MainMenu:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
			opt = m.mainMenu.SelectedItem().(menuItem).key
		} else {
			*m.mainMenu, cmd = m.mainMenu.Update(msg)
		}
	case LibraryMenu:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
			opt = m.libMenu.SelectedItem().(menuItem).key
		} else if ok && k.String() == "esc" {
			m.Pop()
		} else {
			*m.libMenu, cmd = m.libMenu.Update(msg)
		}
	case ThumbMenu:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
			opt = m.thumbMenu.SelectedItem().(menuItem).key
		} else if ok && k.String() == "esc" {
			m.Pop()
		} else {
			*m.thumbMenu, cmd = m.thumbMenu.Update(msg)
		}
	case ConfigMenu:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
			opt = m.configMenu.SelectedItem().(menuItem).key
		} else if ok && k.String() == "esc" {
			m.Pop()
		} else {
			*m.configMenu, cmd = m.configMenu.Update(msg)
		}
	case EditList:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" && !m.editList.SettingFilter() {
			entry := m.editList.SelectedItem().(model2.Entry)
			m.Push(EditScreen)
			return m, func() tea.Msg { // TODO: Need to find a way to pass this back
				return entry
			}
		} else if ok && k.String() == "esc" && !m.editList.SettingFilter() {
			m.editList.ResetFilter()
			m.editList.ResetSelected()
			m.Pop()
		} else {
			*m.editList, cmd = m.editList.Update(msg)
		}
	case GenerateList:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" && !m.editList.SettingFilter() {
			entry := m.generateList.SelectedItem().(model2.Entry)
			m.Push(Waiting)
			m.wait = fmt.Sprintf("Generating thumbnail for %s", entry.Name)
			return m, tea.Batch(m.genSingle(entry), tickCmd())
		} else if ok && k.String() == "esc" && !m.generateList.SettingFilter() {
			m.generateList.ResetFilter()
			m.generateList.ResetSelected()
			m.Pop()
		} else {
			*m.generateList, cmd = m.generateList.Update(msg)
		}
	case RemoveList:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" && !m.editList.SettingFilter() {
			idx := m.removeList.Index()
			m = m.removeEntry(idx)
			m.removeList.RemoveItem(idx)
			m.updates <- m
			return m, func() tea.Msg {
				return updateMsg{}
			}
		} else if ok && k.String() == "esc" && !m.removeList.SettingFilter() {
			m.removeList.ResetFilter()
			m.removeList.ResetSelected()
			m.Pop()
		} else {
			*m.removeList, cmd = m.removeList.Update(msg)
		}
	}

	if opt != "" {
		m, cmd = processMenuItem(m, opt)
	}
	return m, cmd
}

func generateGameList(l *list.Model, entries []model2.Entry, width, height int) *list.Model {
	items := make([]list.Item, 0)
	for _, e := range entries {
		items = append(items, e)
	}

	l.ResetSelected()
	l.ResetFilter()
	l.SetItems(items)
	// Need to reset height & width or it doesn't display right the first time since the number of items in the list
	// has changed from the initial WindowSizeMsg that's fired on startup
	// It's overkill after it's displayed once, but an issue until then.
	l.SetWidth(width)
	l.SetHeight(height)

	return l
}
