package action

// Extra actions which are not from micro

func (h *BufPane) TransformSelection(transformFunc func([]string) []string) {
	cursors := h.Buf.GetCursors()
	for _, c := range cursors {
		lines := c.GetSelectionLines()

		strLines := make([]string, len(lines))
		for i, line := range lines {
			strLines[i] = string(line)
		}

		transformedLines := transformFunc(strLines)

		startLoc := c.CurSelection[0]
		endLoc := c.CurSelection[1]
		if startLoc.GreaterThan(endLoc) {
			startLoc = endLoc
		}

		c.DeleteSelection()

		insertLoc := startLoc
		endLoc = startLoc
		for i, tline := range transformedLines {
			if i != 0 {
				h.Buf.Insert(insertLoc, "\n")
				insertLoc.Y++
				insertLoc.X = 0
			}
			h.Buf.Insert(insertLoc, tline)
			insertLoc.X += len(tline)
		}
		c.SetSelectionStart(startLoc)
		c.SetSelectionEnd(insertLoc)
	}
}

func (h *BufPane) SetManualSelectionStart() bool {
	h.Buf.ManualSelection.Loc = h.Cursor.Loc
	h.Buf.ManualSelection.CurSelection[0] = h.Cursor.Loc
	h.SelectManualSelection()
	return true
}

func (h *BufPane) SetManualSelectionEnd() bool {
	h.Buf.ManualSelection.CurSelection[1] = h.Cursor.Loc
	h.SelectManualSelection()
	return true
}

func (h *BufPane) SelectManualSelection() bool {
	if h.Buf.ManualSelection.CurSelection[0] == h.Buf.ManualSelection.CurSelection[1] {
		return false
	}
	h.Cursor.SetSelectionStart(h.Buf.ManualSelection.CurSelection[0])
	h.Cursor.SetSelectionEnd(h.Buf.ManualSelection.CurSelection[1])
	h.Relocate()
	return true
}
