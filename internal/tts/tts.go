package tts

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"

	"assistance/internal/llm"
)

// Speaker speaks assistant replies.
type Speaker interface {
	Speak(ctx context.Context, reply llm.AssistantReply) error
}

// MultiSpeaker sends replies to multiple outputs.
type MultiSpeaker struct {
	speakers []Speaker
}

// NewMultiSpeaker creates a speaker that writes to every configured output.
func NewMultiSpeaker(speakers ...Speaker) *MultiSpeaker {
	return &MultiSpeaker{speakers: speakers}
}

// Speak sends the reply to all speakers in order.
func (s *MultiSpeaker) Speak(ctx context.Context, reply llm.AssistantReply) error {
	for _, speaker := range s.speakers {
		if err := speaker.Speak(ctx, reply); err != nil {
			return err
		}
	}
	return nil
}

// ConsoleSpeaker prints replies for development.
type ConsoleSpeaker struct {
	out io.Writer
}

// NewConsoleSpeaker creates a console speaker.
func NewConsoleSpeaker(out io.Writer) *ConsoleSpeaker {
	return &ConsoleSpeaker{out: out}
}

// Speak prints the assistant reply.
func (s *ConsoleSpeaker) Speak(_ context.Context, reply llm.AssistantReply) error {
	_, err := fmt.Fprintf(s.out, "assistant: %s\n", reply.Text)
	return err
}

// WindowsSpeaker speaks replies with the built-in Windows speech engine.
type WindowsSpeaker struct{}

// NewWindowsSpeaker creates a local Windows TTS speaker.
func NewWindowsSpeaker() *WindowsSpeaker {
	return &WindowsSpeaker{}
}

// Speak uses PowerShell System.Speech for local TTS.
func (s *WindowsSpeaker) Speak(ctx context.Context, reply llm.AssistantReply) error {
	script := fmt.Sprintf(
		`Add-Type -AssemblyName System.Speech; $s = New-Object System.Speech.Synthesis.SpeechSynthesizer; $s.Speak(%s)`,
		strconv.Quote(reply.Text),
	)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("windows speech: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// GoogleSpeakerConfig configures Google Cloud TTS playback.
type GoogleSpeakerConfig struct {
	CredentialsFile string
	LanguageCode    string
	VoiceName       string
	PlayCommand     string
}

// GoogleSpeaker synthesizes speech and plays it through an external player.
type GoogleSpeaker struct {
	client      *texttospeech.Client
	language    string
	voice       string
	playCommand string
}

// NewGoogleSpeaker creates a Google Cloud TTS speaker.
func NewGoogleSpeaker(ctx context.Context, cfg GoogleSpeakerConfig) (*GoogleSpeaker, error) {
	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}
	client, err := texttospeech.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create text-to-speech client: %w", err)
	}
	if cfg.LanguageCode == "" {
		cfg.LanguageCode = "ar-XA"
	}
	return &GoogleSpeaker{
		client:      client,
		language:    cfg.LanguageCode,
		voice:       cfg.VoiceName,
		playCommand: cfg.PlayCommand,
	}, nil
}

// Speak synthesizes MP3 audio and runs the configured player command.
func (s *GoogleSpeaker) Speak(ctx context.Context, reply llm.AssistantReply) error {
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: reply.Text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: s.language,
			Name:         s.voice,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}
	resp, err := s.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("synthesize speech: %w", err)
	}

	file, err := os.CreateTemp("", "assistant-reply-*.mp3")
	if err != nil {
		return fmt.Errorf("create tts temp file: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(resp.AudioContent); err != nil {
		_ = file.Close()
		return fmt.Errorf("write tts audio: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close tts audio: %w", err)
	}

	command := strings.ReplaceAll(s.playCommand, "{input}", path)
	var cmd *exec.Cmd
	if os.PathSeparator == '\\' {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("play tts audio: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// KokoroSpeakerConfig configures Kokoro TTS playback.
type KokoroSpeakerConfig struct {
	PythonPath  string
	ScriptPath  string
	ModelPath   string
	VoicesPath  string
	VoiceName   string
	PlayCommand string
}

// KokoroSpeaker synthesizes speech locally via Python & ONNX and plays it.
type KokoroSpeaker struct {
	pythonPath  string
	scriptPath  string
	modelPath   string
	voicesPath  string
	voiceName   string
	playCommand string
}

// NewKokoroSpeaker creates a Kokoro Speaker.
func NewKokoroSpeaker(cfg KokoroSpeakerConfig) *KokoroSpeaker {
	if cfg.PythonPath == "" {
		cfg.PythonPath = "python"
	}
	if cfg.ScriptPath == "" {
		cfg.ScriptPath = "scripts/kokoro_tts.py"
	}
	if cfg.ModelPath == "" {
		cfg.ModelPath = "models/kokoro-v1.0.onnx"
	}
	if cfg.VoicesPath == "" {
		cfg.VoicesPath = "models/voices-v1.0.bin"
	}
	if cfg.VoiceName == "" {
		cfg.VoiceName = "af_bella"
	}
	return &KokoroSpeaker{
		pythonPath:  cfg.PythonPath,
		scriptPath:  cfg.ScriptPath,
		modelPath:   cfg.ModelPath,
		voicesPath:  cfg.VoicesPath,
		voiceName:   cfg.VoiceName,
		playCommand: cfg.PlayCommand,
	}
}

// Speak synthesizes text to speech using scripts/kokoro_tts.py and plays it.
func (s *KokoroSpeaker) Speak(ctx context.Context, reply llm.AssistantReply) error {
	// Create a temp file path for output WAV file
	file, err := os.CreateTemp("", "assistant-reply-*.wav")
	if err != nil {
		return fmt.Errorf("create tts temp file: %w", err)
	}
	path := file.Name()
	_ = file.Close()
	// Defer cleanup of the temp file
	defer os.Remove(path)

	// Execute python script
	args := []string{
		s.scriptPath,
		"--text", reply.Text,
		"--output", path,
		"--voice", s.voiceName,
		"--model", s.modelPath,
		"--voices", s.voicesPath,
	}

	cmd := exec.CommandContext(ctx, s.pythonPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run kokoro-tts script: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// Play audio
	command := strings.ReplaceAll(s.playCommand, "{input}", path)
	var playCmd *exec.Cmd
	if os.PathSeparator == '\\' {
		playCmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	} else {
		playCmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	output, err = playCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("play tts audio: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

