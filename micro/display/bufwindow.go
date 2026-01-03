package display

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/sedwards2009/smidgen/micro/buffer"
	"github.com/sedwards2009/smidgen/micro/config"
	"github.com/sedwards2009/smidgen/micro/util"
)

// The BufWindow provides a way of displaying a certain section of a buffer.
type BufWindow struct {
	*View

	// Buffer being shown in this window
	Buf *buffer.Buffer

	bufWidth         int
	bufHeight        int
	gutterOffset     int
	maxLineNumLength int
	Colorscheme      config.Colorscheme
}

// NewBufWindow creates a new window at a location in the screen with a width and height
func NewBufWindow(x, y, width, height int, buf *buffer.Buffer) *BufWindow {
	w := new(BufWindow)
	w.View = new(View)
	w.X, w.Y, w.Width, w.Height = x, y, width, height
	w.SetBuffer(buf)
	return w
}

// SetBuffer sets this window's buffer.
func (w *BufWindow) SetBuffer(b *buffer.Buffer) {
	w.Buf = b
	b.OptionCallback = func(option string, nativeValue any) {
		if option == "softwrap" {
			if nativeValue.(bool) {
				w.StartCol = 0
			} else {
				w.StartLine.Row = 0
			}
		}

		if option == "softwrap" || option == "wordwrap" {
			w.Relocate()
			for _, c := range w.Buf.GetCursors() {
				c.LastWrappedVisualX = c.GetVisualX(true)
			}
		}

		if option == "diffgutter" || option == "ruler" || option == "scrollbar" {
			w.updateDisplayInfo()
			w.Relocate()
		}
	}
	b.GetVisualX = func(loc buffer.Loc) int {
		return w.VLocFromLoc(loc).VisualX
	}
}

// GetView gets the view.
func (w *BufWindow) GetView() *View {
	return w.View
}

// GetView sets the view.
func (w *BufWindow) SetView(view *View) {
	w.View = view
}

// Resize resizes this window.
func (w *BufWindow) Resize(width, height int) {
	w.Width, w.Height = width, height
	w.updateDisplayInfo()

	w.Relocate()
}

// BufView returns the width, height and x,y location of the actual buffer.
// It is not exactly the same as the whole window which also contains gutter,
// ruler, scrollbar.
func (w *BufWindow) BufView() View {
	return View{
		X:         w.X + w.gutterOffset,
		Y:         w.Y,
		Width:     w.bufWidth,
		Height:    w.bufHeight,
		StartLine: w.StartLine,
		StartCol:  w.StartCol,
	}
}

func (w *BufWindow) updateDisplayInfo() {
	b := w.Buf
	w.bufHeight = w.Height

	scrollbarWidth := 0
	if w.Buf.Settings["scrollbar"].(bool) && w.Buf.LinesNum() > w.Height && w.Width > 0 {
		scrollbarWidth = 1
	}

	// We need to know the string length of the largest line number
	// so we can pad appropriately when displaying line numbers
	w.maxLineNumLength = len(strconv.Itoa(b.LinesNum()))

	w.gutterOffset = 0
	if b.Settings["diffgutter"].(bool) {
		w.gutterOffset++
	}
	if b.Settings["ruler"].(bool) {
		w.gutterOffset += w.maxLineNumLength + 1
	}

	if w.gutterOffset > w.Width-scrollbarWidth {
		w.gutterOffset = w.Width - scrollbarWidth
	}

	prevBufWidth := w.bufWidth
	w.bufWidth = w.Width - w.gutterOffset - scrollbarWidth

	if w.bufWidth != prevBufWidth && w.Buf.Settings["softwrap"].(bool) {
		for _, c := range w.Buf.GetCursors() {
			c.LastWrappedVisualX = c.GetVisualX(true)
		}
	}
}

func (w *BufWindow) getStartInfo(screen tcell.Screen, n, lineN int) ([]byte, int, int, *tcell.Style) {
	tabsize := util.IntOpt(w.Buf.Settings["tabsize"])
	width := 0
	bloc := buffer.Loc{0, lineN}
	b := w.Buf.LineBytes(lineN)
	curStyle := w.Colorscheme.GetDefault()
	var s *tcell.Style
	for len(b) > 0 {
		r, _, size := util.DecodeCharacter(b)

		curStyle, found := w.getStyle(curStyle, bloc)
		if found {
			s = &curStyle
		}

		w := 0
		switch r {
		case '\t':
			ts := tabsize - (width % tabsize)
			w = ts
		default:
			w = runewidth.RuneWidth(r)
		}
		if width+w > n {
			return b, n - width, bloc.X, s
		}
		width += w
		b = b[size:]
		bloc.X++
	}
	return b, n - width, bloc.X, s
}

