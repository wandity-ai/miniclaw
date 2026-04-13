---
name: transcribe
description: Transcribe audio/voice files using Groq Whisper API
allowed-tools: "Bash(curl *)"
---

# Transcribe Audio

Transcribe a voice or audio file using the Groq Whisper API, then respond with both a voice note and text.

## Step 1: Identify the audio file

Find the audio file path from the conversation context. It will be in a `[File attached: ...]` or `[Replied-to message has file attached: ...]` line. If no audio file is present, tell the user.

## Step 2: Transcribe

Run this curl command, replacing `<FILE_PATH>` with the actual file path:

```bash
curl -s https://api.groq.com/openai/v1/audio/transcriptions \
  -H "Authorization: Bearer $GROQ_API_KEY" \
  -F "file=@<FILE_PATH>" \
  -F "model=whisper-large-v3-turbo" \
  -F "response_format=json" \
  -F "language=en"
```

The response is JSON: `{"text": "transcribed content here"}`.

If the request fails, check that GROQ_API_KEY is set and the file exists. Report the error to the user.

## Step 3: Formulate your response

Treat the transcribed text as if the user had typed it as a normal text message. Formulate your response text.

## Step 4: Send a voice reply

Generate a voice note from your response text using edge-tts, then queue it in the outbox BEFORE your text response.

```bash
/home/openclaw/.local/bin/edge-tts \
  --voice en-US-MichelleNeural \
  --text "YOUR_RESPONSE_TEXT" \
  --write-media /home/openclaw/.miniclaw/workspace/voice-reply.ogg
```

Then write the outbox to `/home/openclaw/.miniclaw/outbox.json` (BEFORE your text response):

```json
[
  {
    "path": "/home/openclaw/.miniclaw/workspace/voice-reply.ogg",
    "type": "voice",
    "caption": "YOUR_RESPONSE_TEXT"
  }
]
```

The caption is your response text (same as what you speak). This makes the voice note self-contained — no separate text message needed.

## Step 5: Done

Do NOT write a separate text response after sending a voice note. The caption on the voice note contains your reply.

### Notes
- Keep voice replies concise - trim the audio version if the response is long, but caption can be fuller
- If edge-tts fails, send text only and mention TTS is unavailable
