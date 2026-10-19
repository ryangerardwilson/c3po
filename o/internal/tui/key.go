package tui

import "unicode/utf8"

// KeyType names a non-printable key. Printable text is KeyRunes.
type KeyType int

const (
	KeyRunes KeyType = iota
	KeyEnter
	KeyEsc
	KeyBackspace
	KeyTab
	KeyUp
	KeyDown
	KeyRight
	KeyLeft
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
	KeyDelete
	KeyCtrl
	KeyUnknown
)

// KeyMsg is one key press, or a run of printable runes read together.
type KeyMsg struct {
	Type  KeyType
	Runes []rune
	// Ctrl is the character held with ctrl for KeyCtrl, such as 'c'.
	Ctrl rune
	Alt  bool
}

var keyNames = map[KeyType]string{
	KeyEnter:     "enter",
	KeyEsc:       "esc",
	KeyBackspace: "backspace",
	KeyTab:       "tab",
	KeyUp:        "up",
	KeyDown:      "down",
	KeyRight:     "right",
	KeyLeft:      "left",
	KeyHome:      "home",
	KeyEnd:       "end",
	KeyPgUp:      "pgup",
	KeyPgDown:    "pgdown",
	KeyDelete:    "delete",
}

// String is the key's binding name: "j", "enter", "ctrl+k", "alt+x".
func (k KeyMsg) String() string {
	name := k.baseName()
	if k.Alt && name != "" {
		return "alt+" + name
	}
	return name
}

func (k KeyMsg) baseName() string {
	switch k.Type {
	case KeyRunes:
		return string(k.Runes)
	case KeyCtrl:
		return "ctrl+" + string(k.Ctrl)
	}
	return keyNames[k.Type]
}

var controlTypes = map[byte]KeyType{
	0x0d: KeyEnter,
	0x09: KeyTab,
	0x7f: KeyBackspace,
	0x1b: KeyEsc,
}

var sequences = map[string]KeyType{
	"[A": KeyUp, "[B": KeyDown, "[C": KeyRight, "[D": KeyLeft,
	"OA": KeyUp, "OB": KeyDown, "OC": KeyRight, "OD": KeyLeft,
	"[H": KeyHome, "[F": KeyEnd, "OH": KeyHome, "OF": KeyEnd,
	"[1~": KeyHome, "[4~": KeyEnd, "[7~": KeyHome, "[8~": KeyEnd,
	"[5~": KeyPgUp, "[6~": KeyPgDown, "[3~": KeyDelete,
}

// ParseKeys splits bytes read from a raw terminal into key messages.
// Consecutive printable runes become one KeyRunes message.
func ParseKeys(b []byte) []KeyMsg {
	var out []KeyMsg
	for len(b) > 0 {
		k, n := parseOne(b)
		out = append(out, k)
		b = b[n:]
	}
	return out
}

func parseOne(b []byte) (KeyMsg, int) {
	switch {
	case b[0] == 0x1b:
		return parseEscape(b)
	case isControl(b[0]):
		return controlKey(b[0]), 1
	case b[0] == ' ':
		return KeyMsg{Type: KeyRunes, Runes: []rune{' '}}, 1
	}
	return parseRunes(b)
}

func isControl(c byte) bool {
	return c < 0x20 || c == 0x7f
}

func controlKey(c byte) KeyMsg {
	if t, ok := controlTypes[c]; ok {
		return KeyMsg{Type: t}
	}
	if c == 0 {
		return KeyMsg{Type: KeyCtrl, Ctrl: '@'}
	}
	if c <= 0x1a {
		return KeyMsg{Type: KeyCtrl, Ctrl: rune('a' + c - 1)}
	}
	return KeyMsg{Type: KeyCtrl, Ctrl: rune(c + 0x40)}
}

func parseRunes(b []byte) (KeyMsg, int) {
	var runes []rune
	n := 0
	for n < len(b) && b[n] > ' ' && b[n] != 0x7f {
		r, size := utf8.DecodeRune(b[n:])
		runes = append(runes, r)
		n += size
	}
	return KeyMsg{Type: KeyRunes, Runes: runes}, n
}

func parseEscape(b []byte) (KeyMsg, int) {
	if len(b) == 1 || b[1] == 0x1b {
		return KeyMsg{Type: KeyEsc}, 1
	}
	if b[1] == '[' || b[1] == 'O' {
		return parseSequence(b)
	}
	if isControl(b[1]) {
		k := controlKey(b[1])
		k.Alt = true
		return k, 2
	}
	r, size := utf8.DecodeRune(b[1:])
	return KeyMsg{Type: KeyRunes, Runes: []rune{r}, Alt: true}, 1 + size
}

// parseSequence reads ESC [ ... final or ESC O final.
func parseSequence(b []byte) (KeyMsg, int) {
	end := sequenceEnd(b)
	if end < 0 {
		return KeyMsg{Type: KeyUnknown}, len(b)
	}
	if t, ok := sequences[string(b[1:end+1])]; ok {
		return KeyMsg{Type: t}, end + 1
	}
	return KeyMsg{Type: KeyUnknown}, end + 1
}

func sequenceEnd(b []byte) int {
	if b[1] == 'O' {
		if len(b) > 2 {
			return 2
		}
		return -1
	}
	for i := 2; i < len(b); i++ {
		if b[i] >= 0x40 && b[i] <= 0x7e {
			return i
		}
	}
	return -1
}
