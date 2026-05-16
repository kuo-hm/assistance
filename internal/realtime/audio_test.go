package realtime

import (
	"bytes"
	"io"
	"testing"
)

func TestChunkReaderReadChunk(t *testing.T) {
	reader := NewChunkReader(bytes.NewBufferString("abcdef"), 4)

	first, err := reader.ReadChunk()
	if err != nil {
		t.Fatalf("first ReadChunk() error = %v", err)
	}
	if string(first) != "abcd" {
		t.Fatalf("first chunk = %q", first)
	}

	second, err := reader.ReadChunk()
	if err != nil {
		t.Fatalf("second ReadChunk() error = %v", err)
	}
	if string(second) != "ef" {
		t.Fatalf("second chunk = %q", second)
	}

	_, err = reader.ReadChunk()
	if err != io.EOF {
		t.Fatalf("final ReadChunk() error = %v, want EOF", err)
	}
}

func TestChunkReaderRejectsInvalidChunkSize(t *testing.T) {
	reader := NewChunkReader(bytes.NewBufferString("abc"), 0)

	_, err := reader.ReadChunk()
	if err == nil {
		t.Fatal("ReadChunk() error = nil, want error")
	}
}

func TestExpandAudioCommand(t *testing.T) {
	got := expandAudioCommand("aplay -r {sample_rate} {input}", 24000, "/tmp/audio.pcm")
	want := "aplay -r 24000 /tmp/audio.pcm"
	if got != want {
		t.Fatalf("expandAudioCommand() = %q, want %q", got, want)
	}
}
