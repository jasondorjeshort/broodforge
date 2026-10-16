// SPDX-License-Identifier: GPL-3.0-or-later

package control_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
)

func timeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func wantCode(t *testing.T, err error, code string) *control.Error {
	t.Helper()
	var ce *control.Error
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want *control.Error with code %s", err, code)
	}
	if ce.Code != code {
		t.Fatalf("code = %q (%s), want %q", ce.Code, ce.Message, code)
	}
	return ce
}

func TestEchoDefault(t *testing.T) {
	h := controltest.Start(t)
	got, err := control.Do(timeout(t), h.Addr, control.CmdEcho, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"text":"ok"}` {
		t.Fatalf("result = %s", got)
	}
}

func TestEchoStreamsInOrder(t *testing.T) {
	h := controltest.Start(t)
	var events []string
	got, err := control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"events":3,"text":"hi"}`),
		func(ev json.RawMessage) error { events = append(events, string(ev)); return nil })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`{"seq":1,"text":"hi"}`, `{"seq":2,"text":"hi"}`, `{"seq":3,"text":"hi"}`}
	if strings.Join(events, "|") != strings.Join(want, "|") {
		t.Fatalf("events = %q, want %q", events, want)
	}
	if string(got) != `{"text":"hi"}` {
		t.Fatalf("result = %s", got)
	}
}

func TestEchoFailAndBadArgs(t *testing.T) {
	h := controltest.Start(t)
	_, err := control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"fail":"busy"}`), nil)
	if ce := wantCode(t, err, "busy"); ce.Message != "echo was asked to fail" {
		t.Errorf("message = %q", ce.Message)
	}
	_, err = control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"fail":"Not:Kebab"}`), nil)
	wantCode(t, err, control.CodeBadArgs)
	_, err = control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"events":100000}`), nil)
	wantCode(t, err, control.CodeBadArgs)
	_, err = control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"events":"x"}`), nil)
	wantCode(t, err, control.CodeBadArgs)
}

func TestUnknownCommand(t *testing.T) {
	h := controltest.Start(t)
	_, err := control.Do(timeout(t), h.Addr, "nope.nope", nil, nil)
	if ce := wantCode(t, err, control.CodeUnknownCommand); ce.Message != "nope.nope" {
		t.Errorf("message = %q", ce.Message)
	}
}

func TestNotRunning(t *testing.T) {
	_, _, addr := controltest.Root(t)
	_, err := control.Do(timeout(t), addr, control.CmdEcho, nil, nil)
	if ce := wantCode(t, err, control.CodeNotRunning); ce.Message != "no bot at "+addr {
		t.Errorf("message = %q", ce.Message)
	}
	_, err = control.Dial(timeout(t), addr)
	wantCode(t, err, control.CodeNotRunning)
}

