package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/g026r/pocket-toolkit/pkg/models"
	"github.com/g026r/pocket-toolkit/pkg/util"
)

// The input elements, including buttons.
const (
	name = iota
	system
	crc
	sig
	magic
	added
	play
	submit
	cancel
)

var (
	red          = lipgloss.AdaptiveColor{Light: "#992200", Dark: "#ff8800"}
	focusedStyle = selectedItemStyle.PaddingLeft(4)
	errorStyle   = lipgloss.NewStyle().Foreground(red).PaddingLeft(6)

	playRegex = regexp.MustCompile(`^((?P<hours>\d+)[hH])?((?P<minutes>\d+)[mM])?((?P<seconds>\d+)[sS])?$`)
)

// FocusBlurViewer really exists so that Button & textinput.Model instances can be mostly treated the same
type FocusBlurViewer interface {
	Reset()
	Focus() tea.Cmd
	Blur()
	Focused() bool
	View() string
	error() string
	Style(style lipgloss.Style)
	Update(tea.Msg) (FocusBlurViewer, tea.Cmd)
	Value() string
}

// Input is a simple wrapper for textinput.Model to deal with the way it defined Update
type Input struct {
	textinput.Model
}

func (i *Input) error() string {
	if i.Err == nil {
		return ""
	}
	return i.Err.Error()
}

func (i *Input) Update(msg tea.Msg) (FocusBlurViewer, tea.Cmd) {
	var cmd tea.Cmd
	i.Model, cmd = i.Model.Update(msg)
	return i, cmd
}

func (i *Input) Style(style lipgloss.Style) {
	i.PromptStyle = style
}

type Button struct {
	style   lipgloss.Style
	Label   string
	focused bool
}

func (b *Button) Style(style lipgloss.Style) {
	b.style = style
}

func (b *Button) Reset() {
	b.Blur()
}

func (b *Button) Blur() {
	b.focused = false
}

func (b *Button) Focus() tea.Cmd {
	b.focused = true
	return nil
}

func (b *Button) Focused() bool {
	return b.focused
}

func (b *Button) View() string {
	return b.style.Render("[", b.Label, "]")
}

func (b *Button) error() string {
	return ""
}

func (b *Button) Update(tea.Msg) (FocusBlurViewer, tea.Cmd) {
	return b, nil
}

func (b *Button) Value() string {
	return ""
}

func NewInputs() []FocusBlurViewer {
	inputs := make([]FocusBlurViewer, cancel+1)

	sys := Input{textinput.New()}
	// TODO: Should we go with full suggestions instead? These aren't really visible most of the time. But full suggestions makes parsing more difficult
	// sys.SetSuggestions([]string{models.GB.String(), models.GBC.String(), models.GBA.String(), models.GG.String(), models.SMS.String(), models.NGP.String(), models.NGPC.String(), models.PCE.String(), models.Lynx.String()})
	// sys.ShowSuggestions = true
	sys.Prompt = "System: "
	sys.Placeholder = models.GB.String()
	sys.Validate = sysValidate
	sys.PromptStyle = itemStyle
	sys.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	sys.TextStyle = itemStyle.PaddingLeft(2)
	inputs[system] = &sys

	n := Input{textinput.New()}
	n.Prompt = "Name: "
	n.Validate = notBlank
	n.PromptStyle = itemStyle
	n.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	n.TextStyle = itemStyle.PaddingLeft(2)
	inputs[name] = &n

	c := Input{textinput.New()}
	c.Prompt = "CRC32: "
	c.Placeholder = "0x00000000"
	c.Validate = hexValidate
	c.PromptStyle = itemStyle
	c.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	c.TextStyle = itemStyle.PaddingLeft(2)
	inputs[crc] = &c

	s := Input{textinput.New()}
	s.Prompt = "Signature: "
	s.Placeholder = "0x00000000"
	s.Validate = hexValidate
	s.PromptStyle = itemStyle
	s.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	s.TextStyle = itemStyle.PaddingLeft(2)
	inputs[sig] = &s

	m := Input{textinput.New()}
	m.Prompt = "Magic Number: "
	m.Placeholder = "0x0000"
	m.Validate = hexValidate
	m.PromptStyle = itemStyle
	m.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	m.TextStyle = itemStyle.PaddingLeft(2)
	inputs[magic] = &m

	a := Input{textinput.New()}
	a.Prompt = "Date Added: "
	a.Placeholder = "2024-01-15 13:24" // Will be replaced eventually
	a.Validate = dateValidate
	a.PromptStyle = itemStyle
	a.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	a.TextStyle = itemStyle.PaddingLeft(2)
	inputs[added] = &a

	p := Input{textinput.New()}
	p.Prompt = "Played: "
	p.Placeholder = "0h 0m 0s"
	p.Validate = playValidate
	p.PromptStyle = itemStyle
	p.Cursor.Style = selectedItemStyle.PaddingLeft(0)
	p.TextStyle = itemStyle.PaddingLeft(2)
	inputs[play] = &p

	inputs[submit] = &Button{
		Label: "Submit",
		style: itemStyle,
	}
	inputs[cancel] = &Button{
		Label: "Cancel",
		style: itemStyle,
	}

	return inputs
}

