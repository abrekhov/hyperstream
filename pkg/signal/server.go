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
package signal

import (
	"encoding/json"
	"net/http"

	"github.com/abrekhov/hyperstream/pkg/rtc"
	"github.com/abrekhov/hyperstream/web"
	"github.com/pion/webrtc/v3"
)

// Hub handles HTTP signaling and serves the viewer page.
type Hub struct {
	broadcaster *rtc.Broadcaster
}

// NewHub creates a new signaling Hub.
func NewHub(b *rtc.Broadcaster) *Hub {
	return &Hub{broadcaster: b}
}

type offerRequest struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

type answerResponse struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

// HandleOffer processes a WebRTC SDP offer and returns an SDP answer.
func (h *Hub) HandleOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req offerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Type != "offer" {
		http.Error(w, "expected type=offer", http.StatusBadRequest)
		return
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  req.SDP,
	}

	answer, err := h.broadcaster.AddViewer(offer)
	if err != nil {
		http.Error(w, "failed to create answer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(answerResponse{
		Type: "answer",
		SDP:  answer.SDP,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ServeViewer serves the HTML viewer page.
func (h *Hub) ServeViewer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(web.ViewerHTML); err != nil {
		http.Error(w, "failed to serve viewer", http.StatusInternalServerError)
	}
}

// ServeBroadcast serves the HTML broadcaster page.
func (h *Hub) ServeBroadcast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(web.BroadcastHTML); err != nil {
		http.Error(w, "failed to serve broadcast page", http.StatusInternalServerError)
	}
}

// ServeCall serves the HTML call page.
func (h *Hub) ServeCall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(web.CallHTML); err != nil {
		http.Error(w, "failed to serve call page", http.StatusInternalServerError)
	}
}
