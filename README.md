# ⚡ HyperStream

**HyperStream** is a cross-platform WebRTC video and audio streaming server written in Go.
Stream your screen, webcam, or any video file directly to one or more browsers — no plugins,
no accounts, no external relay required.

---

## Features

- **WebRTC delivery** — streams directly to any modern browser (Chrome, Firefox, Safari, Edge)
- **Multiple capture sources** — screen capture, webcam, video file, or test pattern
- **Cross-platform** — Linux, macOS, Windows (amd64 & arm64)
- **Multi-viewer** — unlimited concurrent browser viewers
- **HTTP signaling** — simple POST-based SDP exchange, no WebSocket library needed
- **Built-in HTML viewer** — embedded in the binary, zero setup for viewers
- **Auto-reconnect** — browser automatically reconnects if the connection drops
- **ffmpeg powered** — leverages ffmpeg for capture and codec encoding (VP8 + Opus)
- **DTLS encrypted** — WebRTC mandates encryption in transit

---

## Requirements

- **Go 1.23+** (to build from source)
- **ffmpeg** installed and available in `PATH`

Install ffmpeg:

```bash
# Debian / Ubuntu
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Windows (Chocolatey)
choco install ffmpeg
```

---

## Installation

### Download a pre-built binary

Download the latest binary for your platform from the
[Releases page](https://github.com/abrekhov/hyperstream/releases):

```bash
# Linux amd64 example
curl -L -o hstream \
  https://github.com/abrekhov/hyperstream/releases/latest/download/hstream_linux_amd64
chmod +x hstream
sudo mv hstream /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/abrekhov/hyperstream.git
cd hyperstream
go build -o hstream
```

Or install to `$GOBIN`:

```bash
go install github.com/abrekhov/hyperstream@latest
```

---

## Usage

### Stream a test pattern (no hardware needed)

```bash
hstream broadcast --source test
```

Open `http://localhost:8080` in a browser to watch the stream.

### Capture and stream your screen

```bash
hstream broadcast --source screen
```

### Stream from a webcam / camera

```bash
hstream broadcast --source camera
```

### Stream from a video file

```bash
hstream broadcast --source file --file /path/to/video.mp4
```

---

## CLI Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | `8080` | HTTP server port for viewer connections |
| `--source` | `-s` | `screen` | Media source: `screen`, `camera`, `file`, `test` |
| `--file` | `-f` | — | Input video file path (required when `--source=file`) |
| `--width` | `-W` | `1280` | Capture width in pixels |
| `--height` | `-H` | `720` | Capture height in pixels |
| `--framerate` | `-r` | `30` | Capture frame rate (fps) |

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        HyperStream                           │
│                                                              │
│  ┌──────────┐    RTP/UDP     ┌──────────────────────────┐   │
│  │          │  :5004 (video) │                          │   │
│  │  ffmpeg  │ ─────────────► │   Go RTP listener        │   │
│  │          │  :5006 (audio) │   (pkg/rtc)              │   │
│  │  VP8 +   │ ─────────────► │                          │   │
│  │  Opus    │                │   TrackLocalStaticRTP    │   │
│  └──────────┘                │   (video + audio)        │   │
│       ▲                      └──────────┬───────────────┘   │
│       │                                 │ Write()            │
│  Screen / Camera /                      ▼                    │
│  File / Test pattern         ┌──────────────────────────┐   │
│                              │  WebRTC PeerConnections   │   │
│                              │  (one per browser viewer) │   │
│                              └──────────┬───────────────┘   │
│                                         │ SDP via HTTP POST  │
│                                         ▼                    │
│                              ┌──────────────────────────┐   │
│                              │  HTTP signaling server   │   │
│                              │  GET  /      → viewer.html│  │
│                              │  POST /offer → SDP answer │   │
│                              └──────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                                         │
                          WebRTC (DTLS + SRTP)
                                         │
                    ┌────────────────────┼────────────────────┐
                    ▼                    ▼                    ▼
             ┌────────────┐      ┌────────────┐      ┌────────────┐
             │  Browser 1 │      │  Browser 2 │      │  Browser N │
             │  (Chrome)  │      │ (Firefox)  │      │  (Safari)  │
             └────────────┘      └────────────┘      └────────────┘
```

**Flow:**

1. `hstream broadcast` starts ffmpeg which encodes video (VP8) and audio (Opus)
2. ffmpeg sends encoded RTP packets to localhost UDP ports 5004 (video) and 5006 (audio)
3. The Go server listens on those ports and writes packets into shared `TrackLocalStaticRTP` tracks
4. A viewer opens the URL in a browser; the embedded HTML page creates an `RTCPeerConnection`
5. The browser POSTs an SDP offer to `/offer`; the server creates a peer connection, adds the shared tracks, and returns an SDP answer
6. WebRTC DTLS handshake completes; the server forwards RTP packets from the tracks to all connected viewers

---

## Building for all platforms

Requires [GoReleaser](https://goreleaser.com/install/):

```bash
goreleaser build --snapshot --clean
```

Output binaries are placed in `dist/`.

---

## License

Copyright © 2024 Anton Brekhov

Licensed under the [Apache License, Version 2.0](LICENSE).
