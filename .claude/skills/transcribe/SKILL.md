---
name: transcribe
description: Transcribe audio/voice files using Groq Whisper API
allowed-tools: "Bash(curl *), Bash(/home/openclaw/.local/bin/edge-tts *), Bash(ffmpeg *)"
---

# Transcribe Audio

Transcribe a voice or audio file using the Groq Whisper API, then respond with a voice note followed by a separate text message.

## Step 1: Identify the audio file

Find the audio file path from the conversation context. It will be in a `[File attached: ...]` or `[Replied-to message has file attached: ...]` line. If no audio file is present, tell the user.

## Step 2: Transcribe

Run this curl command, replacing `<FILE_PATH>` with the actual file path:

```bash
curl -s https://api.groq.com/openai/v1/audio/transcriptions \
  -H "Authorization: Bearer $GROQ_API_KEY" \
  -F "file=@<FILE_PATH>" \
  -F "model=whisper-large-v3-turbo" \
  -F "response_format=json"
```

The response is JSON: `{"text": "transcribed content here"}`.

No `language` parameter is set, so Whisper auto-detects the language. This supports both English and Spanish voice notes.

If the request fails, check that GROQ_API_KEY is set and the file exists. Report the error to the user.

## Step 3: Formulate your response

Treat the transcribed text as if the user had typed it as a normal text message. Respond in the SAME language the user spoke (English or Spanish).

## Step 4: Generate the voice note

First, write your full response text to a file (use the Write tool) at:

```
/home/openclaw/.miniclaw-tim/workspace/voice-reply.txt
```

Writing to a file avoids shell-escaping problems with quotes, newlines, and long text.

Then generate the audio with edge-tts and convert it to OGG/Opus with ffmpeg. The voice `en-US-AndrewMultilingualNeural` is multilingual and speaks both English and Spanish naturally, so the same voice is used regardless of language. edge-tts outputs MP3 regardless of the file extension, and Telegram voice notes require OGG/Opus, so the ffmpeg conversion is required:

```bash
/home/openclaw/.local/bin/edge-tts \
  --voice en-US-AndrewMultilingualNeural \
  --file /home/openclaw/.miniclaw-tim/workspace/voice-reply.txt \
  --write-media /home/openclaw/.miniclaw-tim/workspace/voice-reply.mp3

ffmpeg -y -loglevel error \
  -i /home/openclaw/.miniclaw-tim/workspace/voice-reply.mp3 \
  -c:a libopus -b:a 32k -ar 48000 \
  /home/openclaw/.miniclaw-tim/workspace/voice-reply.ogg
```

## Step 5: Queue the voice note in the outbox

Write the outbox to `/home/openclaw/.miniclaw-tim/outbox.json` with an EMPTY caption:

```json
[
  {
    "path": "/home/openclaw/.miniclaw-tim/workspace/voice-reply.ogg",
    "type": "voice",
    "caption": ""
  }
]
```

miniclaw sends the outbox voice note first, then your text response as a separate message. The caption is left empty so your full reply goes in the text message and is not capped by Telegram's 1024-character caption limit.

## Step 6: Send your text response

Output your full response text as your normal reply. miniclaw sends it as a separate text message right after the voice note.

### Notes
- Keep the spoken voice reply natural and reasonably concise; the text message can be fuller
- edge-tts is NOT on PATH, so always call it by its full path `/home/openclaw/.local/bin/edge-tts`
- If edge-tts or ffmpeg fails, send text only and mention the voice note is unavailable
