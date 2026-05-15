//go:build !vosk

package stt

import "errors"

// NewVoskTranscriber is available when built with: go build -tags vosk.
func NewVoskTranscriber(_ string, _ float64) (Transcriber, error) {
	return nil, errors.New("vosk support is not compiled; build with -tags vosk after installing libvosk")
}
