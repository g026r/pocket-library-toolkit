package ui

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
	AboutScreen
)

// stack is a simple struct for keeping track of the screen we are currently on & the ones that came before it.
// It's probably overkill as we could have just used different enum values for some of the screens
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

// Replace the top element in the stack with v, returning the element replaced
// Equivalent to calling Pop and then Push(v)
func (s *stack) Replace(v screen) screen {
	scr := s.Pop()
	s.Push(v)
	return scr
}
