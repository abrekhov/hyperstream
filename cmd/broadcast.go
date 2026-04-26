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
package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/abrekhov/hyperstream/pkg/media"
	"github.com/abrekhov/hyperstream/pkg/rtc"
	"github.com/abrekhov/hyperstream/pkg/signal"
	"github.com/spf13/cobra"
)

var (
	port      int
	source    string
	inputFile string
	width     int
	height    int
	framerate int
)

var broadcastCmd = &cobra.Command{
	Use:   "broadcast",
	Short: "Start broadcasting a video/audio stream",
	Long: `Start a WebRTC broadcast server. Viewers connect via any modern browser.

Sources:
  screen  - capture the entire screen (default)
  camera  - capture the webcam
  file    - stream from a video file (requires --file)
  test    - ffmpeg test pattern, no capture device needed`,
	RunE: Broadcast,
}

func init() {
	broadcastCmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP server port for viewer connections")
	broadcastCmd.Flags().StringVarP(&source, "source", "s", "screen", "Media source: screen, camera, file, test")
	broadcastCmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input video file (required when --source=file)")
	broadcastCmd.Flags().IntVarP(&width, "width", "W", 1280, "Capture width in pixels")
	broadcastCmd.Flags().IntVarP(&height, "height", "H", 720, "Capture height in pixels")
	broadcastCmd.Flags().IntVarP(&framerate, "framerate", "r", 30, "Capture framerate")
}

// Broadcast starts the HTTP server and media pipeline.
func Broadcast(cmd *cobra.Command, args []string) error {
	if source == "file" && inputFile == "" {
		return fmt.Errorf("--file is required when --source=file")
	}

	broadcaster, err := rtc.NewBroadcaster()
	if err != nil {
		return fmt.Errorf("failed to create broadcaster: %w", err)
	}

	if err := broadcaster.StartRTPListeners(5004, 5006); err != nil {
		return fmt.Errorf("failed to start RTP listeners: %w", err)
	}

	cfg := media.Config{
		Source:    source,
		File:      inputFile,
		Width:     width,
		Height:    height,
		Framerate: framerate,
		VideoPort: 5004,
		AudioPort: 5006,
	}

	proc, err := media.StartCapture(cfg)
	if err != nil {
		return fmt.Errorf("failed to start media capture: %w", err)
	}
	defer proc.Stop()

	hub := signal.NewHub(broadcaster)

	mux := http.NewServeMux()
	mux.HandleFunc("/", hub.ServeViewer)
	mux.HandleFunc("/broadcast", hub.ServeBroadcast)
	mux.HandleFunc("/call", hub.ServeCall)
	mux.HandleFunc("/offer", hub.HandleOffer)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("HyperStream broadcasting on http://localhost%s", addr)
	log.Printf("Open the URL in a browser to watch the stream")
	log.Printf("Source: %s | Resolution: %dx%d @ %dfps", source, width, height, framerate)

	return http.ListenAndServe(addr, mux)
}
