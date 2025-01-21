import whisper
import sys
import json

# Load Whisper model
model = whisper.load_model("base")

def transcribe_realtime(audio_bytes: bytes):
    """
    Transcribes audio bytes and returns the text.
    """
    return model.transcribe(audio_bytes, language="en")["text"]

def transcribe_batch(audio_path: str):
    """
    Transcribes a batch audio file and returns the text.
    """
    return model.transcribe(audio_path, language="en")["text"]

if __name__ == "__main__":
    mode = sys.argv[1]  # "realtime" or "batch"
    if mode == "realtime":
        audio_data = sys.stdin.buffer.read()
        transcription = transcribe_realtime(audio_data)
    elif mode == "batch":
        audio_path = sys.argv[2]
        transcription = transcribe_batch(audio_path)
    else:
        raise ValueError("Unsupported mode. Use 'realtime' or 'batch'.")

    # Return transcription result as JSON
    print(json.dumps({"transcription": transcription}))