// Relocate moves the view window so that the cursor is in view
// This is useful if the user has scrolled far away, and then starts typing
// Returns true if the window location is moved
func (w *BufWindow) Relocate() bool {
	b := w.Buf
	height := w.bufHeight
	ret := false
	activeC := w.Buf.GetActiveCursor()
	scrollmargin := int(b.Settings["scrollmargin"].(float64))

	c := w.SLocFromLoc(activeC.Loc)
	bStart := SLoc{0, 0}
	bEnd := w.SLocFromLoc(b.End())

	if c.LessThan(w.Scroll(w.StartLine, scrollmargin)) && c.GreaterThan(w.Scroll(bStart, scrollmargin-1)) {
		w.StartLine = w.Scroll(c, -scrollmargin)
		ret = true
	} else if c.LessThan(w.StartLine) {
		w.StartLine = c
		ret = true
	}
	if c.GreaterThan(w.Scroll(w.StartLine, height-1-scrollmargin)) && c.LessEqual(w.Scroll(bEnd, -scrollmargin)) {
		w.StartLine = w.Scroll(c, -height+1+scrollmargin)
		ret = true
	} else if c.GreaterThan(w.Scroll(bEnd, -scrollmargin)) && c.GreaterThan(w.Scroll(w.StartLine, height-1)) {
		w.StartLine = w.Scroll(bEnd, -height+1)
		ret = true
	}

	// horizontal relocation (scrolling)
	if !b.Settings["softwrap"].(bool) {
		cx := activeC.GetVisualX(false)
		rw := runewidth.RuneWidth(activeC.RuneUnder(activeC.X))
		if rw == 0 {
			rw = 1 // tab or newline
		}

		if cx < w.StartCol {
			w.StartCol = cx
			ret = true
		}
		if cx+rw > w.StartCol+w.bufWidth {
			w.StartCol = cx - w.bufWidth + rw
			ret = true
		}
	}
	return ret
}

// LocFromVisual takes a visual location (x and y position) and returns the
// position in the buffer corresponding to the visual location
// If the requested position does not correspond to a buffer location it returns
// the nearest position
func (w *BufWindow) LocFromVisual(svloc buffer.Loc) buffer.Loc {
	vx := svloc.X - w.X - w.gutterOffset
	if vx < 0 {
		vx = 0
	}
	vloc := VLoc{
		SLoc:    w.Scroll(w.StartLine, svloc.Y-w.Y),
		VisualX: vx + w.StartCol,
	}
	return w.LocFromVLoc(vloc)
}

func (w *BufWindow) drawGutter(screen tcell.Screen, vloc *buffer.Loc, bloc *buffer.Loc) {
	char := ' '
	s := w.Colorscheme.GetDefault()
	for i := 0; i < 2 && vloc.X < w.gutterOffset; i++ {
		screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, char, nil, s)
		vloc.X++
	}
}

func (w *BufWindow) drawDiffGutter(screen tcell.Screen, backgroundStyle tcell.Style, softwrapped bool, vloc *buffer.Loc, bloc *buffer.Loc) {
	if vloc.X >= w.gutterOffset {
		return
	}

	symbol := ' '
	styleName := ""

	switch w.Buf.DiffStatus(bloc.Y) {
	case buffer.DSAdded:
		symbol = '\u258C' // Left half block
		styleName = "diff-added"
	case buffer.DSModified:
		symbol = '\u258C' // Left half block
		styleName = "diff-modified"
	case buffer.DSDeletedAbove:
		if !softwrapped {
			symbol = '\u2594' // Upper one eighth block
			styleName = "diff-deleted"
		}
	}

	style := backgroundStyle
	if s, ok := w.Colorscheme[styleName]; ok {
		foreground, _, _ := s.Decompose()
		style = style.Foreground(foreground)
	}

	screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, symbol, nil, style)
	vloc.X++
}