func sysValidate(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil // Go with "GB" for a blank entry
	}

	if _, err := models.Parse(s); err != nil {
		return fmt.Errorf("System must be one of: %s, %s, %s, %s, %s, %s, %s, %s, %s", models.GB, models.GBC, models.GBA, models.GG, models.SMS, models.NGP, models.NGPC, models.PCE, models.Lynx)
	}
	return nil
}

func notBlank(s string) error {
	if len(strings.TrimSpace(s)) == 0 {
		return fmt.Errorf("Value cannot be blank")
	}
	return nil
}

func hexValidate(s string) error {
	if _, err := util.HexStringTransform(s); err != nil {
		return fmt.Errorf("%s is not a valid hex string", s)
	}
	return nil
}

func dateValidate(s string) error {
	s = strings.TrimSpace(s)
	if s == "" { // If it's blank, we go with today's date
		return nil
	}

	if _, err := parseDate(s); err != nil {
		return fmt.Errorf("Date must be in the format yyyy-MM-dd HH:mm")
	}

	return nil
}

// parseDate does no validation as it assumes that dateValidate has already been run on the value
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		s = time.Now().Format("2006-01-02 15:04")
	}

	dates := []string{"2006-01-02", "2006/01/02", "2006-1-2", "2006/1/2"} // Not doing or mm-dd-yyyy or any other incorrect format
	times := []string{"15:04", "15:04:05", "03:04PM", "03:04 PM", "3:04PM", "3:04 PM",
		"03:04:05PM", "03:04:05 PM", "3:04:05PM", "3:04:05 PM"}

	for _, d := range dates {
		for _, t := range times {
			if result, err := time.Parse(fmt.Sprintf("%s %s", d, t), s); err == nil {
				return result, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("could not parse string: %s", s)
}

// parsePlayTime does no validation as it assumes that playValidate has already been run on the value
func parsePlayTime(s string) uint32 {
	s = strings.ReplaceAll(s, " ", "") // Remove all spaces, since we're allowing 0h0m0s, 0 h 0m 0s, 0h 0m 0s, etc
	if s == "" {
		return 0
	}

	if i, err := strconv.Atoi(s); err == nil {
		return uint32(i)
	}

	if !playRegex.MatchString(s) { // Doesn't match, don't return
		return 0
	}

	var total uint32
	match := playRegex.FindStringSubmatch(s)
	for i, name := range playRegex.SubexpNames() {
		switch name {
		case "hours":
			v, _ := strconv.Atoi(match[i])
			total = total + uint32(3600*v)
		case "minutes":
			v, _ := strconv.Atoi(match[i])
			total = total + uint32(60*v)
		case "seconds":
			v, _ := strconv.Atoi(match[i])
			total = total + uint32(v)
		}
	}
	return total
}

func playValidate(s string) error {
	s = strings.ReplaceAll(s, " ", "")
	if s == "" { // If it's blank, we go with all 0s
		return nil
	}

	// We're going to allow just undifferentiated int values as seconds, because why not
	// But it still has to be 0 or greater
	if i, err := strconv.Atoi(s); err == nil && i < 0 {
		return fmt.Errorf("Play time cannot be a negative value")
	} else if err == nil {
		return nil
	}

	if !playRegex.MatchString(s) {
		return fmt.Errorf("Play time should be in the form: 0h 0m 0s")
	}

	return nil
}
