package screen

func Arrow(canOpen, isOpen bool) string {
	switch {
	case !canOpen:
		return "  "
	case isOpen:
		return "▾ "
	}
	return "▸ "
}

func Branch(last bool) string {
	if last {
		return "└─ "
	}
	return "├─ "
}

func ChildIndent(lastParent bool) string {
	if lastParent {
		return "   "
	}
	return "│  "
}
