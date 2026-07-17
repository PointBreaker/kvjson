// Package interactive provides kv's inline terminal editor.
package interactive

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ErrCanceled is returned when the user cancels interactive input.
var ErrCanceled = errors.New("interactive input canceled")

type field uint8

const (
	keyField field = iota
	valueField
)

type row struct {
	key   textinput.Model
	value textinput.Model
}

type editorModel struct {
	rows        []row
	activeRow   int
	activeField field
	width       int
	height      int
	errText     string
	done        bool
	canceled    bool
	values      map[string]string
	isDark      bool
	styles      editorStyles
}

type editorStyles struct {
	title            lipgloss.Style
	subtitle         lipgloss.Style
	cardActive       lipgloss.Style
	cardComplete     lipgloss.Style
	cardIdle         lipgloss.Style
	cardError        lipgloss.Style
	badgeActive      lipgloss.Style
	badgeComplete    lipgloss.Style
	badgeIdle        lipgloss.Style
	badgeError       lipgloss.Style
	keyLabel         lipgloss.Style
	valueLabel       lipgloss.Style
	inputIdle        lipgloss.Style
	inputKeyActive   lipgloss.Style
	inputValueActive lipgloss.Style
	helpKey          lipgloss.Style
	helpText         lipgloss.Style
	errorBadge       lipgloss.Style
	errorText        lipgloss.Style
	ellipsis         lipgloss.Style
}

// Run opens an inline editor on the controlling terminal and returns its
// collected values. The terminal UI never uses stdout, so stdout remains safe
// for the generated JSON even when kv is used in command substitution.
func Run() (map[string]string, error) {
	input, output, err := tea.OpenTTY()
	if err != nil {
		return nil, fmt.Errorf("interactive mode requires a terminal: %w", err)
	}
	defer input.Close()
	defer output.Close()

	program := tea.NewProgram(
		newEditorModel(),
		tea.WithInput(input),
		tea.WithOutput(output),
	)

	finalModel, err := program.Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return nil, ErrCanceled
	}
	if err != nil {
		return nil, err
	}

	model, ok := finalModel.(editorModel)
	if !ok {
		return nil, errors.New("interactive editor returned an unexpected model")
	}
	if model.canceled {
		return nil, ErrCanceled
	}
	if !model.done {
		return nil, errors.New("interactive editor closed before generating JSON")
	}
	return model.values, nil
}

func newEditorModel() editorModel {
	model := editorModel{
		activeField: keyField,
		width:       80,
		height:      24,
		isDark:      true,
		styles:      newEditorStyles(true),
	}
	model.rows = []row{newRow(model.inputWidth())}
	model.applyInputStyles()
	model.focus(0, keyField)
	return model
}

func newRow(width int) row {
	key := textinput.New()
	key.Prompt = ""
	key.Placeholder = "key"
	key.SetWidth(width)

	value := textinput.New()
	value.Prompt = ""
	value.Placeholder = "value"
	value.SetWidth(width)

	return row{key: key, value: value}
}

func (m editorModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.RequestBackgroundColor)
}

func (m editorModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.BackgroundColorMsg:
		m.isDark = message.IsDark()
		m.styles = newEditorStyles(m.isDark)
		m.applyInputStyles()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.resizeInputs()
		return m, nil

	case tea.InterruptMsg:
		m.canceled = true
		return m, tea.Quit

	case tea.KeyPressMsg:
		switch message.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "ctrl+d":
			return m.finish()
		case "enter":
			return m.advance()
		case "tab":
			return m, m.moveForward()
		case "shift+tab":
			return m, m.moveBackward()
		case "up":
			if m.activeRow > 0 {
				return m, m.focus(m.activeRow-1, m.activeField)
			}
		case "down":
			if m.activeRow+1 < len(m.rows) {
				return m, m.focus(m.activeRow+1, m.activeField)
			}
		case "right":
			if m.activeInput().Position() == utf8.RuneCountInString(m.activeInput().Value()) {
				return m, m.moveForward()
			}
		case "left":
			if m.activeInput().Position() == 0 {
				return m, m.moveBackward()
			}
		}
	}

	m.errText = ""
	input := m.activeInput()
	updated, command := input.Update(message)
	*input = updated
	return m, command
}

func (m editorModel) View() tea.View {
	if m.done || m.canceled {
		return tea.NewView("")
	}

	var sections []string
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.styles.title.Render("KVJSON"),
		" ",
		m.styles.subtitle.Render("interactive builder"),
	)
	sections = append(sections, header)

	start, end := m.visibleRows()
	var cards []string
	if start > 0 {
		cards = append(cards, m.styles.ellipsis.Render("… earlier rows"))
	}
	for index := start; index < end; index++ {
		cards = append(cards, m.renderRow(index))
	}
	if end < len(m.rows) {
		cards = append(cards, m.styles.ellipsis.Render("more rows …"))
	}
	sections = append(sections, strings.Join(cards, "\n\n"))

	if m.errText != "" {
		sections = append(sections, lipgloss.JoinHorizontal(
			lipgloss.Center,
			m.styles.errorBadge.Render("ERROR"),
			" ",
			m.styles.errorText.Render(m.errText),
		))
	}
	sections = append(sections, m.renderHelp())

	return tea.NewView(strings.Join(sections, "\n\n"))
}

