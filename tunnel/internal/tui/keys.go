package tui

// decode names one key from its first byte and, for an escape, the
// bytes that followed it. Unknown keys decode to "".
func decode(first byte, rest string) string {
	if first == 0x1b {
		return escape(rest)
	}
	switch first {
	case 3:
		return "ctrl+c"
	case 21:
		return "ctrl+u"
	case '\r', '\n':
		return "enter"
	case '\t':
		return "tab"
	case 0x7f, 0x08:
		return "backspace"
	}
	if first >= 0x20 && first < 0x7f {
		return string(rune(first))
	}
	return ""
}

func escape(rest string) string {
	switch rest {
	case "":
		return "esc"
	case "[A", "OA":
		return "up"
	case "[B", "OB":
		return "down"
	case "[Z":
		return "shift+tab"
	default:
		return ""
	}
}