func (w *BufWindow) drawLineNum(screen tcell.Screen, lineNumStyle tcell.Style, softwrapped bool, vloc *buffer.Loc, bloc *buffer.Loc) {
	cursorLine := w.Buf.GetActiveCursor().Loc.Y
	var lineInt int
	if w.Buf.Settings["relativeruler"] == false || cursorLine == bloc.Y {
		lineInt = bloc.Y + 1
	} else {
		lineInt = bloc.Y - cursorLine
	}
	lineNum := []rune(strconv.Itoa(util.Abs(lineInt)))

	// Write the spaces before the line number if necessary
	for i := 0; i < w.maxLineNumLength-len(lineNum) && vloc.X < w.gutterOffset; i++ {
		screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, ' ', nil, lineNumStyle)
		vloc.X++
	}
	// Write the actual line number
	for i := 0; i < len(lineNum) && vloc.X < w.gutterOffset; i++ {
		if softwrapped || (w.bufWidth == 0 && w.Buf.Settings["softwrap"] == true) {
			screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, ' ', nil, lineNumStyle)
		} else {
			screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, lineNum[i], nil, lineNumStyle)
		}
		vloc.X++
	}

	// Write the extra space
	if vloc.X < w.gutterOffset {
		screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, ' ', nil, lineNumStyle)
		vloc.X++
	}
}

// getStyle returns the highlight style for the given character position
// If there is no change to the current highlight style it just returns that
func (w *BufWindow) getStyle(style tcell.Style, bloc buffer.Loc) (tcell.Style, bool) {
	if group, ok := w.Buf.Match(bloc.Y)[bloc.X]; ok {
		s := w.Colorscheme.GetColor(group.String())
		return s, true
	}
	return style, false
}

func (w *BufWindow) showCursor(screen tcell.Screen, hasFocus bool, x, y int, main bool) {
	if hasFocus {
		if main {
			screen.ShowCursor(x, y)
		} else {
			w.showFakeCursorMulti(screen, x, y)
		}
	}
}

// ShowFakeCursorMulti is the same as ShowFakeCursor except it does not
// reset previous locations of the cursor
// Fake cursors are also necessary to display multiple cursors
func (w *BufWindow) showFakeCursorMulti(screen tcell.Screen, x int, y int) {
	r, _, _, _ := screen.GetContent(x, y)
	screen.SetContent(x, y, r, nil, w.Colorscheme.GetDefault().Reverse(true))
}

