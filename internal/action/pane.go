package action

import (
	"github.com/sedwards2009/smidgen/internal/display"
)

// A Pane is a general interface for a window in the editor.
type Pane interface {
	Handler
	display.Window
	Name() string
}