func TestSecondServerRefused(t *testing.T) {
	h := controltest.Start(t)
	_, err := control.Listen(h.Addr, control.NewRegistry())
	if ce := wantCode(t, err, control.CodeStartControl); ce.Message != "another bot owns "+h.Addr {
		t.Errorf("message = %q", ce.Message)
	}
	// The first server still answers.
	if _, err := control.Do(timeout(t), h.Addr, control.CmdEcho, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAddressRefusesBadInstanceName(t *testing.T) {
	for _, name := range []string{"", "a/b", `a\b`, "-x", strings.Repeat("a", 65), "sp ace"} {
		if _, err := control.Address(t.TempDir(), name); err == nil {
			t.Errorf("Address(%q) accepted", name)
		}
	}
	for _, name := range []string{"pc", "scratch-0123456789ab", "A.b_c-1"} {
		if _, err := control.Address(t.TempDir(), name); err != nil {
			t.Errorf("Address(%q): %v", name, err)
		}
	}
}

// rawLines writes in and reads n lines back.
func rawLines(t *testing.T, addr, in string, n int) []string {
	t.Helper()
	c, err := control.Dial(timeout(t), addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := io.WriteString(c, in); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(c)
	var out []string
	for range n {
		l, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("after %q: %v", out, err)
		}
		out = append(out, l)
	}
	return out
}

func TestWireBytes(t *testing.T) {
	h := controltest.Start(t)
	got := rawLines(t, h.Addr, `{"id":7,"cmd":"control.echo","args":{"events":1}}`+"\n", 2)
	want := []string{
		`{"id":7,"event":{"seq":1,"text":"ok"}}` + "\n",
		`{"id":7,"ok":true,"result":{"text":"ok"}}` + "\n",
	}
	if strings.Join(got, "") != strings.Join(want, "") {
		t.Fatalf("got %q, want %q", got, want)
	}
	got = rawLines(t, h.Addr, `{"id":9,"cmd":"control.echo","args":{"fail":"x-y"}}`+"\n", 1)
	if got[0] != `{"id":9,"ok":false,"error":{"code":"x-y","message":"echo was asked to fail"}}`+"\n" {
		t.Fatalf("got %q", got)
	}
	got = rawLines(t, h.Addr, "not json\n", 1)
	if !strings.HasPrefix(got[0], `{"id":0,"ok":false,"error":{"code":"bad-request","message":`) {
		t.Fatalf("got %q", got)
	}
	got = rawLines(t, h.Addr, `{"id":3}`+"\n", 1)
	if got[0] != `{"id":3,"ok":false,"error":{"code":"bad-request","message":"no cmd"}}`+"\n" {
		t.Fatalf("got %q", got)
	}
}

// block registers test.block, which waits for its context and reports the cancellation.
func block(cancelled chan<- struct{}) func(*control.Registry) {
	return func(r *control.Registry) {
		r.Handle("test.block", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			if err := call.Emit(json.RawMessage(`{"started":true}`)); err != nil {
				return nil, err
			}
			<-ctx.Done()
			cancelled <- struct{}{}
			return nil, ctx.Err()
		})
	}
}

func TestOneConnectionCarriesConcurrentIDs(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	h := controltest.Start(t, block(cancelled))
	in := `{"id":1,"cmd":"test.block"}` + "\n" + `{"id":2,"cmd":"control.echo"}` + "\n"
	got := rawLines(t, h.Addr, in, 2)
	joined := strings.Join(got, "")
	if !strings.Contains(joined, `{"id":1,"event":{"started":true}}`) || !strings.Contains(joined, `{"id":2,"ok":true,"result":{"text":"ok"}}`) {
		t.Fatalf("got %q", got)
	}
	select { // rawLines closed the connection: the blocked call is cancelled
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("blocked call not cancelled by the disconnect")
	}
}

func TestClientCancelEndsServerCall(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	h := controltest.Start(t, block(cancelled))
	ctx, cancel := context.WithCancel(context.Background())
	_, err := control.Do(ctx, h.Addr, "test.block", nil, func(json.RawMessage) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("server call not cancelled")
	}
}

func TestCloseCancelsInFlight(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	_, _, addr := controltest.Root(t)
	srv, err := control.Listen(addr, func() *control.Registry { r := control.NewRegistry(); block(cancelled)(r); return r }())
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve() }()
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := control.Do(context.Background(), addr, "test.block", nil, func(json.RawMessage) error { close(started); return nil })
		done <- err
	}()
	<-started
	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve: %v", err)
	}
	<-cancelled
	err = <-done
	var ce *control.Error
	if !errors.As(err, &ce) || (ce.Code != control.CodeConnectionLost && ce.Code != control.CodeInternal) {
		t.Fatalf("client err = %v", err)
	}
	if _, err := control.Do(timeout(t), addr, control.CmdEcho, nil, nil); err == nil {
		t.Fatal("closed server still answers")
	} else {
		wantCode(t, err, control.CodeNotRunning)
	}
}

func TestPanicIsInternal(t *testing.T) {
	h := controltest.Start(t, func(r *control.Registry) {
		r.Handle("test.panic", func(context.Context, *control.Call) (json.RawMessage, error) { panic("boom") })
	})
	_, err := control.Do(timeout(t), h.Addr, "test.panic", nil, nil)
	if ce := wantCode(t, err, control.CodeInternal); ce.Message != "panic: boom" {
		t.Errorf("message = %q", ce.Message)
	}
}

func TestManyClientsAtOnce(t *testing.T) {
	h := controltest.Start(t)
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"events":5}`), nil)
			if err == nil && string(got) != `{"text":"ok"}` {
				err = errors.New("result " + string(got))
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestRegistry(t *testing.T) {
	r := control.NewRegistry()
	r.Handle("a.b", func(context.Context, *control.Call) (json.RawMessage, error) { return nil, nil })
	if got := strings.Join(r.Commands(), ","); got != "a.b,control.echo" {
		t.Fatalf("Commands = %s", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate Handle did not panic")
		}
	}()
	r.Handle("a.b", func(context.Context, *control.Call) (json.RawMessage, error) { return nil, nil })
}

func TestLineKind(t *testing.T) {
	cases := map[string]control.LineKind{
		`{"id":1,"event":{}}`:                                    control.LineEvent,
		`{"id":1,"ok":true,"result":null}`:                       control.LineResult,
		`{"id":1,"ok":true,"result":1}`:                          control.LineResult,
		`{"id":1,"ok":false,"error":{"code":"x","message":"y"}}`: control.LineError,
		`{"id":1}`:            control.LineInvalid,
		`{"id":1,"ok":false}`: control.LineInvalid,
		`{"id":1,"ok":true,"error":{"code":"x","message":"y"}}`: control.LineInvalid,
		`{"id":1,"event":{},"ok":true,"result":1}`:              control.LineInvalid,
	}
	for in, want := range cases {
		var l control.Line
		if err := json.Unmarshal([]byte(in), &l); err != nil {
			t.Fatal(err)
		}
		if got := l.Kind(); got != want {
			t.Errorf("%s: Kind = %d, want %d", in, got, want)
		}
	}
}

func TestOneLineAndValidCode(t *testing.T) {
	if got := control.OneLine("a\r\nb\n  c"); got != "a b c" {
		t.Errorf("OneLine = %q", got)
	}
	for code, ok := range map[string]bool{"ok": true, "handoff-refused-attribution": true, "a:b": false, "A": false, "-a": false, "a-": false, "": false} {
		if control.ValidCode(code) != ok {
			t.Errorf("ValidCode(%q) = %v", code, !ok)
		}
	}
}