func (m editorModel) renderRow(index int) string {
	current := m.rows[index]
	card := m.styles.cardIdle
	badge := m.styles.badgeIdle

	if index == m.activeRow {
		card = m.styles.cardActive
		badge = m.styles.badgeActive
		if m.errText != "" {
			card = m.styles.cardError
			badge = m.styles.badgeError
		}
	} else if strings.TrimSpace(current.key.Value()) != "" {
		card = m.styles.cardComplete
		badge = m.styles.badgeComplete
	}

	keyInput := m.styles.inputIdle
	valueInput := m.styles.inputIdle
	if index == m.activeRow && m.activeField == keyField {
		keyInput = m.styles.inputKeyActive
	}
	if index == m.activeRow && m.activeField == valueField {
		valueInput = m.styles.inputValueActive
	}

	content := lipgloss.JoinHorizontal(
		lipgloss.Center,
		badge.Render(fmt.Sprintf("%02d", index+1)),
		" ",
		m.styles.keyLabel.Render("KEY"),
		" ",
		keyInput.Render(current.key.View()),
		"  ",
		m.styles.valueLabel.Render("VALUE"),
		" ",
		valueInput.Render(current.value.View()),
	)
	return card.Render(content)
}

func (m editorModel) renderHelp() string {
	firstLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.styles.helpKey.Render("TAB"), m.styles.helpText.Render(" fields  "),
		m.styles.helpKey.Render("ARROWS"), m.styles.helpText.Render(" navigate  "),
		m.styles.helpKey.Render("ENTER"), m.styles.helpText.Render(" next / finish"),
	)
	secondLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.styles.helpKey.Render("CTRL+D"), m.styles.helpText.Render(" generate  "),
		m.styles.helpKey.Render("ESC"), m.styles.helpText.Render(" cancel"),
	)
	return lipgloss.JoinVertical(lipgloss.Left, firstLine, secondLine)
}

func (m *editorModel) advance() (tea.Model, tea.Cmd) {
	current := m.rows[m.activeRow]
	key := strings.TrimSpace(current.key.Value())
	value := current.value.Value()

	if key == "" && strings.TrimSpace(value) == "" {
		return m.finish()
	}
	if key == "" {
		m.errText = fmt.Sprintf("row %d needs a key", m.activeRow+1)
		return *m, m.focus(m.activeRow, keyField)
	}
	if duplicateRow := m.duplicateRow(key, m.activeRow); duplicateRow >= 0 {
		m.errText = fmt.Sprintf("key %q already exists in row %d", key, duplicateRow+1)
		return *m, m.focus(m.activeRow, keyField)
	}

	m.errText = ""
	if m.activeRow == len(m.rows)-1 {
		m.rows = append(m.rows, newRow(m.inputWidth()))
	}
	return *m, m.focus(m.activeRow+1, keyField)
}

func (m *editorModel) finish() (tea.Model, tea.Cmd) {
	values := make(map[string]string)
	rowsByKey := make(map[string]int)

	for index, row := range m.rows {
		key := strings.TrimSpace(row.key.Value())
		value := row.value.Value()
		if key == "" && strings.TrimSpace(value) == "" {
			continue
		}
		if key == "" {
			m.errText = fmt.Sprintf("row %d needs a key", index+1)
			return *m, m.focus(index, keyField)
		}
		if existingRow, exists := rowsByKey[key]; exists {
			m.errText = fmt.Sprintf("key %q already exists in row %d", key, existingRow+1)
			return *m, m.focus(index, keyField)
		}
		rowsByKey[key] = index
		values[key] = value
	}

	m.values = values
	m.done = true
	m.errText = ""
	return *m, tea.Quit
}

func (m *editorModel) duplicateRow(key string, except int) int {
	for index, row := range m.rows {
		if index != except && strings.TrimSpace(row.key.Value()) == key {
			return index
		}
	}
	return -1
}

func (m *editorModel) moveForward() tea.Cmd {
	if m.activeField == keyField {
		command := m.focus(m.activeRow, valueField)
		m.activeInput().CursorStart()
		return command
	}
	if m.activeRow+1 < len(m.rows) {
		command := m.focus(m.activeRow+1, keyField)
		m.activeInput().CursorStart()
		return command
	}
	command := m.focus(m.activeRow, keyField)
	m.activeInput().CursorStart()
	return command
}

func (m *editorModel) moveBackward() tea.Cmd {
	if m.activeField == valueField {
		command := m.focus(m.activeRow, keyField)
		m.activeInput().CursorEnd()
		return command
	}
	if m.activeRow > 0 {
		command := m.focus(m.activeRow-1, valueField)
		m.activeInput().CursorEnd()
		return command
	}
	command := m.focus(m.activeRow, valueField)
	m.activeInput().CursorEnd()
	return command
}

