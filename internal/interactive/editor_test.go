package interactive

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestEditorAddsRowsAndFinishes(t *testing.T) {
	model := newEditorModel()
	model = typeText(t, model, "name")
	model = press(t, model, tea.KeyTab, "")
	model = typeText(t, model, "Alice")
	model = press(t, model, tea.KeyEnter, "")

	if len(model.rows) != 2 || model.activeRow != 1 || model.activeField != keyField {
		t.Fatalf("after enter: rows=%d activeRow=%d activeField=%d", len(model.rows), model.activeRow, model.activeField)
	}

	model = typeText(t, model, "role")
	model = press(t, model, tea.KeyTab, "")
	model = typeText(t, model, "admin")
	model = pressKey(t, model, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})

	want := map[string]string{"name": "Alice", "role": "admin"}
	if !model.done || !reflect.DeepEqual(model.values, want) {
		t.Fatalf("finished model: done=%v values=%#v, want %#v", model.done, model.values, want)
	}
}

func TestEditorArrowKeysCrossFieldBoundaries(t *testing.T) {
	model := newEditorModel()
	model = typeText(t, model, "ab")
	model = press(t, model, tea.KeyRight, "")
	if model.activeField != valueField {
		t.Fatalf("right at key boundary activeField=%d, want valueField", model.activeField)
	}

	model = press(t, model, tea.KeyLeft, "")
	if model.activeField != keyField || model.activeInput().Position() != 2 {
		t.Fatalf("left at value boundary activeField=%d position=%d", model.activeField, model.activeInput().Position())
	}
}

func TestEditorVerticalNavigationAndCancel(t *testing.T) {
	model := newEditorModel()
	model = typeText(t, model, "empty")
	model = press(t, model, tea.KeyEnter, "")

	model = press(t, model, tea.KeyUp, "")
	if model.activeRow != 0 || model.activeField != keyField {
		t.Fatalf("up: activeRow=%d activeField=%d", model.activeRow, model.activeField)
	}

	model = press(t, model, tea.KeyDown, "")
	if model.activeRow != 1 || model.activeField != keyField {
		t.Fatalf("down: activeRow=%d activeField=%d", model.activeRow, model.activeField)
	}

	model = press(t, model, tea.KeyEscape, "")
	if !model.canceled || model.done {
		t.Fatalf("escape: canceled=%v done=%v", model.canceled, model.done)
	}
}

func TestEditorRejectsDuplicateKeys(t *testing.T) {
	model := newEditorModel()
	model = typeText(t, model, "a")
	model = press(t, model, tea.KeyTab, "")
	model = typeText(t, model, "one")
	model = press(t, model, tea.KeyEnter, "")
	model = typeText(t, model, "a")
	model = press(t, model, tea.KeyTab, "")
	model = typeText(t, model, "two")
	model = press(t, model, tea.KeyEnter, "")

	if model.done || !strings.Contains(model.errText, "already exists") {
		t.Fatalf("duplicate result: done=%v errText=%q", model.done, model.errText)
	}
}

func TestEditorBlankRowFinishes(t *testing.T) {
	model := newEditorModel()
	model = press(t, model, tea.KeyEnter, "")
	if !model.done || len(model.values) != 0 {
		t.Fatalf("blank enter: done=%v values=%#v", model.done, model.values)
	}
}

func TestEditorViewIsInline(t *testing.T) {
	model := newEditorModel()
	view := model.View()
	if view.AltScreen {
		t.Fatal("editor should not use the alternate screen")
	}
	if !strings.Contains(view.Content, "interactive JSON builder") || !strings.Contains(view.Content, "ctrl+d done") {
		t.Fatalf("unexpected view: %q", view.Content)
	}
}

func typeText(t *testing.T, model editorModel, value string) editorModel {
	t.Helper()
	for _, character := range value {
		model = press(t, model, character, string(character))
	}
	return model
}

func press(t *testing.T, model editorModel, code rune, text string) editorModel {
	t.Helper()
	return pressKey(t, model, tea.KeyPressMsg{Code: code, Text: text})
}

func pressKey(t *testing.T, model editorModel, message tea.KeyPressMsg) editorModel {
	t.Helper()
	updated, _ := model.Update(message)
	result, ok := updated.(editorModel)
	if !ok {
		t.Fatalf("Update() returned %T, want editorModel", updated)
	}
	return result
}
