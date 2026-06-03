#!/usr/bin/env python3
import os
import sys
import argparse
import urllib.request

def check_dependencies():
    missing = []
    try:
        import kokoro_onnx
    except ImportError:
        missing.append("kokoro-onnx")
    try:
        import soundfile
    except ImportError:
        missing.append("soundfile")
    
    if missing:
        sys.stderr.write(f"Error: Missing Python dependencies: {', '.join(missing)}\n")
        sys.stderr.write("Please run: pip install kokoro-onnx soundfile\n")
        sys.exit(1)

def download_file(url, destination):
    if os.path.exists(destination):
        return

    # Ensure parent directory exists
    parent_dir = os.path.dirname(destination)
    if parent_dir and not os.path.exists(parent_dir):
        os.makedirs(parent_dir, exist_ok=True)

    print(f"Downloading {os.path.basename(destination)} from {url}...")
    try:
        # Use urllib to download
        with urllib.request.urlopen(url) as response, open(destination, "wb") as out_file:
            # Copy data in chunks with basic progress output
            length = response.getheader('content-length')
            if length:
                length = int(length)
                block_size = 1024 * 1024
                downloaded = 0
                while True:
                    buffer = response.read(block_size)
                    if not buffer:
                        break
                    downloaded += len(buffer)
                    out_file.write(buffer)
                    percent = (downloaded / length) * 100
                    sys.stdout.write(f"\rDownloading: {percent:.1f}% ({downloaded // (1024*1024)}MB / {length // (1024*1024)}MB)")
                    sys.stdout.flush()
                print()
            else:
                out_file.write(response.read())
        print(f"Successfully downloaded {os.path.basename(destination)}.")
    except Exception as e:
        sys.stderr.write(f"Error downloading {url}: {e}\n")
        if os.path.exists(destination):
            os.remove(destination)
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Kokoro TTS CLI Wrapper")
    parser.add_argument("--text", required=True, help="Text to speak")
    parser.add_argument("--output", required=True, help="Output path for WAV file")
    parser.add_argument("--voice", default="af_bella", help="Voice name (e.g. af_bella, am_adam)")
    parser.add_argument("--model", default="models/kokoro-v1.0.onnx", help="Path to kokoro ONNX model file")
    parser.add_argument("--voices", default="models/voices-v1.0.bin", help="Path to voices BIN file")
    args = parser.parse_args()

    # Define URLs for auto-downloading if missing
    model_url = "https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/kokoro-v1.0.onnx"
    voices_url = "https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/voices-v1.0.bin"

    # Download model files if they don't exist
    if not os.path.exists(args.model):
        download_file(model_url, args.model)
    if not os.path.exists(args.voices):
        download_file(voices_url, args.voices)

    # Verify dependencies are installed before importing
    check_dependencies()

    from kokoro_onnx import Kokoro
    import soundfile as sf

    try:
        # Load the model
        kokoro = Kokoro(args.model, args.voices)
        
        # Perform synthesis
        samples, sample_rate = kokoro.create(args.text, voice=args.voice, speed=1.0, lang="en-us")
        
        # Ensure target output directory exists
        out_dir = os.path.dirname(args.output)
        if out_dir and not os.path.exists(out_dir):
            os.makedirs(out_dir, exist_ok=True)
            
        # Write wav file
        sf.write(args.output, samples, sample_rate)
        print(f"Generated TTS output saved to {args.output}")
    except Exception as e:
        sys.stderr.write(f"Error during TTS generation: {e}\n")
        sys.exit(1)

if __name__ == "__main__":
    main()
