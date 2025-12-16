package buffer

// GetSelection returns the cursor's selection
func (c *Cursor) GetSelectionLines() [][]byte {
	if InBounds(c.CurSelection[0], c.buf) && InBounds(c.CurSelection[1], c.buf) {
		if c.CurSelection[0].GreaterThan(c.CurSelection[1]) {
			return c.buf.SubstrLines(c.CurSelection[1], c.CurSelection[0])
		}
		return c.buf.SubstrLines(c.CurSelection[0], c.CurSelection[1])
	}
	return [][]byte{}
}
