package app

import "unicode"

// setInput stores the command text and clamps the cursor onto it.
func (m model) setInput(rs []rune, at int) model {
	if at < 0 {
		at = 0
	}
	if at > len(rs) {
		at = len(rs)
	}
	m.input = string(rs)
	m.inputAt = at
	return m
}

func (m model) insertInput(text string) model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	add := []rune(text)
	next := make([]rune, 0, len(rs)+len(add))
	next = append(next, rs[:at]...)
	next = append(next, add...)
	next = append(next, rs[at:]...)
	return m.setInput(next, at+len(add))
}

func (m model) moveInput(delta int) model {
	return m.setInput([]rune(m.input), m.inputAt+delta)
}

func (m model) moveWord(forward bool) model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	if forward {
		at = skipWordForward(rs, at)
	} else {
		at = skipWordBackward(rs, at)
	}
	return m.setInput(rs, at)
}

func (m model) deleteBack() model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	if at == 0 {
		return m.setInput(rs, 0)
	}
	next := make([]rune, 0, len(rs)-1)
	next = append(next, rs[:at-1]...)
	next = append(next, rs[at:]...)
	return m.setInput(next, at-1)
}

func (m model) deleteForward() model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	if at >= len(rs) {
		return m.setInput(rs, at)
	}
	next := make([]rune, 0, len(rs)-1)
	next = append(next, rs[:at]...)
	next = append(next, rs[at+1:]...)
	return m.setInput(next, at)
}

// killWordBack removes the spaces just before the cursor, then the word
// before those spaces. A word is a run of non-space characters.
func (m model) killWordBack() model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	to := skipWordBackward(rs, at)
	next := make([]rune, 0, len(rs)-(at-to))
	next = append(next, rs[:to]...)
	next = append(next, rs[at:]...)
	return m.setInput(next, to)
}

func (m model) killWordForward() model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	to := skipWordForward(rs, at)
	next := make([]rune, 0, len(rs)-(to-at))
	next = append(next, rs[:at]...)
	next = append(next, rs[to:]...)
	return m.setInput(next, at)
}

func (m model) killToStart() model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	return m.setInput(append([]rune(nil), rs[at:]...), 0)
}

func (m model) killToEnd() model {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	return m.setInput(append([]rune(nil), rs[:at]...), at)
}

func clampAt(rs []rune, at int) int {
	if at < 0 {
		return 0
	}
	if at > len(rs) {
		return len(rs)
	}
	return at
}

func skipWordForward(rs []rune, at int) int {
	for at < len(rs) && unicode.IsSpace(rs[at]) {
		at++
	}
	for at < len(rs) && !unicode.IsSpace(rs[at]) {
		at++
	}
	return at
}

func skipWordBackward(rs []rune, at int) int {
	for at > 0 && unicode.IsSpace(rs[at-1]) {
		at--
	}
	for at > 0 && !unicode.IsSpace(rs[at-1]) {
		at--
	}
	return at
}
