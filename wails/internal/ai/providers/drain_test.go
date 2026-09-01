package providers

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestDrainLines_BasicLines(t *testing.T) {
	var got []string
	DrainLines(strings.NewReader("a\nb\nc\n"), func(l string) { got = append(got, l) })
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("got %v", got)
	}
}

func TestDrainLines_NoTrailingNewline(t *testing.T) {
	var got []string
	DrainLines(strings.NewReader("a\nb"), func(l string) { got = append(got, l) })
	if strings.Join(got, ",") != "a,b" {
		t.Fatalf("got %v", got)
	}
}

func TestDrainLines_CRLFAndEmptyLines(t *testing.T) {
	var got []string
	DrainLines(strings.NewReader("a\r\n\r\nb\r\n"), func(l string) { got = append(got, l) })
	if len(got) != 3 || got[0] != "a" || got[1] != "" || got[2] != "b" {
		t.Fatalf("got %#v", got)
	}
}

func TestDrainLines_Empty(t *testing.T) {
	called := false
	DrainLines(strings.NewReader(""), func(l string) { called = true })
	if called {
		t.Fatal("callback should not fire on empty input")
	}
}

// TestDrainLines_OverLongLineDoesNotStall is the core regression: a single
// line far longer than any scanner buffer must be truncated (not dropped) and
// draining must continue to subsequent lines. A bufio.Scanner would return
// ErrTooLong and stop, leaving the pipe undrained.
func TestDrainLines_OverLongLineDoesNotStall(t *testing.T) {
	huge := strings.Repeat("x", 5*1024*1024) // 5MB, well past maxDrainLine
	input := huge + "\nafter\n"

	done := make(chan struct{})
	var lines []string
	go func() {
		DrainLines(strings.NewReader(input), func(l string) { lines = append(lines, l) })
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("DrainLines stalled on an over-long line")
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if len(lines[0]) != maxDrainLine {
		t.Fatalf("over-long line should be truncated to %d, got %d", maxDrainLine, len(lines[0]))
	}
	if lines[1] != "after" {
		t.Fatalf("draining did not continue past over-long line: %q", lines[1])
	}
}

// slowReader emits its payload one byte per Read to exercise partial reads.
type slowReader struct {
	data []byte
	pos  int
}

func (s *slowReader) Read(p []byte) (int, error) {
	if s.pos >= len(s.data) {
		return 0, io.EOF
	}
	p[0] = s.data[s.pos]
	s.pos++
	return 1, nil
}

func TestDrainLines_PartialReads(t *testing.T) {
	var got []string
	DrainLines(&slowReader{data: []byte("hello\nworld\n")}, func(l string) { got = append(got, l) })
	if strings.Join(got, ",") != "hello,world" {
		t.Fatalf("got %v", got)
	}
}
