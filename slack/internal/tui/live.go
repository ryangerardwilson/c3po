package tui

import "github.com/ryangerardwilson/c3po/internal/tuisession"

// Run opens the screen. A non-terminal returns before a session is written.
func Run(id string) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	sess, err := tuisession.Open("slack", id)
	if err != nil {
		return err
	}
	defer sess.Close()
	board, err := fetchBoard()
	if err != nil {
		board = Board{Error: err.Error()}
	}
	return newModel(board, sess).loop(term)
}
