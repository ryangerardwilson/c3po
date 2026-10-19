package tui

import "unicode/utf8"

// input is a one-line field. Printable keys append, backspace deletes
// the last character, and limit caps the length in characters.
type input struct {
	value   string
	limit   int
	focused bool
}

func (in *input) key(k string) {
	switch {
	case k == "backspace":
		if in.value != "" {
			_, size := utf8.DecodeLastRuneInString(in.value)
			in.value = in.value[:len(in.value)-size]
		}
	case k == "ctrl+u":
		in.value = ""
	case printable(k) && utf8.RuneCountInString(in.value) < in.limit:
		in.value += k
	}
}

func (in input) view(color bool) string {
	if !in.focused {
		return in.value
	}
	if color {
		return in.value + "\x1b[7m \x1b[27m"
	}
	return in.value + "_"
}

func printable(k string) bool {
	return len(k) == 1 && k[0] >= 0x20 && k[0] < 0x7f
}
