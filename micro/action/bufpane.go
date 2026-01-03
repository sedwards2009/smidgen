package action

import (
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/sedwards2009/smidgen/micro/buffer"
	"github.com/sedwards2009/smidgen/micro/display"
	"github.com/sedwards2009/smidgen/micro/util"
)

type BufAction any

// BufKeyAction represents an action bound to a key.
type BufKeyAction func(*BufPane) bool

// BufMouseAction is an action that must be bound to a mouse event.
type BufMouseAction func(*BufPane, *tcell.EventMouse, func()) bool

// BufBindings stores the bindings for the buffer pane type.
var BufBindings *KeyTree

// BufKeyActionGeneral makes a general pane action from a BufKeyAction.
func BufKeyActionGeneral(a BufKeyAction) PaneKeyAction {
	return func(p Pane, takeFocus func()) bool {
		return a(p.(*BufPane))
	}
}

// BufMouseActionGeneral makes a general pane mouse action from a BufKeyAction.
func BufMouseActionGeneral(a BufMouseAction) PaneMouseAction {
	return func(p Pane, me *tcell.EventMouse, takeFocus func()) bool {
		return a(p.(*BufPane), me, takeFocus)
	}
}

func init() {
	BufBindings = BindingMappingToKeyTree(bufdefaults)
}

// BufMapEvent maps an event to an action
func BufMapEvent(bufBindings *KeyTree, k Event, action string) {
	var actionfns []BufAction
	var names []string
	var types []byte
	for i := 0; ; i++ {
		if action == "" {
			break
		}

		idx := util.IndexAnyUnquoted(action, "&|,")
		a := action
		if idx >= 0 {
			a = action[:idx]
			types = append(types, action[idx])
			action = action[idx+1:]
		} else {
			types = append(types, ' ')
			action = ""
		}

		var afn BufAction
		if f, ok := BufKeyActions[a]; ok {
			afn = f
			names = append(names, a)
		} else if f, ok := BufMouseActions[a]; ok {
			afn = f
			names = append(names, a)
		} else {
			log.Printf("Error in bindings: action %s does not exist", a)
			continue
		}
		actionfns = append(actionfns, afn)
	}
	bufAction := func(h *BufPane, te *tcell.EventMouse, takeFocus func()) bool {
		for i, a := range actionfns {
			var success bool
			if _, ok := MultiActions[names[i]]; ok {
				success = true
				for _, c := range h.Buf.GetCursors() {
					h.Buf.SetCurCursor(c.Num)
					h.Cursor = c
					success = success && h.execAction(a, names[i], te, takeFocus)
				}
			} else {
				h.Buf.SetCurCursor(0)
				h.Cursor = h.Buf.GetActiveCursor()
				success = h.execAction(a, names[i], te, takeFocus)
			}

			if (!success && types[i] == '&') || (success && types[i] == '|') {
				break
			}
		}
		return true
	}

	switch e := k.(type) {
	case KeyEvent, KeySequenceEvent, RawEvent:
		bufBindings.RegisterKeyBinding(e, BufKeyActionGeneral(func(h *BufPane) bool {
			return bufAction(h, nil, func() {})
		}))
	case MouseEvent:
		bufBindings.RegisterMouseBinding(e, BufMouseActionGeneral(bufAction))
	}
}

// The BufPane connects the buffer and the window
// It provides a cursor (or multiple) and defines a set of actions
// that can be taken on the buffer
// The ActionHandler can access the window for necessary info about
// visual positions for mouse clicks and scrolling
type BufPane struct {
	display.BWindow

	// Buf is the buffer this BufPane views
	Buf *buffer.Buffer
	// Bindings stores the association of key events and actions
	bindings *KeyTree

	// Cursor is the currently active buffer cursor
	Cursor *buffer.Cursor

	// Since tcell doesn't differentiate between a mouse press event
	// and a mouse move event with button pressed (nor between a mouse
	// release event and a mouse move event with no buttons pressed),
	// we need to keep track of whether or not the mouse was previously
	// pressed, to determine mouse release and mouse drag events.
	// Moreover, since in case of a release event tcell doesn't tell us
	// which button was released, we need to keep track of which
	// (possibly multiple) buttons were pressed previously.
	mousePressed map[MouseEvent]bool

	// This stores when the last click was
	// This is useful for detecting double and triple clicks
	lastClickTime time.Time
	lastLoc       buffer.Loc

	// freshClip returns true if one or more lines have been cut to the clipboard
	// and have never been pasted yet.
	freshClip bool

	// Was the last mouse event actually a double click?
	// Useful for detecting triple clicks -- if a double click is detected
	// but the last mouse event was actually a double click, it's a triple click
	DoubleClick bool
	// Same here, just to keep track for mouse move events
	TripleClick bool

	// Should the current multiple cursor selection search based on word or
	// based on selection (false for selection, true for word)
	multiWord bool

	// remember original location of a search in case the search is canceled
	searchOrig buffer.Loc

	// The pane may not yet be fully initialized after its creation
	// since we may not know the window geometry yet. In such case we finish
	// its initialization a bit later, after the initial resize.
	initialized bool
}

