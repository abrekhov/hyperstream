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
package rtc

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
)

// Broadcaster manages WebRTC peer connections and forwards RTP streams to all viewers.
type Broadcaster struct {
	api        *webrtc.API
	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
	peers      []*webrtc.PeerConnection
	mu         sync.Mutex
}

// NewBroadcaster creates a Broadcaster with VP8 (PT 96) and Opus (PT 111) codecs.
// These payload types match ffmpeg's default RTP output, avoiding PT rewriting.
func NewBroadcaster() (*Broadcaster, error) {
	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8,
			ClockRate: 90000,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		return nil, fmt.Errorf("register VP8 codec: %w", err)
	}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, fmt.Errorf("register Opus codec: %w", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		"video", "hyperstream-video",
	)
	if err != nil {
		return nil, fmt.Errorf("create video track: %w", err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "hyperstream-audio",
	)
	if err != nil {
		return nil, fmt.Errorf("create audio track: %w", err)
	}

	return &Broadcaster{
		api:        api,
		videoTrack: videoTrack,
		audioTrack: audioTrack,
	}, nil
}

// AddViewer creates a new WebRTC peer connection for a viewer.
// It accepts an SDP offer from the browser and returns an SDP answer.
func (b *Broadcaster) AddViewer(offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := b.api.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}

	if _, err = pc.AddTrack(b.videoTrack); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add video track: %w", err)
	}

	if _, err = pc.AddTrack(b.audioTrack); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add audio track: %w", err)
	}

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("viewer ICE: %s (total peers: %d)", state, len(b.peers))
		switch state {
		case webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed,
			webrtc.ICEConnectionStateDisconnected:
			b.removePeer(pc)
			_ = pc.Close()
		}
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("set remote description: %w", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("create answer: %w", err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)

	if err = pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("set local description: %w", err)
	}

	<-gatherComplete

	b.mu.Lock()
	b.peers = append(b.peers, pc)
	count := len(b.peers)
	b.mu.Unlock()

	log.Printf("viewer connected (total: %d)", count)

	ld := pc.LocalDescription()
	return ld, nil
}

func (b *Broadcaster) removePeer(pc *webrtc.PeerConnection) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, p := range b.peers {
		if p == pc {
			b.peers = append(b.peers[:i], b.peers[i+1:]...)
			log.Printf("viewer disconnected (total: %d)", len(b.peers))
			return
		}
	}
}

// StartRTPListeners opens UDP ports for incoming RTP from ffmpeg.
func (b *Broadcaster) StartRTPListeners(videoPort, audioPort int) error {
	videoConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: videoPort,
	})
	if err != nil {
		return fmt.Errorf("video RTP listener (port %d): %w", videoPort, err)
	}

	audioConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: audioPort,
	})
	if err != nil {
		_ = videoConn.Close()
		return fmt.Errorf("audio RTP listener (port %d): %w", audioPort, err)
	}

	go b.forwardRTP(videoConn, b.videoTrack)
	go b.forwardRTP(audioConn, b.audioTrack)

	return nil
}

func (b *Broadcaster) forwardRTP(conn *net.UDPConn, track *webrtc.TrackLocalStaticRTP) {
	buf := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}
		if _, err = track.Write(buf[:n]); err != nil {
			continue
		}
	}
}
