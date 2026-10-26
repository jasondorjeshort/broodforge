// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// prompter asks the owner (decisions/0027-setup-commands.md, `broodforge setup`): every
// question has a flag; --yes takes the shown default and refuses needs-input-<step> naming the
// flag where there is none; off a terminal a question is refused the same way, so a script
// never blocks on one. Questions go to stderr, so --json's stdout stays one object per line.
// Reading stdin off a terminal is for the key's pause (a line, or its end) and
// `setup git-credential`'s value only.
type prompter struct {
	in      *lineReader
	out     io.Writer
	tty     bool
	yes     bool
	echoOff func() (restore func())
}

// Prompt asks text and returns the answer, def on an empty one; flag and step name the refusal.
func (p *prompter) Prompt(step, flag, text, def string) (string, error) {
	if p.yes {
		if def != "" {
			return def, nil
		}
		return "", needsInput(step, flag)
	}
	if !p.tty {
		return "", needsInput(step, flag)
	}
	shown := text
	if def != "" {
		shown += " [" + def + "]"
	}
	fmt.Fprint(p.out, shown+" ")
	line, err := p.in.line()
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if line == "" {
		if def != "" {
			return def, nil
		}
		return "", needsInput(step, flag)
	}
	return line, nil
}

// Hidden asks text on the terminal with its echo off; off a terminal it answers "".
func (p *prompter) Hidden(text string) (string, error) {
	if !p.tty {
		return "", nil
	}
	fmt.Fprint(p.out, text+" ")
	if p.echoOff != nil {
		if restore := p.echoOff(); restore != nil {
			defer func() {
				restore()
				fmt.Fprintln(p.out)
			}()
		}
	}
	line, err := p.in.line()
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return line, nil
}

// Pause prints text and waits for a line on stdin, or its end; --yes does not wait.
func (p *prompter) Pause(text string) error {
	if p.yes {
		return nil
	}
	fmt.Fprintln(p.out, text)
	_, err := p.in.line()
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// Confirm asks a yes-or-no question whose default is no; --yes answers yes; off a terminal it
// is refused needs-input-<step>, pass --yes.
func (p *prompter) Confirm(step, text string) (bool, error) {
	if p.yes {
		return true, nil
	}
	if !p.tty {
		return false, needsInput(step, "yes")
	}
	fmt.Fprint(p.out, text+" ")
	line, err := p.in.line()
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	return strings.EqualFold(line, "y") || strings.EqualFold(line, "yes"), nil
}

// All reads the rest of stdin, one trailing newline dropped: `setup git-credential`'s value off
// a terminal.
func (p *prompter) All() (string, error) {
	b, err := io.ReadAll(p.in.r)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r"), nil
}

// lineReader reads stdin a line at a time, created on first use, so a run that asks nothing
// never touches stdin.
type lineReader struct {
	src io.Reader
	r   *bufio.Reader
}

func newLineReader(src io.Reader) *lineReader {
	if src == nil {
		src = strings.NewReader("")
	}
	return &lineReader{src: src, r: bufio.NewReader(src)}
}

// line is the next line without its newline; at the end of input the text before it and io.EOF.
func (l *lineReader) line() (string, error) {
	s, err := l.r.ReadString('\n')
	s = strings.TrimRight(s, "\r\n")
	if err != nil && s != "" && errors.Is(err, io.EOF) {
		return s, nil
	}
	return s, err
}