func newBufPane(buf *buffer.Buffer, win display.BWindow) *BufPane {
	h := new(BufPane)
	h.Buf = buf
	h.BWindow = win

	h.Cursor = h.Buf.GetActiveCursor()
	h.mousePressed = make(map[MouseEvent]bool)

	return h
}

// NewBufPane creates a new buffer pane with the given window.
func NewBufPane(buf *buffer.Buffer, win display.BWindow) *BufPane {
	h := newBufPane(buf, win)
	h.finishInitialize()
	return h
}

// NewBufPaneFromBuf constructs a new pane from the given buffer and automatically
// creates a buf window.
func NewBufPaneFromBuf(buf *buffer.Buffer) *BufPane {
	w := display.NewBufWindow(0, 0, 0, 0, buf)
	h := newBufPane(buf, w)
	// Postpone finishing initializing the pane until we know the actual geometry
	// of the buf window.
	return h
}

// TODO: make sure splitID and tab are set before finishInitialize is called
func (h *BufPane) finishInitialize() {
	h.initialRelocate()
	h.initialized = true
}

// Resize resizes the pane
func (h *BufPane) Resize(width, height int) {
	h.BWindow.Resize(width, height)
	if !h.initialized {
		h.finishInitialize()
	}
}

// GotoLoc moves the cursor to a new location and adjusts the view accordingly.
// Use GotoLoc when the new location may be far away from the current location.
func (h *BufPane) GotoLoc(loc buffer.Loc) {
	sloc := h.SLocFromLoc(loc)
	d := h.Diff(h.SLocFromLoc(h.Cursor.Loc), sloc)

	h.Cursor.GotoLoc(loc)

	// If the new location is far away from the previous one,
	// ensure the cursor is at 25% of the window height
	height := h.BufView().Height
	if util.Abs(d) >= height {
		v := h.GetView()
		v.StartLine = h.Scroll(sloc, -height/4)
		h.ScrollAdjust()
		v.StartCol = 0
	}
	h.Relocate()
}

func (h *BufPane) SetStartLine(sloc display.SLoc) {
	v := h.GetView()
	v.StartLine = sloc
}

func (h *BufPane) initialRelocate() {
	sloc := h.SLocFromLoc(h.Cursor.Loc)
	height := h.BufView().Height

	// If the initial cursor location is far away from the beginning
	// of the buffer, ensure the cursor is at 25% of the window height
	v := h.GetView()
	if h.Diff(display.SLoc{0, 0}, sloc) < height {
		v.StartLine = display.SLoc{0, 0}
	} else {
		v.StartLine = h.Scroll(sloc, -height/4)
		h.ScrollAdjust()
	}
	v.StartCol = 0
	h.Relocate()
}

// HandleEvent executes the tcell event properly
func (h *BufPane) HandleEvent(event tcell.Event, takeFocus func()) {
	switch e := event.(type) {
	// case *tcell.EventRaw:
	// 	re := RawEvent{
	// 		esc: e.EscSeq(),
	// 	}
	// 	h.DoKeyEvent(re)
	// case *tcell.EventPaste:
	// 	h.paste(e.Text())
	// 	h.Relocate()
	case *tcell.EventKey:
		ke := keyEvent(e)

		done := h.DoKeyEvent(ke, takeFocus)
		if !done && e.Key() == tcell.KeyRune {
			h.DoRuneInsert(e.Rune())
		}
	case *tcell.EventMouse:
		if e.Buttons() != tcell.ButtonNone {
			me := MouseEvent{
				btn:   e.Buttons(),
				mod:   metaToAlt(e.Modifiers()),
				state: MousePress,
			}
			isDrag := len(h.mousePressed) > 0

			if e.Buttons() & ^(tcell.WheelUp|tcell.WheelDown|tcell.WheelLeft|tcell.WheelRight) != tcell.ButtonNone {
				h.mousePressed[me] = true
			}

			if isDrag {
				me.state = MouseDrag
			}
			h.DoMouseEvent(me, e, takeFocus)
		} else {
			// Mouse event with no click - mouse was just released.
			// If there were multiple mouse buttons pressed, we don't know which one
			// was actually released, so we assume they all were released.
			for me := range h.mousePressed {
				delete(h.mousePressed, me)

				me.state = MouseRelease
				h.DoMouseEvent(me, e, takeFocus)
			}
		}
	}
	h.Buf.MergeCursors()

	cursors := h.Buf.GetCursors()
	for _, c := range cursors {
		if c.NewTrailingWsY != c.Y && (!c.HasSelection() ||
			(c.NewTrailingWsY != c.CurSelection[0].Y && c.NewTrailingWsY != c.CurSelection[1].Y)) {
			c.NewTrailingWsY = -1
		}
	}
}

