package buffer

// SubstrLines returns an array of bytes between between two locations.
func (la *LineArray) SubstrLines(start, end Loc) [][]byte {
	startX := runeToByteIndex(start.X, la.lines[start.Y].data)
	endX := runeToByteIndex(end.X, la.lines[end.Y].data)
	result := make([][]byte, 0)
	if start.Y == end.Y {
		src := la.lines[start.Y].data[startX:endX]
		dest := make([]byte, len(src))
		copy(dest, src)
		result = append(result, dest)
		return result
	}

	str := make([]byte, 0, len(la.lines[start.Y+1].data)*(end.Y-start.Y))
	str = append(str, la.lines[start.Y].data[startX:]...)
	result = append(result, str)

	for i := start.Y + 1; i <= end.Y-1; i++ {
		str = make([]byte, 0, len(la.lines[i].data))
		str = append(str, la.lines[i].data...)
		result = append(result, str)

	}

	str = make([]byte, 0, endX)
	str = append(str, la.lines[end.Y].data[:endX]...)
	result = append(result, str)
	return result
}
