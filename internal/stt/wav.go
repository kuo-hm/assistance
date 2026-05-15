package stt

import (
	"encoding/binary"
	"errors"
)

func pcmFromWAV(data []byte) ([]byte, int32, error) {
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return data, 0, nil
	}

	var sampleRate int32
	for offset := 12; offset+8 <= len(data); {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(data) {
			return nil, 0, errors.New("invalid wav chunk size")
		}

		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				sampleRate = int32(binary.LittleEndian.Uint32(data[chunkStart+4 : chunkStart+8]))
			}
		case "data":
			return data[chunkStart:chunkEnd], sampleRate, nil
		}

		offset = chunkEnd
		if offset%2 == 1 {
			offset++
		}
	}
	return nil, 0, errors.New("wav data chunk not found")
}
