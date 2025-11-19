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

type View struct {
	*tview.Box
	buffer    *buffer.Buffer
	bufWindow *display.BufWindow
	bufPane   *action.BufPane
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
	action.InitBindings()
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

func NewBufferFromString(content string, path string) *Buffer {
	return &Buffer{
		Buffer: buffer.NewBufferFromString(content, path),
	}
}

type Colorscheme config.Colorscheme

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
