package ui

import "testing"

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

func TestStack_Clear(t *testing.T) {
	t.Parallel()
	sut := stack{}
	if len(sut.s) != 0 {
		t.Fatalf("stack not empty")
	}

	sut.Clear()
	if len(sut.s) != 0 {
		t.Errorf("stack not empty: %v", sut.s)
	}

	for i := range FatalError {
		for j := range i {
			sut.Push(j)
		}
		sut.Clear()
		if len(sut.s) != 0 {
			t.Errorf("stack not empty: %v", sut.s)
		}
	}
}
