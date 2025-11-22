package smidgen

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen/internal/action"
	"github.com/sedwards2009/smidgen/internal/buffer"
	"github.com/sedwards2009/smidgen/internal/config"
	"github.com/sedwards2009/smidgen/internal/display"
	"github.com/sedwards2009/smidgen/runtime"
)

type Buffer struct {
	*buffer.Buffer
}

type Cursor struct {
	*buffer.Cursor
}

type View struct {
	*tview.Box
	buffer    *buffer.Buffer
	bufWindow *display.BufWindow
	bufPane   *action.BufPane
}

type ActionController struct {
	*action.BufPane
}

func NewView(app *tview.Application, buffer *Buffer) *View {
	v := &View{
		Box:       tview.NewBox(),
		buffer:    buffer.Buffer,
		bufWindow: display.NewBufWindow(0, 0, 10, 10, buffer.Buffer),
	}

	buffer.RegisterRedrawCallback(func() {
		app.QueueUpdateDraw(func() {
			// Just trigger a redraw of the buffer window
		})
	})

	v.bufPane = action.NewBufPane(v.buffer, v.bufWindow)
	return v
}

func (v *View) Draw(screen tcell.Screen) {
	v.Box.Draw(screen)

	innerX, innerY, width, height := v.GetInnerRect()
	v.bufWindow.X = innerX
	v.bufWindow.Y = innerY
	v.bufWindow.Resize(width, height)
	v.bufWindow.Display(screen)
}

func (v *View) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return v.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		v.bufPane.HandleEvent(event)
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (v *View) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return v.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		x, y := event.Position()
		if !v.InRect(x, y) {
			return false, nil
		}
		v.bufPane.HandleEvent(event)
		return true, nil
	})
}

func (v *View) SetColorscheme(cs Colorscheme) {
	v.bufWindow.Colorscheme = config.Colorscheme(cs)
	v.buffer.UpdateRules()
}

func (v *View) Buffer() *Buffer {
	return &Buffer{v.buffer}
}

func (v *View) Cursor() *Cursor {
	return &Cursor{v.bufPane.Cursor}
}

func (v *View) Relocate() {
	v.bufWindow.Relocate()
}

func (v *View) ActionController() *ActionController {
	return &ActionController{v.bufPane}
}

type Keybindings struct {
	*action.KeyTree
}

func ParseKeybindings(config map[string]string) Keybindings {
	return Keybindings{action.BindingMappingToKeyTree(config)}
}

func (v *View) SetKeybindings(keybindings Keybindings) {
	v.bufPane.SetBindings(keybindings.KeyTree)
}

func NewBufferFromString(content string, path string) *Buffer {
	return &Buffer{
		Buffer: buffer.NewBufferFromString(content, path),
	}
}

type Colorscheme config.Colorscheme

func (colorscheme Colorscheme) GetColor(color string) tcell.Style {
	return config.Colorscheme(colorscheme).GetColor(color)
}

func LoadInternalColorscheme(name string) (Colorscheme, bool) {
	data, err := runtime.Asset("runtime/colorschemes/" + name + ".micro")
	if err != nil {
		return nil, false
	}
	return ParseColorscheme(string(data)), true
}

func ParseColorscheme(data string) Colorscheme {
	return Colorscheme(config.ParseColorscheme(data))
}

func init() {
	config.InitRuntimeFiles()
}

type Action func() bool

func (v *View) MapActionNameToAction(name string) Action {
	if f, ok := action.BufKeyActions[name]; ok {
		return func() bool {
			return f(v.bufPane)
		}
	}
	return nil
}

