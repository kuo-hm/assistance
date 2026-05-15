package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// AudioClip is recorded audio ready for STT.
type AudioClip struct {
	Data            []byte
	MIMEType        string
	SampleRateHertz int32
	Path            string
}

// RecordOptions controls one user utterance capture.
type RecordOptions struct {
	SilenceTimeout time.Duration
}

// Recorder records user speech until silence or configured command completion.
type Recorder interface {
	RecordUntilSilence(ctx context.Context, opts RecordOptions) (AudioClip, error)
}

// ExternalRecorder records audio by running a command that writes a WAV file.
type ExternalRecorder struct {
	command       string
	recordSeconds int
}

// NewExternalRecorder creates a recorder backed by a command such as arecord.
func NewExternalRecorder(command string, recordSeconds int) *ExternalRecorder {
	return &ExternalRecorder{command: command, recordSeconds: recordSeconds}
}

// RecordUntilSilence records one utterance. If no command is configured, it returns an empty clip for console STT mode.
func (r *ExternalRecorder) RecordUntilSilence(ctx context.Context, _ RecordOptions) (AudioClip, error) {
	if r.command == "" {
		return AudioClip{MIMEType: "audio/wav", SampleRateHertz: 16000}, nil
	}

	file, err := os.CreateTemp("", "assistant-recording-*.wav")
	if err != nil {
		return AudioClip{}, fmt.Errorf("create temp audio file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return AudioClip{}, fmt.Errorf("close temp audio file: %w", err)
	}

	command := strings.ReplaceAll(r.command, "{output}", path)
	command = strings.ReplaceAll(command, "{seconds}", strconv.Itoa(r.recordSeconds))
	data, err := os.ReadFile(path)
	if commandErr := runShell(ctx, command); commandErr != nil {
		data, err = os.ReadFile(path)
		if err != nil || len(data) == 0 {
			return AudioClip{}, commandErr
		}
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return AudioClip{}, fmt.Errorf("read recorded audio: %w", err)
	}
	if len(data) == 0 {
		return AudioClip{}, errors.New("record command produced empty audio")
	}

	return AudioClip{
		Data:            data,
		MIMEType:        "audio/wav",
		SampleRateHertz: 16000,
		Path:            path,
	}, nil
}

func runShell(ctx context.Context, command string) error {
	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("audio command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
