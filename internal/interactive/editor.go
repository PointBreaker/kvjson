// Package interactive provides kv's inline terminal editor.
package interactive

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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
		rows:        []row{newRow(28)},
		activeField: keyField,
		width:       80,
		height:      24,
	}
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
	return textinput.Blink
}

func (m editorModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
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

	var output strings.Builder
	output.WriteString("kv · interactive JSON builder\n")

	start, end := m.visibleRows()
	if start > 0 {
		output.WriteString("  …\n")
	}
	for index := start; index < end; index++ {
		marker := " "
		if index == m.activeRow {
			marker = "›"
		}
		fmt.Fprintf(
			&output,
			"%s %d  key [%s] │ value [%s]\n",
			marker,
			index+1,
			m.rows[index].key.View(),
			m.rows[index].value.View(),
		)
	}
	if end < len(m.rows) {
		output.WriteString("  …\n")
	}

	if m.errText != "" {
		fmt.Fprintf(&output, "! %s\n", m.errText)
	}
	output.WriteString("tab/⇧tab fields · ←/→ boundaries · ↑/↓ rows · enter next/finish · ctrl+d done · esc cancel")

	return tea.NewView(output.String())
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

func (m editorModel) inputWidth() int {
	width := (m.width - 27) / 2
	if width < 8 {
		return 8
	}
	if width > 36 {
		return 36
	}
	return width
}

func (m editorModel) visibleRows() (int, int) {
	maxRows := m.height - 5
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
