#!/usr/bin/env python3
import sys
import argparse

def check_dependencies():
    try:
        import openwakeword
        import pvrecorder
    except ImportError:
        sys.stderr.write("Error: openwakeword or pvrecorder python packages are not installed.\n")
        sys.stderr.write("Please run: pip3 install openwakeword pvrecorder\n")
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="OpenWakeWord Free Detector")
    parser.add_argument("--keyword", default="hey_google", help="Pre-trained word: 'hey_google', 'alexa', 'hey_jarvis', 'ok_nabu'")
    parser.add_argument("--threshold", type=float, default=0.5, help="Confidence threshold between 0.0 and 1.0 (Default: 0.5)")
    args = parser.parse_args()

    check_dependencies()

    from openwakeword.model import Model
    from pvrecorder import PvRecorder

    sys.stderr.write(f"OpenWakeWord: Initializing engine for '{args.keyword}'...\n")
    sys.stderr.flush()

    try:
        # Initialize the engine targeting your specific model phrase
        # openWakeWord automatically fetches its internal pre-trained models
        oww_model = Model(wakeword_models=[args.keyword])
    except Exception as e:
        sys.stderr.write(f"Failed to initialize OpenWakeWord model: {e}\n")
        sys.stderr.write("Hint: Make sure the keyword string matches openWakeWord defaults exactly.\n")
        sys.exit(1)

    # openWakeWord expects 16KHz audio chunks of 1280 samples (80ms)
    recorder = PvRecorder(device_index=-1, frame_length=1280)
    
    sys.stderr.write(f"OpenWakeWord: Listening for '{args.keyword}' (threshold: {args.threshold})...\n")
    sys.stderr.flush()

    try:
        recorder.start()
        while True:
            # Grab audio chunk from the default microphone
            audio_frame = recorder.read()
            
            # Feed the raw 16-bit PCM data directly to the model
            # oww_model handles internal downsampling/spectrogram extraction automatically
            prediction = oww_model.predict(audio_frame)
            
            # Get the confidence score for our selected keyword
            confidence = prediction[args.keyword]
            
            if confidence >= args.threshold:
                # Print a single line to stdout when heard (matching your exact interface behavior)
                print("WAKE_DETECTED", flush=True)
                sys.stderr.write(f"OpenWakeWord: Wake word '{args.keyword}' detected! (Score: {confidence:.2f})\n")
                sys.stderr.flush()
                
                # Reset model state after a positive trigger to prevent back-to-back duplicate firings
                oww_model.reset()
                
    except KeyboardInterrupt:
        sys.stderr.write("\nStopping wake word engine safely...\n")
    except Exception as e:
        sys.stderr.write(f"Error in OpenWakeWord Stream: {e}\n")
    finally:
        recorder.delete()

if __name__ == "__main__":
    main()
