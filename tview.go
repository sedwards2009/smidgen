package smidgen

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Smidgen struct {
	*tview.Box
}

func NewSmidgen() *Smidgen {
	s := &Smidgen{
		Box: tview.NewBox(),
	}
	return s
}

func (s *Smidgen) Draw(screen tcell.Screen) {
	s.Box.Draw(screen)
	// Custom drawing logic can be added here
}
