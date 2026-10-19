package tui

import (
	"os"
	"sync"
	"time"
)

// Program runs one Model against the terminal.
type Program struct {
	model    Model
	term     *terminal
	msgs     chan Msg
	done     chan struct{}
	inputMu  sync.Mutex
	renderer renderer
}

// NewProgram prepares m to run.
func NewProgram(m Model) *Program {
	return &Program{model: m, msgs: make(chan Msg, 64), done: make(chan struct{})}
}

// Run takes the terminal, runs the event loop until the model quits, and
// returns the final model.
func (p *Program) Run() (Model, error) {
	t, err := openTerminal()
	if err != nil {
		return p.model, err
	}
	p.term = t
	defer t.close()
	if err := t.raw(); err != nil {
		return p.model, err
	}
	defer t.restore()
	defer close(p.done)
	p.renderer = renderer{out: t.out}
	_, _ = t.out.WriteString("\x1b[?25l")
	defer t.out.WriteString("\x1b[?25h")
	go p.readInput()
	stopSignals := p.watchSignals()
	defer stopSignals()
	return p.loop()
}

func (p *Program) loop() (Model, error) {
	p.dispatch(p.model.Init())
	p.handle(p.term.size())
	for {
		p.renderer.paint(p.model.View())
		msg := <-p.msgs
		switch msg.(type) {
		case QuitMsg:
			p.renderer.paint(p.model.View())
			p.renderer.stop()
			return p.model, nil
		case interruptMsg:
			p.renderer.stop()
			return p.model, ErrInterrupted
		}
		p.handle(msg)
	}
}

func (p *Program) handle(msg Msg) {
	switch msg := msg.(type) {
	case nil:
		return
	case BatchMsg:
		for _, cmd := range msg {
			p.dispatch(cmd)
		}
		return
	case execMsg:
		p.exec(msg)
		return
	case WindowSizeMsg:
		p.renderer.resize(msg.Width, msg.Height)
	}
	next, cmd := p.model.Update(msg)
	p.model = next
	p.dispatch(cmd)
}

func (p *Program) dispatch(cmd Cmd) {
	if cmd == nil {
		return
	}
	go func() { p.send(cmd()) }()
}

func (p *Program) send(msg Msg) {
	if msg == nil {
		return
	}
	select {
	case p.msgs <- msg:
	case <-p.done:
	}
}

const inputPoll = 50 * time.Millisecond

func (p *Program) readInput() {
	buf := make([]byte, 1024)
	for !p.stopped() {
		p.inputMu.Lock()
		keys, err := p.term.readKeys(buf, inputPoll)
		p.inputMu.Unlock()
		if err != nil {
			p.send(QuitMsg{})
			return
		}
		for _, k := range keys {
			p.send(k)
		}
	}
}

func (p *Program) stopped() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// exec releases the terminal to an external command and takes it back.
func (p *Program) exec(msg execMsg) {
	p.inputMu.Lock()
	p.renderer.stop()
	_, _ = p.term.out.WriteString("\x1b[?25h")
	p.term.restore()
	attach(msg, p.term)
	err := msg.cmd.Run()
	_ = p.term.raw()
	_, _ = p.term.out.WriteString("\x1b[?25l")
	p.renderer.repaint()
	p.inputMu.Unlock()
	if msg.fn != nil {
		p.handle(msg.fn(err))
	}
}

func attach(msg execMsg, t *terminal) {
	if msg.cmd.Stdin == nil {
		msg.cmd.Stdin = t.in
	}
	if msg.cmd.Stdout == nil {
		msg.cmd.Stdout = t.out
	}
	if msg.cmd.Stderr == nil {
		msg.cmd.Stderr = os.Stderr
	}
}
