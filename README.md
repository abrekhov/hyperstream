<div align="center">

# ⚡ HyperStream

**Broadcast video & audio to any browser via WebRTC — zero plugins, zero accounts, zero servers**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/abrekhov/hyperstream?style=flat-square&color=brightgreen)](https://github.com/abrekhov/hyperstream/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/abrekhov/hyperstream/test.yaml?style=flat-square)](https://github.com/abrekhov/hyperstream/actions)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat-square)](https://github.com/abrekhov/hyperstream/releases)

Stream your **screen**, **webcam**, or **video file** to one or many browsers in seconds.  
No relay server. No accounts. No browser plugins. Just open the URL.

</div>

---

## How it works

```
Your machine                                   Viewer's browser
─────────────────────────────────────────────────────────────────
 ffmpeg (capture)
   │
   │  VP8 + Opus RTP
   │  over localhost UDP
   ▼
 hstream (Go server)  ──── WebRTC (DTLS/SRTP) ───►  Chrome / Firefox
   │                  ──── WebRTC (DTLS/SRTP) ───►  Safari
   │                  ──── WebRTC (DTLS/SRTP) ───►  Edge
   │
   └── HTTP :8080
         GET  /        → embedded HTML viewer
         POST /offer   → SDP handshake
```

ffmpeg captures and encodes locally → packets flow into Go → WebRTC delivers encrypted streams to every browser that opens the page.

---

## Quick start

### 1. Install ffmpeg

```bash
# Ubuntu / Debian
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Windows (Chocolatey)
choco install ffmpeg

# Windows (Scoop)
scoop install ffmpeg
```

### 2. Install hstream

**Download a pre-built binary** from [Releases](https://github.com/abrekhov/hyperstream/releases):

```bash
# Linux amd64
curl -L -o hstream \
  https://github.com/abrekhov/hyperstream/releases/latest/download/hstream_linux_amd64
chmod +x hstream && sudo mv hstream /usr/local/bin/

# macOS arm64 (Apple Silicon)
curl -L -o hstream \
  https://github.com/abrekhov/hyperstream/releases/latest/download/hstream_darwin_arm64
chmod +x hstream && sudo mv hstream /usr/local/bin/
```

Or **build from source** (requires Go 1.23+):

```bash
git clone https://github.com/abrekhov/hyperstream.git
cd hyperstream
go build -o hstream .
```

### 3. Broadcast

```bash
hstream broadcast --source test
```

Open **http://localhost:8080** in any browser. Done.

---

## Sources

### Test pattern — no hardware needed

```bash
hstream broadcast --source test
```

Uses ffmpeg's built-in video test pattern and a 440 Hz sine tone.
Perfect for verifying the setup before using a real source.

### Screen capture

```bash
hstream broadcast --source screen
```

Captures your entire display.

> **Linux:** requires X11 (`DISPLAY` must be set). For Wayland, use a screen recorder that outputs to a file, then stream with `--source file`.  
> **macOS:** grant Screen Recording permission in System Settings → Privacy & Security.  
> **Windows:** uses GDI grab (works on all versions).

### Webcam / camera

```bash
hstream broadcast --source camera
```

> **Linux:** uses the first V4L2 device (`/dev/video0`). Change it via `--source file` with a symlink, or patch `pkg/media/ffmpeg.go`.  
> **macOS:** uses AVFoundation device index `0`.  
> **Windows:** uses the default DirectShow video device.

### Video file

```bash
hstream broadcast --source file --file /path/to/video.mp4
```

Streams any file ffmpeg can read: `.mp4`, `.mkv`, `.mov`, `.avi`, `.webm`, etc.  
Playback is real-time (`-re` flag) — the file loops at the end.

---

## CLI reference

```
hstream broadcast [flags]

Flags:
  -p, --port       int     HTTP port for viewers            (default: 8080)
  -s, --source     string  screen | camera | file | test    (default: screen)
  -f, --file       string  path to video file (source=file only)
  -W, --width      int     capture width in pixels           (default: 1280)
  -H, --height     int     capture height in pixels          (default: 720)
  -r, --framerate  int     capture frame rate                (default: 30)
  -h, --help               show help
```

**Examples:**

```bash
# Stream screen at 1080p 60fps on port 9090
hstream broadcast --source screen --width 1920 --height 1080 --framerate 60 --port 9090

# Stream webcam at 720p 30fps
hstream broadcast --source camera --width 1280 --height 720

# Stream a movie file
hstream broadcast --source file --file ~/Videos/demo.mp4

# Test pattern on default port
hstream broadcast --source test
```

---

## Multiple viewers

All viewers share the same encoded stream — the server encodes **once** and forwards RTP packets to every connected peer. There is no upper limit on the number of simultaneous viewers.

```
hstream broadcast --source screen --port 8080

# Anyone on your network opens:
http://<your-ip>:8080
```

---

## Accessing from another machine

By default hstream binds to all interfaces. Share your local IP:

```bash
# Find your IP
ip route get 1 | awk '{print $7; exit}'   # Linux
ipconfig getifaddr en0                     # macOS
```

Then viewers open `http://192.168.x.x:8080` in their browser.

> For access over the internet without port forwarding, use a reverse tunnel like  
> `ssh -R 8080:localhost:8080 user@yourserver.com` and share `http://yourserver.com:8080`.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  hstream process                                            │
│                                                             │
│  ┌─────────┐   RTP UDP :5004 (VP8)    ┌──────────────────┐ │
│  │         │ ────────────────────────► │                  │ │
│  │ ffmpeg  │   RTP UDP :5006 (Opus)   │  RTP listeners   │ │
│  │         │ ────────────────────────► │  (pkg/rtc)       │ │
│  └─────────┘                          │                  │ │
│       ▲                               │  TrackLocalRTP   │ │
│  Screen / Cam /                       │  video + audio   │ │
│  File / Test                          └────────┬─────────┘ │
│                                                │            │
│                                    track.Write(rtpPacket)  │
│                                                │            │
│                                       ┌────────▼─────────┐ │
│                                       │  PeerConnection  │ │
│                                       │  per viewer      │ │
│                                       └────────┬─────────┘ │
│                                                │            │
│  ┌──────────────────────────────────────────┐  │            │
│  │  HTTP server :8080                       │  │            │
│  │  GET  /       → embedded index.html      │  │            │
│  │  POST /offer  → SDP offer → SDP answer   │◄─┘            │
│  └──────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
                         │ WebRTC (DTLS + SRTP)
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
    ┌─────────┐    ┌─────────┐    ┌─────────┐
    │Browser 1│    │Browser 2│    │Browser N│
    └─────────┘    └─────────┘    └─────────┘
```

**Connection flow:**

1. `hstream broadcast` starts ffmpeg which encodes VP8 video and Opus audio
2. ffmpeg sends RTP packets to `127.0.0.1:5004` (video) and `:5006` (audio)
3. Go listeners read packets and write them into shared `TrackLocalStaticRTP` tracks
4. A viewer opens the URL; the embedded HTML creates an `RTCPeerConnection` with `recvonly` transceivers
5. Browser POSTs an SDP offer to `/offer`; hstream creates a peer connection, adds the shared tracks, returns an SDP answer
6. DTLS handshake completes; hstream forwards incoming RTP to the peer's SRTP sender

---

## Codec details

| Track | Codec | Payload Type | Clock Rate |
|-------|-------|-------------|------------|
| Video | VP8   | 96          | 90 000 Hz  |
| Audio | Opus  | 111         | 48 000 Hz  |

Payload types are hardcoded to match ffmpeg's defaults — no RTP rewriting needed.

---

## Build for all platforms

Requires [GoReleaser](https://goreleaser.com/install/):

```bash
goreleaser build --snapshot --clean
```

Output in `dist/`:

```
dist/
├── hstream_linux_amd64
├── hstream_linux_arm64
├── hstream_darwin_amd64
├── hstream_darwin_arm64
├── hstream_windows_amd64.exe
└── hstream_windows_arm64.exe
```

The binary is **CGO-free** — no C dependencies, statically linked, runs anywhere.

---

## Troubleshooting

**Black screen / no video in browser**

- Confirm ffmpeg is in `PATH`: `ffmpeg -version`
- Try `--source test` first to isolate hardware issues
- Run with stderr enabled: comment out `// cmd.Stderr = os.Stderr` in `pkg/media/ffmpeg.go` line 37

**Connection stuck at "Connecting..."**

- Check firewall: WebRTC uses UDP, make sure UDP traffic isn't blocked
- STUN (Google `stun.l.google.com:19302`) must be reachable from both sides
- For LAN-only use: both machines need to be on the same network

**Audio missing**

- Linux: ensure PulseAudio/PipeWire is running (`pulseaudio --check`)
- macOS: grant Microphone access to Terminal in System Settings
- Windows: check that the default audio device is active

**ffmpeg: "screen capture" fails on Linux**

```bash
# Check your DISPLAY variable
echo $DISPLAY     # should print :0 or :1
export DISPLAY=:0 # set it if empty
```

**macOS: permission denied for screen capture**

Go to **System Settings → Privacy & Security → Screen & System Audio Recording** and enable your terminal app.

---

## Project structure

```
hyperstream/
├── main.go                   # Entry point
├── cmd/
│   ├── root.go               # CLI root (hstream)
│   └── broadcast.go          # broadcast subcommand
├── pkg/
│   ├── rtc/
│   │   └── broadcaster.go    # WebRTC peer management + RTP forwarding
│   ├── signal/
│   │   └── server.go         # HTTP server: serves viewer + handles SDP
│   └── media/
│       └── ffmpeg.go         # ffmpeg process: capture → RTP
├── web/
│   ├── embed.go              # //go:embed index.html
│   └── index.html            # Browser viewer (embedded in binary)
├── go.mod
├── .goreleaser.yaml          # Cross-platform release builds
└── .github/workflows/
    ├── release.yaml          # Publish binaries on git tag v*.*.*
    └── test.yaml             # Build + vet on every push
```

---

## Releasing

Create a version tag to trigger a GitHub Actions release:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

GitHub Actions builds binaries for all 6 platform/arch combinations and publishes them to GitHub Releases automatically.

---

## License

Copyright © 2024 Anton Brekhov

Licensed under the [Apache License, Version 2.0](LICENSE).