func (m *editorModel) focus(rowIndex int, focusedField field) tea.Cmd {
	for index := range m.rows {
		m.rows[index].key.Blur()
		m.rows[index].value.Blur()
	}
	m.activeRow = rowIndex
	m.activeField = focusedField
	return m.activeInput().Focus()
}

func (m *editorModel) activeInput() *textinput.Model {
	if m.activeField == valueField {
		return &m.rows[m.activeRow].value
	}
	return &m.rows[m.activeRow].key
}

func (m *editorModel) resizeInputs() {
	width := m.inputWidth()
	for index := range m.rows {
		m.rows[index].key.SetWidth(width)
		m.rows[index].value.SetWidth(width)
	}
}

func (m *editorModel) applyInputStyles() {
	styles := textinput.DefaultStyles(m.isDark)
	for index := range m.rows {
		m.rows[index].key.SetStyles(styles)
		m.rows[index].value.SetStyles(styles)
	}
}

func (m editorModel) inputWidth() int {
	width := (m.width - 31) / 2
	if width < 8 {
		return 8
	}
	if width > 36 {
		return 36
	}
	return width
}

func (m editorModel) visibleRows() (int, int) {
	maxRows := (m.height - 7) / 4
	if maxRows < 1 {
		maxRows = 1
	}
	if maxRows >= len(m.rows) {
		return 0, len(m.rows)
	}

	start := m.activeRow - maxRows + 1
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > len(m.rows) {
		end = len(m.rows)
		start = end - maxRows
	}
	return start, end
}

func newEditorStyles(isDark bool) editorStyles {
	pick := lipgloss.LightDark(isDark)
	primary := pick(lipgloss.Color("#6D28D9"), lipgloss.Color("#A78BFA"))
	primaryFill := pick(lipgloss.Color("#6D28D9"), lipgloss.Color("#7C3AED"))
	keyFill := pick(lipgloss.Color("#0E7490"), lipgloss.Color("#22D3EE"))
	keyForeground := pick(lipgloss.Color("#FFFFFF"), lipgloss.Color("#083344"))
	keyActive := pick(lipgloss.Color("#CFFAFE"), lipgloss.Color("#164E63"))
	valueFill := pick(lipgloss.Color("#BE185D"), lipgloss.Color("#F472B6"))
	valueForeground := pick(lipgloss.Color("#FFFFFF"), lipgloss.Color("#500724"))
	valueActive := pick(lipgloss.Color("#FCE7F3"), lipgloss.Color("#831843"))
	success := pick(lipgloss.Color("#047857"), lipgloss.Color("#34D399"))
	successFill := pick(lipgloss.Color("#047857"), lipgloss.Color("#059669"))
	danger := pick(lipgloss.Color("#E11D48"), lipgloss.Color("#FB7185"))
	dangerFill := pick(lipgloss.Color("#BE123C"), lipgloss.Color("#E11D48"))
	text := pick(lipgloss.Color("#0F172A"), lipgloss.Color("#F8FAFC"))
	muted := pick(lipgloss.Color("#64748B"), lipgloss.Color("#94A3B8"))
	mutedFill := pick(lipgloss.Color("#CBD5E1"), lipgloss.Color("#475569"))
	panel := pick(lipgloss.Color("#F8FAFC"), lipgloss.Color("#0F172A"))
	input := pick(lipgloss.Color("#E2E8F0"), lipgloss.Color("#1E293B"))
	white := lipgloss.Color("#FFFFFF")

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Background(panel)
	badgeBase := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	labelBase := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	inputBase := lipgloss.NewStyle().Foreground(text).Padding(0, 1)

	return editorStyles{
		title:            lipgloss.NewStyle().Bold(true).Foreground(white).Background(primaryFill).Padding(0, 1),
		subtitle:         lipgloss.NewStyle().Foreground(muted),
		cardActive:       card.BorderForeground(primary),
		cardComplete:     card.BorderForeground(success),
		cardIdle:         card.BorderForeground(mutedFill),
		cardError:        card.BorderForeground(danger),
		badgeActive:      badgeBase.Foreground(white).Background(primaryFill),
		badgeComplete:    badgeBase.Foreground(white).Background(successFill),
		badgeIdle:        badgeBase.Foreground(text).Background(mutedFill),
		badgeError:       badgeBase.Foreground(white).Background(dangerFill),
		keyLabel:         labelBase.Foreground(keyForeground).Background(keyFill),
		valueLabel:       labelBase.Foreground(valueForeground).Background(valueFill),
		inputIdle:        inputBase.Background(input),
		inputKeyActive:   inputBase.Background(keyActive),
		inputValueActive: inputBase.Background(valueActive),
		helpKey:          lipgloss.NewStyle().Bold(true).Foreground(primary).Background(input).Padding(0, 1),
		helpText:         lipgloss.NewStyle().Foreground(muted),
		errorBadge:       lipgloss.NewStyle().Bold(true).Foreground(white).Background(dangerFill).Padding(0, 1),
		errorText:        lipgloss.NewStyle().Foreground(danger),
		ellipsis:         lipgloss.NewStyle().Foreground(muted).Italic(true),
	}
}
