#!/usr/bin/env python3
import sys
import argparse

def check_dependencies():
    try:
        import pocketsphinx
    except ImportError:
        sys.stderr.write("Error: pocketsphinx python package is not installed.\n")
        sys.stderr.write("Please run: pip install pocketsphinx\n")
        sys.stderr.write("Note: You may also need to install portaudio: sudo apt install -y libportaudio2\n")
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Pocketsphinx Wake Word Detector")
    parser.add_argument("--phrase", default="hello", help="Wake phrase to listen for")
    parser.add_argument("--threshold", type=float, default=1e-20, help="Keywords spotting threshold (e.g. 1e-20)")
    args = parser.parse_args()

    check_dependencies()

    from pocketsphinx import LiveSpeech

    # Note: LiveSpeech listens to the system default audio recording device
    # Make sure your USB mic is set as the default capture device in Pipewire/Alsa.
    sys.stderr.write(f"Pocketsphinx: Listening for keyphrase '{args.phrase}' (threshold: {args.threshold})...\n")
    sys.stderr.flush()

    try:
        speech = LiveSpeech(
            keyphrase=args.phrase,
            kws_threshold=args.threshold
        )

        for phrase in speech:
            # Print a single line to stdout when heard
            print("WAKE_DETECTED", flush=True)
            sys.stderr.write(f"Pocketsphinx: Wake word '{args.phrase}' detected!\n")
            sys.stderr.flush()
            
    except Exception as e:
        sys.stderr.write(f"Error in Pocketsphinx: {e}\n")
        sys.exit(1)

if __name__ == "__main__":
    main()
