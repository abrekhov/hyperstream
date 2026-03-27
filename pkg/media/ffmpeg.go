/*
Copyright © 2024 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package media

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
)

// Config holds media capture configuration.
type Config struct {
	Source    string
	File      string
	Width     int
	Height    int
	Framerate int
	VideoPort int
	AudioPort int
}

// Process represents a running ffmpeg capture process.
type Process struct {
	cmd *exec.Cmd
}

// Stop terminates the ffmpeg process.
func (p *Process) Stop() {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

// StartCapture launches ffmpeg with the given configuration.
// ffmpeg must be installed and available in PATH.
func StartCapture(cfg Config) (*Process, error) {
	args := buildArgs(cfg)
	cmd := exec.Command("ffmpeg", args...)
	// Uncomment the following line to see ffmpeg output for debugging:
	// cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg (is ffmpeg installed?): %w", err)
	}

	return &Process{cmd: cmd}, nil
}

func videoOutput(inputIdx, port int) []string {
	return []string{
		"-map", strconv.Itoa(inputIdx) + ":v",
		"-c:v", "libvpx",
		"-b:v", "2M",
		"-deadline", "realtime",
		"-cpu-used", "4",
		"-payload_type", "96",
		"-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d", port),
	}
}

func audioOutput(inputIdx, port int) []string {
	return []string{
		"-map", strconv.Itoa(inputIdx) + ":a",
		"-c:a", "libopus",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", "128k",
		"-payload_type", "111",
		"-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d", port),
	}
}

func buildArgs(cfg Config) []string {
	switch cfg.Source {
	case "file":
		return buildFileArgs(cfg)
	case "camera":
		return buildCameraArgs(cfg)
	case "test":
		return buildTestArgs(cfg)
	default:
		return buildScreenArgs(cfg)
	}
}

func buildTestArgs(cfg Config) []string {
	args := []string{
		"-f", "lavfi", "-i",
		fmt.Sprintf("testsrc=size=%dx%d:rate=%d", cfg.Width, cfg.Height, cfg.Framerate),
		"-f", "lavfi", "-i",
		"sine=frequency=440:sample_rate=48000",
	}
	return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(1, cfg.AudioPort)...)...)
}

func buildFileArgs(cfg Config) []string {
	args := []string{"-re", "-i", cfg.File}
	return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(0, cfg.AudioPort)...)...)
}

func buildScreenArgs(cfg Config) []string {
	fps := strconv.Itoa(cfg.Framerate)
	size := fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)

	switch runtime.GOOS {
	case "linux":
		args := []string{
			"-f", "x11grab", "-framerate", fps, "-video_size", size, "-i", ":0.0",
			"-f", "pulse", "-i", "default",
		}
		return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(1, cfg.AudioPort)...)...)
	case "darwin":
		// avfoundation: "screen_index:audio_index"
		args := []string{"-f", "avfoundation", "-framerate", fps, "-i", "1:0"}
		return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(0, cfg.AudioPort)...)...)
	case "windows":
		args := []string{
			"-f", "gdigrab", "-framerate", fps, "-i", "desktop",
			"-f", "dshow", "-i", "audio=default",
		}
		return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(1, cfg.AudioPort)...)...)
	default:
		return buildTestArgs(cfg)
	}
}

func buildCameraArgs(cfg Config) []string {
	fps := strconv.Itoa(cfg.Framerate)
	size := fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)

	switch runtime.GOOS {
	case "linux":
		args := []string{
			"-f", "v4l2", "-framerate", fps, "-video_size", size, "-i", "/dev/video0",
			"-f", "pulse", "-i", "default",
		}
		return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(1, cfg.AudioPort)...)...)
	case "darwin":
		args := []string{"-f", "avfoundation", "-framerate", fps, "-i", "0:0"}
		return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(0, cfg.AudioPort)...)...)
	case "windows":
		args := []string{"-f", "dshow", "-i", "video=:audio=default"}
		return append(args, append(videoOutput(0, cfg.VideoPort), audioOutput(0, cfg.AudioPort)...)...)
	default:
		return buildTestArgs(cfg)
	}
}
