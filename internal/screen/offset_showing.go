package screen

func OffsetShowing(offset, line, height int) int {
	return min(max(offset, line-height+1), line)
}
