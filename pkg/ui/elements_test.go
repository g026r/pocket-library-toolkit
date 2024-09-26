package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/g026r/pocket-toolkit/pkg/models"
)

func Test_SysValidate(t *testing.T) {
	t.Parallel()
	// Check all the real systems
	for i := range models.Lynx + 1 {
		if err := sysValidate(i.String()); err != nil {
			t.Errorf("Expected nil but got %v", err)
		}
	}

	// Check blank
	if err := sysValidate(""); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	// Check a real system but lower case
	if err := sysValidate(strings.ToLower(models.GBC.String())); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	// Check a real system + spaces
	// Check a real system but lower case
	if err := sysValidate(fmt.Sprintf("  %s   ", models.GG)); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	// Check an invalid system
	if err := sysValidate("  %s   "); err == nil {
		t.Error("Expected error but got nil")
	}
}

func Test_HexValidate(t *testing.T) {
	t.Parallel()

	s := ""
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = "0"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate(" 0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "1"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate(" 0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "2"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate("0x" + s + " "); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "3"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate("0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "A"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate("0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "b"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate("0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "c"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate("0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	s = s + "d"
	if err := hexValidate(s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := hexValidate("0x" + s); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	// Too long
	if err := hexValidate(s + "e"); err == nil {
		t.Error("Expected err but got nil")
	}
	if err := hexValidate("0x" + s + "e"); err == nil {
		t.Error("Expected err but got nil")
	}

	// Invalid char
	if err := hexValidate(s + "q"); err == nil {
		t.Error("Expected err but got nil")
	}
	if err := hexValidate("0x" + s + "q"); err == nil {
		t.Error("Expected err but got nil")
	}
}

func Test_PlayValidate(t *testing.T) {
	t.Parallel()
	if err := playValidate(""); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("123"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("0h 0m 0s"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := playValidate("0h0m0s"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("  111h 34s"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("34s  "); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("7 h  "); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("575m"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("45h6m"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("-1"); err == nil {
		t.Error("Expected err but got nil")
	}

	if err := playValidate("not a number"); err == nil {
		t.Error("Expected err but got nil")
	}
}

func Test_DateValidate(t *testing.T) {

}
