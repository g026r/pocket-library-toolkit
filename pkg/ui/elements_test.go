package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/g026r/pocket-library-toolkit/pkg/models"
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
	if err := playValidate("1,230"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("0h 0m 0s"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := playValidate("0h0m0s"); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}

	if err := playValidate("1,000h0m0s"); err != nil {
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

	if err := playValidate(fmt.Sprintf("%d", MAX_PLAYTIME+1)); err == nil {
		t.Error("Expected err but got nil")
	}

	if err := playValidate("not a number"); err == nil {
		t.Error("Expected err but got nil")
	}
}

func Test_DateValidate(t *testing.T) {
	t.Parallel()

	dates := []string{"2023-04-02", "2023/04/02"}
	times := []string{"2:34:05", "2:34", "2:34:05PM", "2:34PM", "2:34 PM", "2:34:05 PM", "02:34:05", "02:34:05", "14:34:05", "14:34"}

	// All the possible date/time combinations I permit
	for _, d := range dates {
		for _, h := range times {
			if err := dateValidate(strings.Join([]string{d, h}, " ")); err != nil {
				t.Errorf("Expected nil but got %v", err)
			}
		}
	}

	if err := dateValidate(""); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := dateValidate(strings.Join([]string{" ", dates[0], times[0], " "}, " ")); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	if err := dateValidate(dates[0]); err == nil {
		t.Errorf("Expected err but got nil")
	}
	if err := dateValidate(times[0]); err == nil {
		t.Errorf("Expected err but got nil")
	}
	if err := dateValidate(dates[0] + times[0]); err == nil {
		t.Errorf("Expected err but got nil")
	}
}

func Test_parsePlayTimes(t *testing.T) {
	t.Parallel()

	if p := parsePlayTime("123"); p != 123 {
		t.Errorf("Expected %d; got %d", 123, p)
	}
	if p := parsePlayTime("1,230"); p != 1230 {
		t.Errorf("Expected %d; got %d", 1230, p)
	}
	if p := parsePlayTime("123s"); p != 123 {
		t.Errorf("Expected %d; got %d", 123, p)
	}
	if p := parsePlayTime("2m 3s"); p != 123 {
		t.Errorf("Expected %d; got %d", 123, p)
	}
	if p := parsePlayTime("7h 40m 15s"); p != 27615 {
		t.Errorf("Expected %d; got %d", 27615, p)
	}
	if p := parsePlayTime("7h 40m 1,500s"); p != 29100 {
		t.Errorf("Expected %d; got %d", 29100, p)
	}
	if p := parsePlayTime("7d 40m 1,500s"); p != 0 { // Invalid value returns 0
		t.Errorf("Expected %d; got %d", 0, p)
	}
}

func Test_notBlank(t *testing.T) {
	t.Parallel()

	if err := notBlank(""); err == nil {
		t.Errorf("Expected err but got nil")
	}
	if err := notBlank("  "); err == nil {
		t.Errorf("Expected err but got nil")
	}
	if err := notBlank(" . "); err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
}
