package consoleio

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
)

// LineReader serializes console reads so multiple components do not race on stdin.
type LineReader struct {
	scanner *bufio.Scanner
	mu      sync.Mutex
	pending []string
}

// NewLineReader creates a coordinated line reader.
func NewLineReader(in io.Reader) *LineReader {
	return &LineReader{scanner: bufio.NewScanner(in)}
}

// ReadLine reads one line while honoring context cancellation before the read starts.
func (r *LineReader) ReadLine(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pending) > 0 {
		line := r.pending[0]
		r.pending = r.pending[1:]
		return line, nil
	}

	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return "", fmt.Errorf("read console line: %w", err)
		}
		return "", io.EOF
	}
	return r.scanner.Text(), nil
}

// PushLine makes a line available to the next reader call.
func (r *LineReader) PushLine(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pending = append([]string{line}, r.pending...)
}
