package stt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"assistance/internal/audio"
	"assistance/internal/consoleio"
	speech "cloud.google.com/go/speech/apiv1"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"google.golang.org/api/option"
)

// Transcript is the recognized user utterance.
type Transcript struct {
	Text       string
	Language   string
	Confidence float32
}

// Transcriber converts audio to text.
type Transcriber interface {
	Transcribe(ctx context.Context, clip audio.AudioClip, languages []string) (Transcript, error)
}

// ConsoleTranscriber reads typed transcripts for local development.
type ConsoleTranscriber struct {
	lines *consoleio.LineReader
	out   io.Writer
}

// NewConsoleTranscriber creates a console STT adapter.
func NewConsoleTranscriber(in io.Reader, out io.Writer) *ConsoleTranscriber {
	return NewConsoleTranscriberWithReader(consoleio.NewLineReader(in), out)
}

// NewConsoleTranscriberWithReader creates a console STT adapter with a shared reader.
func NewConsoleTranscriberWithReader(lines *consoleio.LineReader, out io.Writer) *ConsoleTranscriber {
	return &ConsoleTranscriber{lines: lines, out: out}
}

// Transcribe reads one text line from stdin.
func (t *ConsoleTranscriber) Transcribe(ctx context.Context, _ audio.AudioClip, languages []string) (Transcript, error) {
	var text string
	for text == "" {
		fmt.Fprint(t.out, "you: ")
		line, err := t.lines.ReadLine(ctx)
		if err != nil {
			return Transcript{}, err
		}
		text = strings.TrimSpace(line)
	}
	language := ""
	if len(languages) > 0 {
		language = languages[0]
	}
	return Transcript{Text: text, Language: language, Confidence: 1}, nil
}

// GoogleTranscriber uses Google Cloud Speech-to-Text.
type GoogleTranscriber struct {
	client *speech.Client
}

// NewGoogleTranscriber creates a Google Cloud STT client.
func NewGoogleTranscriber(ctx context.Context, credentialsFile string) (*GoogleTranscriber, error) {
	opts := []option.ClientOption{}
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}
	client, err := speech.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create speech client: %w", err)
	}
	return &GoogleTranscriber{client: client}, nil
}

// Transcribe sends WAV/LINEAR16 audio to Google Cloud STT.
func (t *GoogleTranscriber) Transcribe(ctx context.Context, clip audio.AudioClip, languages []string) (Transcript, error) {
	if len(clip.Data) == 0 {
		return Transcript{}, errors.New("google STT requires recorded audio data")
	}
	if len(languages) == 0 {
		return Transcript{}, errors.New("google STT requires at least one language")
	}

	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:            clip.SampleRateHertz,
			LanguageCode:               languages[0],
			AlternativeLanguageCodes:   rest(languages),
			EnableAutomaticPunctuation: true,
			Model:                      "latest_long",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: clip.Data},
		},
	}

	resp, err := t.client.Recognize(ctx, req)
	if err != nil {
		return Transcript{}, fmt.Errorf("recognize speech: %w", err)
	}

	var best Transcript
	for _, result := range resp.Results {
		if len(result.Alternatives) == 0 {
			continue
		}
		alt := result.Alternatives[0]
		if best.Text != "" {
			best.Text += " "
		}
		best.Text += strings.TrimSpace(alt.Transcript)
		best.Confidence = max(best.Confidence, alt.Confidence)
		best.Language = result.LanguageCode
	}
	if strings.TrimSpace(best.Text) == "" {
		return Transcript{}, errors.New("speech recognition returned no transcript")
	}
	return best, nil
}

func rest(values []string) []string {
	if len(values) <= 1 {
		return nil
	}
	return values[1:]
}
