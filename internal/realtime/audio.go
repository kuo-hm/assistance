package realtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// ChunkReader streams raw PCM chunks.
type ChunkReader struct {
	reader    io.Reader
	chunkSize int
}

// NewChunkReader creates a chunk reader for streaming audio.
func NewChunkReader(reader io.Reader, chunkSize int) *ChunkReader {
	return &ChunkReader{reader: reader, chunkSize: chunkSize}
}

// ReadChunk reads one chunk, returning a short final chunk when available.
func (r *ChunkReader) ReadChunk() ([]byte, error) {
	if r.chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be positive")
	}
	buf := make([]byte, r.chunkSize)
	n, err := r.reader.Read(buf)
	if n > 0 {
		return buf[:n], nil
	}
	return nil, err
}

// InputStream owns an external audio capture command.
type InputStream struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

// StartInputStream starts an audio capture command and returns raw PCM stdout.
func StartInputStream(ctx context.Context, command string, sampleRate int) (*InputStream, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("realtime input command is required")
	}
	cmd := shellCommand(ctx, expandAudioCommand(command, sampleRate, ""))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open input command stdout: %w", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start input command: %w", err)
	}
	return &InputStream{cmd: cmd, stdout: stdout}, nil
}

// Reader returns the raw PCM stream.
func (s *InputStream) Reader() io.Reader {
	return s.stdout
}

// Close stops the capture command.
func (s *InputStream) Close() error {
	if s == nil {
		return nil
	}
	if s.stdout != nil {
		_ = s.stdout.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		return s.cmd.Wait()
	}
	return nil
}

// OutputPlayer plays PCM chunks through a persistent command stdin.
type OutputPlayer struct {
	command    string
	sampleRate int
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	mu         sync.Mutex
}

// NewOutputPlayer creates a command-backed player.
func NewOutputPlayer(command string, sampleRate int) *OutputPlayer {
	return &OutputPlayer{command: command, sampleRate: sampleRate}
}

// Play writes one PCM chunk to the player. Commands containing {input} use a
// temporary file fallback; other commands receive raw PCM through stdin.
func (p *OutputPlayer) Play(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if strings.TrimSpace(p.command) == "" {
		return fmt.Errorf("realtime output command is required")
	}
	if !strings.Contains(p.command, "{input}") {
		return p.writeStream(ctx, data)
	}
	file, err := os.CreateTemp("", "assistant-live-*.pcm")
	if err != nil {
		return fmt.Errorf("create realtime audio temp file: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write realtime audio temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close realtime audio temp file: %w", err)
	}
	defer os.Remove(path)

	cmd := shellCommand(ctx, expandAudioCommand(p.command, p.sampleRate, path))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("play realtime audio: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Close stops a persistent streaming player if one was started.
func (p *OutputPlayer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var err error
	if p.stdin != nil {
		err = p.stdin.Close()
		p.stdin = nil
	}
	if p.cmd != nil && p.cmd.Process != nil {
		if killErr := p.cmd.Process.Kill(); killErr != nil && err == nil {
			err = killErr
		}
		if waitErr := p.cmd.Wait(); waitErr != nil && err == nil {
			err = waitErr
		}
		p.cmd = nil
	}
	return err
}

func (p *OutputPlayer) writeStream(ctx context.Context, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdin == nil {
		cmd := shellCommand(ctx, expandAudioCommand(p.command, p.sampleRate, ""))
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("open output command stdin: %w", err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			_ = stdin.Close()
			return fmt.Errorf("start output command: %w", err)
		}
		p.cmd = cmd
		p.stdin = stdin
	}
	if _, err := p.stdin.Write(data); err != nil {
		return fmt.Errorf("write realtime audio stream: %w", err)
	}
	return nil
}

func expandAudioCommand(command string, sampleRate int, inputPath string) string {
	command = strings.ReplaceAll(command, "{sample_rate}", strconv.Itoa(sampleRate))
	command = strings.ReplaceAll(command, "{input}", inputPath)
	return command
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if os.PathSeparator == '\\' {
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}
