package rtc

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/rs/zerolog/log"
)

type Session struct {
	Id            string
	peer          Peer
	connectedPeer Peer
	stop          int32
}

func NewSession(id string, desc *webrtc.SessionDescription) (session *Session, answer webrtc.SessionDescription, err error) {
	peer, err := NewPeer()
	if err != nil {
		log.Error().Err(err).Msg("Session New Peer")
		return
	}
	// Allow to send/receive audio track
	err = peer.AddAudioTrack()
	if err != nil {
		log.Error().Err(err).Msg("Session Add Audio Track")
		return
	}
	// Allow to send/receive video track
	err = peer.AddVideoTrack()
	if err != nil {
		log.Error().Err(err).Msg("Session Add Video Track")
		return
	}
	audioTrack, err := newAudioTrack()
	if err != nil {
		log.Error().Err(err).Msg("Session New Audio Track")
		return
	}
	videoTrack, err := newVideoTrack()
	if err != nil {
		log.Error().Err(err).Msg("Session New Video Track")
		return
	}
	session = &Session{Id: id, peer: peer, connectedPeer: Peer{nil, audioTrack, videoTrack}, stop: 0}
	// Set the remote SessionDescription
	err = peer.SetRemoteDescription(*desc)
	if err != nil {
		log.Error().Err(err).Msg("Session Set Remote Description")
		return
	}
	// Create answer
	answer, err = peer.CreateAnswer(nil)
	if err != nil {
		log.Error().Err(err).Msg("Session Create Answer")
		return
	}
	// Sets the LocalDescription, and starts our UDP listeners
	err = peer.SetLocalDescription(answer)
	if err != nil {
		log.Error().Err(err).Msg("Session Set Local Description")
		return
	}
	session.peer.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		switch remoteTrack.Kind() {
		case webrtc.RTPCodecTypeVideo:
			go session.initiatePLI(remoteTrack.SSRC(), session.peer)
			go session.transmit(session.connectedPeer.videoTrack, remoteTrack)
		case webrtc.RTPCodecTypeAudio:
			go session.transmit(session.connectedPeer.audioTrack, remoteTrack)
		}
	})
	return
}

func (s *Session) Connect(desc *webrtc.SessionDescription) (answer webrtc.SessionDescription, err error) {
	err = s.connectedPeer.InitPeer()
	if err != nil {
		log.Error().Err(err).Msg("Session Init Peer")
		return
	}
	err = s.connectedPeer.SetRemoteDescription(*desc)
	if err != nil {
		log.Error().Err(err).Msg("Session Set Remote Description")
		return
	}
	answer, err = s.connectedPeer.CreateAnswer(nil)
	if err != nil {
		log.Error().Err(err).Msg("Session Create Answer")
		return
	}
	err = s.connectedPeer.SetLocalDescription(answer)
	if err != nil {
		log.Error().Err(err).Msg("Session Set Local Description")
		return
	}
	s.connectedPeer.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		switch remoteTrack.Kind() {
		case webrtc.RTPCodecTypeVideo:
			go s.initiatePLI(remoteTrack.SSRC(), s.connectedPeer)
			go s.transmit(s.peer.videoTrack, remoteTrack)
		case webrtc.RTPCodecTypeAudio:
			go s.transmit(s.peer.audioTrack, remoteTrack)
		}
	})
	return
}

func (s *Session) IsConnected() bool {
	return s.connectedPeer.PeerConnection != nil
}

func (s *Session) Leave() {
	err := s.connectedPeer.Close()
	if err != nil {
		log.Debug().Err(err).Msg("Session Leave; Connected Peer Close")
	}
}

func (s *Session) Close() {
	atomic.StoreInt32(&s.stop, 1)
	err := s.connectedPeer.Close()
	if err != nil {
		log.Debug().Err(err).Msg("Session Close; Connected Peer Close")
	}
	err = s.peer.Close()
	if err != nil {
		log.Debug().Err(err).Msg("Session Close; Peer Close")
	}
}

func (s *Session) IsStopped() bool {
	return atomic.LoadInt32(&s.stop) != 0
}

func (s *Session) initiatePLI(ssrc uint32, peer Peer) {
	// Send a PLI on an interval so that the publisher is pushing a keyframe every 5 secs
	ticker := time.NewTicker(time.Second * 5)
	pliPkt := []rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}}
	for !s.IsStopped() {
		_ = <-ticker.C
		err := peer.WriteRTCP(pliPkt)
		if err != nil {
			log.Debug().Err(err).Msg("Session initiate PLI")
		}
	}
	ticker.Stop()
}

func ReadRTP(track *webrtc.Track, bytes []byte, packet *rtp.Packet) (err error) {
	n, err := track.Read(bytes)
	if err != nil {
		return
	}
	packet.ExtensionPayload = nil
	err = packet.Unmarshal(bytes[:n])
	return
}

func (s *Session) transmit(track *webrtc.Track, remoteTrack *webrtc.Track) {
	bytes := make([]byte, 1460)
	packet := rtp.Packet{}
	for !s.IsStopped() {
		err := ReadRTP(remoteTrack, bytes, &packet)
		if err != nil {
			log.Error().Err(err).Msg("Session Track Read")
			break
		}
		packet.SSRC = track.SSRC()
		err = track.WriteRTP(&packet)
		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		if err != nil && err != io.ErrClosedPipe {
			log.Error().Err(err).Msg("Session Track Write")
			break
		}
	}
}
