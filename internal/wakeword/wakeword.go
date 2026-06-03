package wakeword

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"assistance/internal/audio"
	"assistance/internal/consoleio"
	"assistance/internal/stt"
)

// WakeEvent is emitted when the assistant should start listening.
type WakeEvent struct {
	Phrase     string
	DetectedAt time.Time
}

// VoiceConfig controls STT-backed phrase wake detection.
type VoiceConfig struct {
	Phrase        string
	Aliases       []string
	Languages     []string
	RecordSeconds int
	MinConfidence float32
	PollDelay     time.Duration
	Debug         bool
}

// VoiceDetector records short audio windows and wakes when STT hears the phrase.
type VoiceDetector struct {
	recorder    audio.Recorder
	transcriber stt.Transcriber
	config      VoiceConfig
}

// NewVoiceDetector creates an STT-backed wake detector.
func NewVoiceDetector(recorder audio.Recorder, transcriber stt.Transcriber, config VoiceConfig) *VoiceDetector {
	if config.RecordSeconds <= 0 {
		config.RecordSeconds = 2
	}
	if config.PollDelay <= 0 {
		config.PollDelay = 250 * time.Millisecond
	}
	return &VoiceDetector{recorder: recorder, transcriber: transcriber, config: config}
}

// Listen emits a wake event when a recorded clip contains the configured phrase.
func (d *VoiceDetector) Listen(ctx context.Context) (<-chan WakeEvent, error) {
	if d.recorder == nil {
		return nil, fmt.Errorf("voice wake recorder is required")
	}
	if d.transcriber == nil {
		return nil, fmt.Errorf("voice wake transcriber is required")
	}
	if strings.TrimSpace(d.config.Phrase) == "" {
		return nil, fmt.Errorf("voice wake phrase is required")
	}

	events := make(chan WakeEvent)
	go func() {
		defer close(events)
		targets := normalizePhrases(d.config.Phrase, d.config.Aliases)
		if d.config.Debug {
			fmt.Printf("wake match phrases: %q\n", targets)
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			fmt.Printf("Wake Word Detector: Listening for %q (or aliases %q)...\n", d.config.Phrase, d.config.Aliases)
			clip, err := d.recorder.RecordUntilSilence(ctx, audio.RecordOptions{})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Warn("voice wake recording failed", "error", err)
				sleepOrDone(ctx, d.config.PollDelay)
				continue
			}

			fmt.Println("Wake Word Detector: Analyzing captured audio...")
			transcript, err := d.transcriber.Transcribe(ctx, clip, d.config.Languages)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Debug("voice wake transcription skipped", "error", err)
				fmt.Printf("Wake Word Detector: Failed to recognize speech (error: %v)\n", err)
				sleepOrDone(ctx, d.config.PollDelay)
				continue
			}

			fmt.Printf("Wake Word Detector heard: %q (confidence: %.2f, language: %q)\n", transcript.Text, transcript.Confidence, transcript.Language)
			if d.config.MinConfidence > 0 && transcript.Confidence > 0 && transcript.Confidence < d.config.MinConfidence {
				fmt.Printf("Wake Word Detector: Ignored due to low confidence (below min: %.2f)\n", d.config.MinConfidence)
				continue
			}
			heard := normalizePhrase(transcript.Text)
			if matchesAnyPhrase(heard, targets) {
				fmt.Println("SUCCESS: Wake phrase matched!")
				select {
				case events <- WakeEvent{Phrase: d.config.Phrase, DetectedAt: time.Now()}:
				case <-ctx.Done():
					return
				}
			} else if heard != "" {
				fmt.Printf("Wake Word Detector: Heard %q but it did not match targets\n", heard)
			}
		}
	}()
	return events, nil
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

func normalizePhrase(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = nonWord.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func normalizePhrases(phrase string, aliases []string) []string {
	values := make([]string, 0, len(aliases)+1)
	for _, value := range append([]string{phrase}, aliases...) {
		normalized := normalizePhrase(value)
		if normalized != "" {
			values = append(values, normalized)
		}
	}
	return values
}

func matchesAnyPhrase(heard string, targets []string) bool {
	if heard == "" {
		return false
	}
	for _, target := range targets {
		if heard == target || strings.Contains(heard, target) {
			return true
		}
	}
	return false
}

func sleepOrDone(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
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