// displayBuffer draws the buffer being shown in this window on the screen.Screen
func (w *BufWindow) displayBuffer(screen tcell.Screen, hasFocus bool) {
	if hasFocus {
		// We are actice, so we control the cursor
		screen.HideCursor()
	}

	b := w.Buf

	if w.Height <= 0 || w.Width <= 0 {
		return
	}

	maxWidth := w.gutterOffset + w.bufWidth

	if b.ModifiedThisFrame {
		if b.Settings["diffgutter"].(bool) {
			b.UpdateDiff()
		}
		b.ModifiedThisFrame = false
	}

	var matchingBraces []buffer.Loc
	// bracePairs is defined in buffer.go
	if b.Settings["matchbrace"].(bool) {
		for _, c := range b.GetCursors() {
			if c.HasSelection() {
				continue
			}

			mb, left, found := b.FindMatchingBrace(c.Loc)
			if found {
				matchingBraces = append(matchingBraces, mb)
				if !left {
					if b.Settings["matchbracestyle"].(string) != "highlight" {
						matchingBraces = append(matchingBraces, c.Loc)
					}
				} else {
					matchingBraces = append(matchingBraces, c.Loc.Move(-1, b))
				}
			}
		}
	}

	lineNumStyle := w.Colorscheme.GetDefault()
	if style, ok := w.Colorscheme["line-number"]; ok {
		lineNumStyle = style
	}
	curNumStyle := w.Colorscheme.GetDefault()
	if style, ok := w.Colorscheme["current-line-number"]; ok {
		if !b.Settings["cursorline"].(bool) {
			curNumStyle = lineNumStyle
		} else {
			curNumStyle = style
		}
	}

	bookmarkStyle := w.Colorscheme.GetDefault()
	bookmarkStyle = bookmarkStyle.Reverse(true)
	if style, ok := w.Colorscheme["bookmark"]; ok {
		bookmarkStyle = style
	}

	softwrap := b.Settings["softwrap"].(bool)
	wordwrap := softwrap && b.Settings["wordwrap"].(bool)

	tabsize := util.IntOpt(b.Settings["tabsize"])
	colorcolumn := util.IntOpt(b.Settings["colorcolumn"])

	// this represents the current draw position
	// within the current window
	vloc := buffer.Loc{X: 0, Y: 0}
	if softwrap {
		// the start line may be partially out of the current window
		vloc.Y = -w.StartLine.Row
	}

	// this represents the current draw position in the buffer (char positions)
	bloc := buffer.Loc{X: -1, Y: w.StartLine.Line}

	cursors := b.GetCursors()

	curStyle := w.Colorscheme.GetDefault()

	// Parse showchars which is in the format of key1=val1,key2=val2,...
	spacechars := " "
	tabchars := b.Settings["indentchar"].(string)
	var indentspacechars string
	var indenttabchars string
	for _, entry := range strings.Split(b.Settings["showchars"].(string), ",") {
		split := strings.SplitN(entry, "=", 2)
		if len(split) < 2 {
			continue
		}
		key, val := split[0], split[1]
		switch key {
		case "space":
			spacechars = val
		case "tab":
			tabchars = val
		case "ispace":
			indentspacechars = val
		case "itab":
			indenttabchars = val
		}
	}

	bookmarkLines := b.BookmarkLinesSet()

	for ; vloc.Y < w.bufHeight; vloc.Y++ {
		vloc.X = 0

		currentLine := false
		for _, c := range cursors {
			if !c.HasSelection() && bloc.Y == c.Y && hasFocus {
				currentLine = true
				break
			}
		}

		s := lineNumStyle
		if currentLine {
			s = curNumStyle
		}

		if vloc.Y >= 0 {
			if b.Settings["diffgutter"].(bool) {
				w.drawDiffGutter(screen, s, false, &vloc, &bloc)
			}

			if b.Settings["ruler"].(bool) {
				if _, ok := bookmarkLines[bloc.Y]; ok {
					w.drawLineNum(screen, bookmarkStyle, false, &vloc, &bloc)
				} else {
					w.drawLineNum(screen, s, false, &vloc, &bloc)
				}
			}
		} else {
			vloc.X = w.gutterOffset
		}

		bline := b.LineBytes(bloc.Y)
		blineLen := util.CharacterCount(bline)

		leadingwsEnd := len(util.GetLeadingWhitespace(bline))
		trailingwsStart := blineLen - util.CharacterCount(util.GetTrailingWhitespace(bline))

		line, nColsBeforeStart, bslice, startStyle := w.getStartInfo(screen, w.StartCol, bloc.Y)
		if startStyle != nil {
			curStyle = *startStyle
		}
		bloc.X = bslice

		// returns the rune to be drawn, style of it and if the bg should be preserved
		getRuneStyle := func(r rune, style tcell.Style, showoffset int, linex int, isplaceholder bool) (rune, tcell.Style, bool) {
			if nColsBeforeStart > 0 || vloc.Y < 0 || isplaceholder {
				return r, style, false
			}

			for _, mb := range matchingBraces {
				if mb.X == bloc.X && mb.Y == bloc.Y {
					if b.Settings["matchbracestyle"].(string) == "highlight" {
						if s, ok := w.Colorscheme["match-brace"]; ok {
							return r, s, false
						} else {
							return r, style.Reverse(true), false
						}
					} else {
						return r, style.Underline(true), false
					}
				}
			}

			if r != '\t' && r != ' ' {
				return r, style, false
			}

			var indentrunes []rune
			switch r {
			case '\t':
				if bloc.X < leadingwsEnd && indenttabchars != "" {
					indentrunes = []rune(indenttabchars)
				} else {
					indentrunes = []rune(tabchars)
				}
			case ' ':
				if linex%tabsize == 0 && bloc.X < leadingwsEnd && indentspacechars != "" {
					indentrunes = []rune(indentspacechars)
				} else {
					indentrunes = []rune(spacechars)
				}
			}

			var drawrune rune
			if showoffset < len(indentrunes) {
				drawrune = indentrunes[showoffset]
			} else {
				// use space if no showchars or after we showed showchars
				drawrune = ' '
			}

			if s, ok := w.Colorscheme["indent-char"]; ok {
				fg, _, _ := s.Decompose()
				style = style.Foreground(fg)
			}

			preservebg := false
			if b.Settings["hltaberrors"].(bool) && bloc.X < leadingwsEnd {
				if s, ok := w.Colorscheme["tab-error"]; ok {
					if b.Settings["tabstospaces"].(bool) && r == '\t' {
						fg, _, _ := s.Decompose()
						style = style.Background(fg)
						preservebg = true
					} else if !b.Settings["tabstospaces"].(bool) && r == ' ' {
						fg, _, _ := s.Decompose()
						style = style.Background(fg)
						preservebg = true
					}
				}
			}

			if b.Settings["hltrailingws"].(bool) {
				if s, ok := w.Colorscheme["trailingws"]; ok {
					if bloc.X >= trailingwsStart && bloc.X < blineLen {
						hl := true
						for _, c := range cursors {
							if c.NewTrailingWsY == bloc.Y {
								hl = false
								break
							}
						}
						if hl {
							fg, _, _ := s.Decompose()
							style = style.Background(fg)
							preservebg = true
						}
					}
				}
			}

			return drawrune, style, preservebg
		}

		draw := func(r rune, combc []rune, style tcell.Style, highlight bool, showcursor bool, preservebg bool) {
			defer func() {
				if nColsBeforeStart <= 0 {
					vloc.X++
				}
				nColsBeforeStart--
			}()

			if nColsBeforeStart > 0 || vloc.Y < 0 {
				return
			}

			if highlight {
				if w.Buf.HighlightSearch && w.Buf.SearchMatch(bloc) {
					style = w.Colorscheme.GetDefault().Reverse(true)
					if s, ok := w.Colorscheme["hlsearch"]; ok {
						style = s
					}
				}

				_, origBg, _ := style.Decompose()
				_, defBg, _ := w.Colorscheme.GetDefault().Decompose()

				// syntax or hlsearch highlighting with non-default background takes precedence
				// over cursor-line and color-column
				if !preservebg && origBg != defBg {
					preservebg = true
				}

				for _, c := range cursors {
					if c.HasSelection() &&
						(bloc.GreaterEqual(c.CurSelection[0]) && bloc.LessThan(c.CurSelection[1]) ||
							bloc.LessThan(c.CurSelection[0]) && bloc.GreaterEqual(c.CurSelection[1])) {
						// The current character is selected
						style = w.Colorscheme.GetDefault().Reverse(true)

						if s, ok := w.Colorscheme["selection"]; ok {
							style = s
						}
					}

					if b.Settings["cursorline"].(bool) && hasFocus && !preservebg &&
						!c.HasSelection() && c.Y == bloc.Y {
						if s, ok := w.Colorscheme["cursor-line"]; ok {
							fg, _, _ := s.Decompose()
							style = style.Background(fg)
						}
					}
				}

				if s, ok := w.Colorscheme["color-column"]; ok {
					if colorcolumn != 0 && vloc.X-w.gutterOffset+w.StartCol == colorcolumn && !preservebg {
						fg, _, _ := s.Decompose()
						style = style.Background(fg)
					}
				}
			}

			screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, r, combc, style)

			if showcursor {
				for _, c := range cursors {
					if c.X == bloc.X && c.Y == bloc.Y && !c.HasSelection() {
						w.showCursor(screen, hasFocus, w.X+vloc.X, w.Y+vloc.Y, c.Num == 0)
					}
				}
			}
		}

		wrap := func() {
			vloc.X = 0

			if vloc.Y >= 0 {
				if b.Settings["diffgutter"].(bool) {
					w.drawDiffGutter(screen, lineNumStyle, true, &vloc, &bloc)
				}

				// This will draw an empty line number because the current line is wrapped
				if b.Settings["ruler"].(bool) {
					if _, ok := bookmarkLines[bloc.Y]; ok {
						w.drawLineNum(screen, bookmarkStyle, true, &vloc, &bloc)
					} else {
						w.drawLineNum(screen, lineNumStyle, true, &vloc, &bloc)
					}
				}
			} else {
				vloc.X = w.gutterOffset
			}
		}

		type glyph struct {
			r     rune
			combc []rune
			style tcell.Style
			width int
		}

		var word []glyph
		if wordwrap {
			word = make([]glyph, 0, w.bufWidth)
		} else {
			word = make([]glyph, 0, 1)
		}
		wordwidth := 0

		totalwidth := w.StartCol - nColsBeforeStart
		for len(line) > 0 && vloc.X < maxWidth {
			r, combc, size := util.DecodeCharacter(line)
			line = line[size:]

			loc := buffer.Loc{X: bloc.X + len(word), Y: bloc.Y}
			curStyle, _ = w.getStyle(curStyle, loc)

			width := 0

			linex := totalwidth
			switch r {
			case '\t':
				ts := tabsize - (totalwidth % tabsize)
				width = util.Min(ts, maxWidth-vloc.X)
				totalwidth += ts
			default:
				width = runewidth.RuneWidth(r)
				totalwidth += width
			}

			word = append(word, glyph{r, combc, curStyle, width})
			wordwidth += width

			// Collect a complete word to know its width.
			// If wordwrap is off, every single character is a complete "word".
			if wordwrap {
				if !util.IsWhitespace(r) && len(line) > 0 && wordwidth < w.bufWidth {
					continue
				}
			}

			// If a word (or just a wide rune) does not fit in the window
			if vloc.X+wordwidth > maxWidth && vloc.X > w.gutterOffset {
				for vloc.X < maxWidth {
					draw(' ', nil, w.Colorscheme.GetDefault(), false, false, true)
				}

				// We either stop or we wrap to draw the word in the next line
				if !softwrap {
					break
				} else {
					vloc.Y++
					if vloc.Y >= w.bufHeight {
						break
					}
					wrap()
				}
			}

			for _, r := range word {
				drawrune, drawstyle, preservebg := getRuneStyle(r.r, r.style, 0, linex, false)
				draw(drawrune, r.combc, drawstyle, true, true, preservebg)

				// Draw extra characters for tabs or wide runes
				for i := 1; i < r.width; i++ {
					if r.r == '\t' {
						drawrune, drawstyle, preservebg = getRuneStyle('\t', r.style, i, linex+i, false)
					} else {
						drawrune, drawstyle, preservebg = getRuneStyle(' ', r.style, i, linex+i, true)
					}
					draw(drawrune, nil, drawstyle, true, false, preservebg)
				}
				bloc.X++
			}

			word = word[:0]
			wordwidth = 0

			// If we reach the end of the window then we either stop or we wrap for softwrap
			if vloc.X >= maxWidth {
				if !softwrap {
					break
				} else {
					vloc.Y++
					if vloc.Y >= w.bufHeight {
						break
					}
					wrap()
				}
			}
		}

		style := w.Colorscheme.GetDefault()
		for _, c := range cursors {
			if b.Settings["cursorline"].(bool) && hasFocus &&
				!c.HasSelection() && c.Y == bloc.Y {
				if s, ok := w.Colorscheme["cursor-line"]; ok {
					fg, _, _ := s.Decompose()
					style = style.Background(fg)
				}
			}
		}
		for i := vloc.X; i < maxWidth; i++ {
			curStyle := style
			if s, ok := w.Colorscheme["color-column"]; ok {
				if colorcolumn != 0 && i-w.gutterOffset+w.StartCol == colorcolumn {
					fg, _, _ := s.Decompose()
					curStyle = style.Background(fg)
				}
			}
			screen.SetContent(i+w.X, vloc.Y+w.Y, ' ', nil, curStyle)
		}

		if vloc.X != maxWidth {
			// Display newline within a selection
			drawrune, drawstyle, preservebg := getRuneStyle(' ', w.Colorscheme.GetDefault(), 0, totalwidth, true)
			draw(drawrune, nil, drawstyle, true, true, preservebg)
		}

		bloc.X = w.StartCol
		bloc.Y++
		if bloc.Y >= b.LinesNum() {
			break
		}
	}

	// Fill in any blank lines at the bottom of the editor
	style := w.Colorscheme.GetDefault()
	for y := vloc.Y + 1; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			screen.SetContent(w.X+x, y+w.Y, ' ', nil, style)
		}
	}
}

func (w *BufWindow) displayScrollBar(screen tcell.Screen) {
	if w.Buf.Settings["scrollbar"].(bool) && w.Buf.LinesNum() > w.Height {
		scrollX := w.X + w.Width - 1
		barsize := int(float64(w.Height) / float64(w.Buf.LinesNum()) * float64(w.Height))
		if barsize < 1 {
			barsize = 1
		}
		barstart := w.Y + int(float64(w.StartLine.Line)/float64(w.Buf.LinesNum())*float64(w.Height))

		scrollBarStyle := w.Colorscheme.GetDefault().Reverse(true)
		if style, ok := w.Colorscheme["scrollbar"]; ok {
			scrollBarStyle = style
		}

		scrollBarRune := []rune("|")

		for y := barstart; y < util.Min(barstart+barsize, w.Y+w.bufHeight); y++ {
			screen.SetContent(scrollX, y, scrollBarRune[0], nil, scrollBarStyle)
		}
	}
}

// Display displays the buffer
func (w *BufWindow) Display(screen tcell.Screen, hasFocus bool) {
	w.updateDisplayInfo()
	w.displayScrollBar(screen)
	w.displayBuffer(screen, hasFocus)
}
