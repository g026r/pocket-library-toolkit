package ui

// This file exists just to keep model.go somewhat smaller.
type menuKey string

const (
	lib      menuKey = "lib"
	thumbs   menuKey = "thumbs"
	config   menuKey = "config"
	save     menuKey = "save"
	quit     menuKey = "quit"
	add      menuKey = "add"
	edit     menuKey = "edit"
	rm       menuKey = "rm"
	fix      menuKey = "fix"
	back     menuKey = "back"
	missing  menuKey = "missing"
	single   menuKey = "single"
	genlib   menuKey = "genlib"
	all      menuKey = "all"
	prune    menuKey = "prune"
	showAdd  menuKey = "showAdd"
	advEdit  menuKey = "advEdit"
	rmThumbs menuKey = "rmThumbs"
)

type screen int

const (
	MainMenu screen = iota
	LibraryMenu
	ThumbMenu
	ConfigMenu
	EditList
	RemoveList
	GenerateList
	AddScreen
	EditScreen
	Saving
	Waiting
	Initializing
	FatalError
)