func (h *BufPane) SetBindings(b *KeyTree) {
	h.bindings = b
}

// Bindings returns the current bindings tree for this buffer.
func (h *BufPane) Bindings() *KeyTree {
	if h.bindings != nil {
		return h.bindings
	}
	return BufBindings
}

// DoKeyEvent executes a key event by finding the action it is bound
// to and executing it (possibly multiple times for multiple cursors).
// Returns true if the action was executed OR if there are more keys
// remaining to process before executing an action (if this is a key
// sequence event). Returns false if no action found.
func (h *BufPane) DoKeyEvent(e Event, takeFocus func()) bool {
	binds := h.Bindings()
	action, more := binds.NextEvent(e, nil)
	if action != nil && !more {
		action(h, takeFocus)
		binds.ResetEvents()
		return true
	} else if action == nil && !more {
		binds.ResetEvents()
	}
	return more
}

func (h *BufPane) execAction(action BufAction, name string, te *tcell.EventMouse, takeFocus func()) bool {
	if name != "Autocomplete" && name != "CycleAutocompleteBack" {
		h.Buf.HasSuggestions = false
	}

	var success bool
	switch a := action.(type) {
	case BufKeyAction:
		success = a(h)
	case BufMouseAction:
		success = a(h, te, takeFocus)
	}

	return success
}

func (h *BufPane) HasKeyEvent(e Event) bool {
	// TODO
	return true
	// _, ok := BufKeyBindings[e]
	// return ok
}

// DoMouseEvent executes a mouse event by finding the action it is bound
// to and executing it
func (h *BufPane) DoMouseEvent(e MouseEvent, te *tcell.EventMouse, takeFocus func()) bool {
	binds := h.Bindings()
	action, _ := binds.NextEvent(e, te)
	if action != nil {
		action(h, takeFocus)
		binds.ResetEvents()
		return true
	}
	// TODO
	return false

	// if action, ok := BufMouseBindings[e]; ok {
	// 	if action(h, te) {
	// 		h.Relocate()
	// 	}
	// 	return true
	// } else if h.HasKeyEvent(e) {
	// 	return h.DoKeyEvent(e)
	// }
	// return false
}

// DoRuneInsert inserts a given rune into the current buffer
// (possibly multiple times for multiple cursors)
func (h *BufPane) DoRuneInsert(r rune) {
	cursors := h.Buf.GetCursors()
	for _, c := range cursors {
		// Insert a character
		h.Buf.SetCurCursor(c.Num)
		h.Cursor = c
		if c.HasSelection() {
			c.DeleteSelection()
			c.ResetSelection()
		}

		if h.Buf.OverwriteMode {
			next := c.Loc
			next.X++
			h.Buf.Replace(c.Loc, next, string(r))
		} else {
			h.Buf.Insert(c.Loc, string(r))
		}
		h.Relocate()
	}
}

