package ui

import (
	"testing"

	model2 "github.com/g026r/pocket-library-editor/pkg/model"
)

func TestStack_Peek(t *testing.T) {
	t.Parallel()
	stk := stack{}

	// Confirm that an empty stack returns MainMenu
	if stk.Peek() != MainMenu {
		t.Errorf("Expected %v got %v", MainMenu, stk.Peek())
	}

	// Confirm that if we add this to the slice, it returns properly
	stk.s = append(stk.s, ThumbMenu)
	if stk.Peek() != ThumbMenu {
		t.Errorf("Expected %v got %v", ThumbMenu, stk.Peek())
	}
	// Confirm that Peek() didn't Pop()
	if len(stk.s) != 1 {
		t.Errorf("Expected %d got %d", 1, len(stk.s))
	}

	// Confirm that Peek returns the most recent addition
	stk.s = append(stk.s, FatalError)
	if stk.Peek() != FatalError {
		t.Errorf("Expected %d got %d", FatalError, stk.Peek())
	}
	if len(stk.s) != 2 {
		t.Errorf("Expected %d got %d", 2, len(stk.s))
	}

}

func TestStack_Pop(t *testing.T) {
	t.Parallel()
	sut := stack{}

	// Confirm that an empty stack returns MainMenu
	if sut.Pop() != MainMenu {
		t.Errorf("Expected %d got %d", MainMenu, sut.Peek())
	}

	// Confirm that if we add this to the slice, it returns properly
	sut.s = append(sut.s, ThumbMenu, Waiting)
	if sut.Pop() != Waiting {
		t.Errorf("Expected %d got %d", Waiting, sut.Peek())
	}

	// Confirm that Pop() didn't Peek()
	if sut.Peek() != ThumbMenu {
		t.Errorf("Expected %d got %d", ThumbMenu, sut.Peek())
	}
	if len(sut.s) != 1 {
		t.Errorf("Expected %v got %v", []screen{}, sut.s)
	}
}

func TestStack_Push(t *testing.T) {
	t.Parallel()
	sut := stack{}
	if len(sut.s) != 0 {
		t.Fatalf("stack not empty")
	}
	sut.Push(ConfigMenu)
	if len(sut.s) != 1 {
		t.Errorf("Expected %d got %d", 1, len(sut.s))
	}
	if sut.Peek() != ConfigMenu {
		t.Errorf("Expected %d got %d", ConfigMenu, sut.Peek())
	}

	// Confirm that Push goes on the top of the stack
	sut.Push(EditScreen)
	if len(sut.s) != 2 {
		t.Errorf("Expected %d got %d", 1, len(sut.s))
	}
	if sut.Peek() != EditScreen {
		t.Errorf("Expected %d got %d", EditScreen, sut.Peek())
	}
}

func TestApplication_fixPlayTimes(t *testing.T) {
	t.Parallel()
	var p float64
	sut := model{
		updates: make(chan model, 1),
		percent: &p,
		PlayTimes: map[uint32]model2.PlayTime{
			0x0: {Played: 0x0000ABCD}, 0x1: {Played: 0x0100ABCD}, 0x40: {Played: 0x0400ABCD}, 0xF: {Played: 0xFF00ABCD},
		}}

	msg := sut.playfix()
	switch msg.(type) {
	case updateMsg: // Don't need to do anything
	default:
		t.Errorf("Expected updateMsg got %v", msg)
	}

	sut = <-sut.updates
	for k, v := range sut.PlayTimes {
		if v.Played != 0x0000ABCD {
			t.Errorf("0x%02x Expected 0x0000ABCD; got 0x%08x", k, v.Played)
		}
	}
}