// Actions
const (
	ActionCursorUp               = "CursorUp"
	ActionCursorDown             = "CursorDown"
	ActionCursorPageUp           = "CursorPageUp"
	ActionCursorPageDown         = "CursorPageDown"
	ActionCursorLeft             = "CursorLeft"
	ActionCursorRight            = "CursorRight"
	ActionCursorStart            = "CursorStart"
	ActionCursorEnd              = "CursorEnd"
	ActionSelectToStart          = "SelectToStart"
	ActionSelectToEnd            = "SelectToEnd"
	ActionSelectUp               = "SelectUp"
	ActionSelectDown             = "SelectDown"
	ActionSelectLeft             = "SelectLeft"
	ActionSelectRight            = "SelectRight"
	ActionWordRight              = "WordRight"
	ActionWordLeft               = "WordLeft"
	ActionSelectWordRight        = "SelectWordRight"
	ActionSelectWordLeft         = "SelectWordLeft"
	ActionDeleteWordRight        = "DeleteWordRight"
	ActionDeleteWordLeft         = "DeleteWordLeft"
	ActionSelectLine             = "SelectLine"
	ActionSelectToStartOfLine    = "SelectToStartOfLine"
	ActionSelectToEndOfLine      = "SelectToEndOfLine"
	ActionParagraphPrevious      = "ParagraphPrevious"
	ActionParagraphNext          = "ParagraphNext"
	ActionInsertNewline          = "InsertNewline"
	ActionInsertSpace            = "InsertSpace"
	ActionBackspace              = "Backspace"
	ActionDelete                 = "Delete"
	ActionInsertTab              = "InsertTab"
	ActionCenter                 = "Center"
	ActionUndo                   = "Undo"
	ActionRedo                   = "Redo"
	ActionCopy                   = "Copy"
	ActionCut                    = "Cut"
	ActionCutLine                = "CutLine"
	ActionDuplicateLine          = "DuplicateLine"
	ActionDeleteLine             = "DeleteLine"
	ActionMoveLinesUp            = "MoveLinesUp"
	ActionMoveLinesDown          = "MoveLinesDown"
	ActionIndentSelection        = "IndentSelection"
	ActionOutdentSelection       = "OutdentSelection"
	ActionOutdentLine            = "OutdentLine"
	ActionPaste                  = "Paste"
	ActionSelectAll              = "SelectAll"
	ActionStart                  = "Start"
	ActionEnd                    = "End"
	ActionPageUp                 = "PageUp"
	ActionPageDown               = "PageDown"
	ActionSelectPageUp           = "SelectPageUp"
	ActionSelectPageDown         = "SelectPageDown"
	ActionHalfPageUp             = "HalfPageUp"
	ActionHalfPageDown           = "HalfPageDown"
	ActionStartOfLine            = "StartOfLine"
	ActionEndOfLine              = "EndOfLine"
	ActionToggleRuler            = "ToggleRuler"
	ActionToggleOverwriteMode    = "ToggleOverwriteMode"
	ActionEscape                 = "Escape"
	ActionScrollUp               = "ScrollUp"
	ActionScrollDown             = "ScrollDown"
	ActionSpawnMultiCursor       = "SpawnMultiCursor"
	ActionSpawnMultiCursorSelect = "SpawnMultiCursorSelect"
	ActionRemoveMultiCursor      = "RemoveMultiCursor"
	ActionRemoveAllMultiCursors  = "RemoveAllMultiCursors"
	ActionSkipMultiCursor        = "SkipMultiCursor"
	ActionJumpToMatchingBrace    = "JumpToMatchingBrace"
	ActionInsertEnter            = "InsertEnter"
	ActionUnbindKey              = "UnbindKey"
)

type KeyDesc struct {
	KeyCode   tcell.Key
	Modifiers tcell.ModMask
	R         rune
}

// Utility function for parsing key sequences which have the same format as Smidgen keybindings
func ParseKeySequence(k string) (KeyDesc, bool) {
	kd, ok := action.ParseKeyboardSequence(k)
	if !ok {
		return KeyDesc{}, false
	}
	return KeyDesc{
		KeyCode:   kd.KeyCode,
		Modifiers: kd.Modifiers,
		R:         kd.R,
	}, true
}
