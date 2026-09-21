package ssh

import (
	"bufio"
	"os"
	"testing"
)

// swapStdinPipe replaces os.Stdin with a pipe holding the given input.
func swapStdinPipe(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		r.Close()
	})
}

// TestDoKeyboardInteractivePipedInput covers scripted stdin: answers are
// read as whole lines even for echo=false questions (a terminal is required
// to hide them), and a missing trailing newline still yields the answer.
func TestDoKeyboardInteractivePipedInput(t *testing.T) {
	// the test process stdin is not a terminal, both questions fall back
	// to plain reads
	swapStdinPipe(t, "answer one\nans wer two")
	answers, err := doKeyboardInteractive(bufio.NewReader(os.Stdin), false)("user", "", []string{"q1: ", "q2: "}, []bool{false, false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answers) != 2 || answers[0] != "answer one" || answers[1] != "ans wer two" {
		t.Fatalf("answers = %v, want [answer one ans wer two]", answers)
	}
}

func TestDoKeyboardInteractiveNoTrailingNewline(t *testing.T) {
	swapStdinPipe(t, "answer one")
	answers, err := doKeyboardInteractive(bufio.NewReader(os.Stdin), false)("user", "", []string{"q1: "}, []bool{false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answers) != 1 || answers[0] != "answer one" {
		t.Fatalf("answers = %v, want [answer one]", answers)
	}
}