// BufKeyActions contains the list of all possible key actions the bufhandler could execute
var BufKeyActions = map[string]BufKeyAction{
	"CursorUp":                  (*BufPane).CursorUp,
	"CursorDown":                (*BufPane).CursorDown,
	"CursorPageUp":              (*BufPane).CursorPageUp,
	"CursorPageDown":            (*BufPane).CursorPageDown,
	"CursorLeft":                (*BufPane).CursorLeft,
	"CursorRight":               (*BufPane).CursorRight,
	"CursorStart":               (*BufPane).CursorStart,
	"CursorEnd":                 (*BufPane).CursorEnd,
	"CursorToViewTop":           (*BufPane).CursorToViewTop,
	"CursorToViewCenter":        (*BufPane).CursorToViewCenter,
	"CursorToViewBottom":        (*BufPane).CursorToViewBottom,
	"SelectToStart":             (*BufPane).SelectToStart,
	"SelectToEnd":               (*BufPane).SelectToEnd,
	"SelectUp":                  (*BufPane).SelectUp,
	"SelectDown":                (*BufPane).SelectDown,
	"SelectLeft":                (*BufPane).SelectLeft,
	"SelectRight":               (*BufPane).SelectRight,
	"WordRight":                 (*BufPane).WordRight,
	"WordLeft":                  (*BufPane).WordLeft,
	"SubWordRight":              (*BufPane).SubWordRight,
	"SubWordLeft":               (*BufPane).SubWordLeft,
	"SelectWordRight":           (*BufPane).SelectWordRight,
	"SelectWordLeft":            (*BufPane).SelectWordLeft,
	"SelectSubWordRight":        (*BufPane).SelectSubWordRight,
	"SelectSubWordLeft":         (*BufPane).SelectSubWordLeft,
	"DeleteWordRight":           (*BufPane).DeleteWordRight,
	"DeleteWordLeft":            (*BufPane).DeleteWordLeft,
	"DeleteSubWordRight":        (*BufPane).DeleteSubWordRight,
	"DeleteSubWordLeft":         (*BufPane).DeleteSubWordLeft,
	"SelectLine":                (*BufPane).SelectLine,
	"SelectToStartOfLine":       (*BufPane).SelectToStartOfLine,
	"SelectToStartOfText":       (*BufPane).SelectToStartOfText,
	"SelectToStartOfTextToggle": (*BufPane).SelectToStartOfTextToggle,
	"SelectToEndOfLine":         (*BufPane).SelectToEndOfLine,
	"ParagraphPrevious":         (*BufPane).ParagraphPrevious,
	"ParagraphNext":             (*BufPane).ParagraphNext,
	"SelectToParagraphPrevious": (*BufPane).SelectToParagraphPrevious,
	"SelectToParagraphNext":     (*BufPane).SelectToParagraphNext,
	"InsertNewline":             (*BufPane).InsertNewline,
	"Backspace":                 (*BufPane).Backspace,
	"Delete":                    (*BufPane).Delete,
	"InsertTab":                 (*BufPane).InsertTab,
	"DiffNext":                  (*BufPane).DiffNext,
	"DiffPrevious":              (*BufPane).DiffPrevious,
	"Center":                    (*BufPane).Center,
	"Undo":                      (*BufPane).Undo,
	"Redo":                      (*BufPane).Redo,
	"Copy":                      (*BufPane).Copy,
	"CopyLine":                  (*BufPane).CopyLine,
	"Cut":                       (*BufPane).Cut,
	"CutLine":                   (*BufPane).CutLine,
	"Duplicate":                 (*BufPane).Duplicate,
	"DuplicateLine":             (*BufPane).DuplicateLine,
	"DeleteLine":                (*BufPane).DeleteLine,
	"MoveLinesUp":               (*BufPane).MoveLinesUp,
	"MoveLinesDown":             (*BufPane).MoveLinesDown,
	"IndentSelection":           (*BufPane).IndentSelection,
	"OutdentSelection":          (*BufPane).OutdentSelection,
	// "Autocomplete":          (*BufPane).Autocomplete,
	// "CycleAutocompleteBack": (*BufPane).CycleAutocompleteBack,
	"OutdentLine":       (*BufPane).OutdentLine,
	"IndentLine":        (*BufPane).IndentLine,
	"Paste":             (*BufPane).Paste,
	"PastePrimary":      (*BufPane).PastePrimary,
	"SelectAll":         (*BufPane).SelectAll,
	"Start":             (*BufPane).Start,
	"End":               (*BufPane).End,
	"PageUp":            (*BufPane).PageUp,
	"PageDown":          (*BufPane).PageDown,
	"SelectPageUp":      (*BufPane).SelectPageUp,
	"SelectPageDown":    (*BufPane).SelectPageDown,
	"HalfPageUp":        (*BufPane).HalfPageUp,
	"HalfPageDown":      (*BufPane).HalfPageDown,
	"StartOfText":       (*BufPane).StartOfText,
	"StartOfTextToggle": (*BufPane).StartOfTextToggle,
	"StartOfLine":       (*BufPane).StartOfLine,
	"EndOfLine":         (*BufPane).EndOfLine,
	"ToggleDiffGutter":  (*BufPane).ToggleDiffGutter,
	"ToggleRuler":       (*BufPane).ToggleRuler,
	// "ToggleHighlightSearch":     (*BufPane).ToggleHighlightSearch,
	// "UnhighlightSearch":         (*BufPane).UnhighlightSearch,
	// "ResetSearch":               (*BufPane).ResetSearch,
	// "ShellMode":                 (*BufPane).ShellMode,
	// "CommandMode":               (*BufPane).CommandMode,
	"ToggleOverwriteMode": (*BufPane).ToggleOverwriteMode,
	"Escape":              (*BufPane).Escape,
	// "ToggleMacro": (*BufPane).ToggleMacro,
	// "PlayMacro":   (*BufPane).PlayMacro,
	"ScrollUp":               (*BufPane).ScrollUpAction,
	"ScrollDown":             (*BufPane).ScrollDownAction,
	"SpawnMultiCursor":       (*BufPane).SpawnMultiCursor,
	"SpawnMultiCursorUp":     (*BufPane).SpawnMultiCursorUp,
	"SpawnMultiCursorDown":   (*BufPane).SpawnMultiCursorDown,
	"SpawnMultiCursorSelect": (*BufPane).SpawnMultiCursorSelect,
	"RemoveMultiCursor":      (*BufPane).RemoveMultiCursor,
	"RemoveAllMultiCursors":  (*BufPane).RemoveAllMultiCursors,
	"SkipMultiCursor":        (*BufPane).SkipMultiCursor,
	"SkipMultiCursorBack":    (*BufPane).SkipMultiCursorBack,
	"JumpToMatchingBrace":    (*BufPane).JumpToMatchingBrace,
	"Deselect":               (*BufPane).Deselect,
	"None":                   (*BufPane).None,

	// This was changed to InsertNewline but I don't want to break backwards compatibility
	"InsertEnter": (*BufPane).InsertNewline,

	"SetManualSelectionStart": (*BufPane).SetManualSelectionStart,
	"SetManualSelectionEnd":   (*BufPane).SetManualSelectionEnd,
	"ToggleBookmark":          (*BufPane).ToggleBookmark,
}

