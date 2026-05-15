package wakeword

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"assistance/internal/consoleio"
)

// WakeEvent is emitted when the assistant should start listening.
type WakeEvent struct {
	Phrase     string
	DetectedAt time.Time
}

// Detector listens for a wake phrase.
type Detector interface {
	Listen(ctx context.Context) (<-chan WakeEvent, error)
}

// ConsoleDetector waits for Enter. It is the default Windows development mode.
type ConsoleDetector struct {
	lines  *consoleio.LineReader
	out    io.Writer
	phrase string
}

// NewConsoleDetector creates a console wake detector.
func NewConsoleDetector(in io.Reader, out io.Writer, phrase string) *ConsoleDetector {
	return NewConsoleDetectorWithReader(consoleio.NewLineReader(in), out, phrase)
}

// NewConsoleDetectorWithReader creates a console wake detector with a shared reader.
func NewConsoleDetectorWithReader(lines *consoleio.LineReader, out io.Writer, phrase string) *ConsoleDetector {
	return &ConsoleDetector{lines: lines, out: out, phrase: phrase}
}

// Listen emits a wake event each time Enter is pressed.
func (d *ConsoleDetector) Listen(ctx context.Context) (<-chan WakeEvent, error) {
	events := make(chan WakeEvent)
	go func() {
		defer close(events)
		for {
			fmt.Fprintf(d.out, "press Enter after saying %q: ", d.phrase)
			line, err := d.lines.ReadLine(ctx)
			if err != nil {
				return
			}
			if strings.TrimSpace(line) != "" {
				d.lines.PushLine(line)
			}
			select {
			case events <- WakeEvent{Phrase: d.phrase, DetectedAt: time.Now()}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

// CommandDetector wraps an external wake-word command such as a Porcupine demo binary.
type CommandDetector struct {
	command string
	phrase  string
}

// NewCommandDetector creates a wake detector backed by an external command.
func NewCommandDetector(command string, phrase string) *CommandDetector {
	return &CommandDetector{command: command, phrase: phrase}
}

// Listen emits one wake event every time the command prints a line.
func (d *CommandDetector) Listen(ctx context.Context) (<-chan WakeEvent, error) {
	if d.command == "" {
		return nil, fmt.Errorf("wake command is required for command wake provider")
	}

	var cmd *exec.Cmd
	if os.PathSeparator == '\\' {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", d.command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", d.command)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open wake command stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start wake command: %w", err)
	}

	events := make(chan WakeEvent)
	go func() {
		defer close(events)
		defer func() {
			if waitErr := cmd.Wait(); waitErr != nil && ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "wake command stopped: %v\n", waitErr)
			}
		}()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			select {
			case events <- WakeEvent{Phrase: d.phrase, DetectedAt: time.Now()}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}
