package tts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"assistance/internal/llm"
)

func TestKokoroSpeaker(t *testing.T) {
	// Check if python is available in PATH, if not skip
	_, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available in PATH, skipping TestKokoroSpeaker")
	}

	// Create a dummy python script that writes a mock wav file
	tmpDir := t.TempDir()
	dummyScriptPath := filepath.Join(tmpDir, "dummy_kokoro.py")
	dummyScript := `
import argparse
import sys

parser = argparse.ArgumentParser()
parser.add_argument("--text")
parser.add_argument("--output")
parser.add_argument("--voice")
parser.add_argument("--model")
parser.add_argument("--voices")
args = parser.parse_args()

# Write dummy content to output wav path
with open(args.output, "w") as f:
    f.write("mock audio content")
`
	if err := os.WriteFile(dummyScriptPath, []byte(dummyScript), 0644); err != nil {
		t.Fatal(err)
	}

	// For Windows, playCommand should do nothing. "cmd /c exit 0" works.
	// For Unix/macOS, "true" works.
	playCmd := "cmd /c exit 0"
	if os.PathSeparator != '\\' {
		playCmd = "true"
	}

	speaker := NewKokoroSpeaker(KokoroSpeakerConfig{
		PythonPath:  "python",
		ScriptPath:  dummyScriptPath,
		ModelPath:   "dummy_model.onnx",
		VoicesPath:  "dummy_voices.bin",
		VoiceName:   "af_bella",
		PlayCommand: playCmd,
	})

	ctx := context.Background()
	reply := llm.AssistantReply{
		Text: "Hello world",
	}

	err = speaker.Speak(ctx, reply)
	if err != nil {
		t.Fatalf("Speak() unexpected error: %v", err)
	}
}