// BufMouseActions contains the list of all possible mouse actions the bufhandler could execute
var BufMouseActions = map[string]BufMouseAction{
	"MousePress":       (*BufPane).MousePress,
	"MouseDrag":        (*BufPane).MouseDrag,
	"MouseRelease":     (*BufPane).MouseRelease,
	"MouseMultiCursor": (*BufPane).MouseMultiCursor,
}

// MultiActions is a list of actions that should be executed multiple
// times if there are multiple cursors (one per cursor)
// Generally actions that modify global editor state like quitting or
// saving should not be included in this list
var MultiActions = map[string]bool{
	"CursorUp":                  true,
	"CursorDown":                true,
	"CursorPageUp":              true,
	"CursorPageDown":            true,
	"CursorLeft":                true,
	"CursorRight":               true,
	"CursorStart":               true,
	"CursorEnd":                 true,
	"SelectToStart":             true,
	"SelectToEnd":               true,
	"SelectUp":                  true,
	"SelectDown":                true,
	"SelectLeft":                true,
	"SelectRight":               true,
	"WordRight":                 true,
	"WordLeft":                  true,
	"SubWordRight":              true,
	"SubWordLeft":               true,
	"SelectWordRight":           true,
	"SelectWordLeft":            true,
	"SelectSubWordRight":        true,
	"SelectSubWordLeft":         true,
	"DeleteWordRight":           true,
	"DeleteWordLeft":            true,
	"DeleteSubWordRight":        true,
	"DeleteSubWordLeft":         true,
	"SelectLine":                true,
	"SelectToStartOfLine":       true,
	"SelectToStartOfText":       true,
	"SelectToStartOfTextToggle": true,
	"SelectToEndOfLine":         true,
	"ParagraphPrevious":         true,
	"ParagraphNext":             true,
	"InsertNewline":             true,
	"Backspace":                 true,
	"Delete":                    true,
	"InsertTab":                 true,
	"FindNext":                  true,
	"FindPrevious":              true,
	"CopyLine":                  true,
	"Copy":                      true,
	"Cut":                       true,
	"CutLine":                   true,
	"Duplicate":                 true,
	"DuplicateLine":             true,
	"DeleteLine":                true,
	"MoveLinesUp":               true,
	"MoveLinesDown":             true,
	"IndentSelection":           true,
	"OutdentSelection":          true,
	"OutdentLine":               true,
	"IndentLine":                true,
	"Paste":                     true,
	"PastePrimary":              true,
	"SelectPageUp":              true,
	"SelectPageDown":            true,
	"StartOfLine":               true,
	"StartOfText":               true,
	"StartOfTextToggle":         true,
	"EndOfLine":                 true,
	"JumpToMatchingBrace":       true,
}
