package api

import (
	"errors"
	"strings"
	"time"

	"github.com/pion/webrtc/v2"
	"github.com/rs/zerolog/log"

	"video-chat/api/cache"
	"video-chat/api/rtc"
)

type sessions struct {
	cache *cache.Cache
}

type Sessions interface {
	New(id string, sdpTypeStr string, sdp string) (rtc.SessionDesc, error)
	Join(id string, sdpTypeStr string, sdp string) (rtc.SessionDesc, error)
	Close(id string)
	Leave(id string)
}

func toSDPType(typeStr string) (SDPType webrtc.SDPType, err error) {
	switch strings.ToLower(typeStr) {
	case "offer":
		SDPType = webrtc.SDPTypeOffer
	case "pranswer":
		SDPType = webrtc.SDPTypePranswer
	case "answer":
		SDPType = webrtc.SDPTypeAnswer
	case "rollback":
		SDPType = webrtc.SDPTypeRollback
	default:
		err = errors.New("Unknown SDP type: " + typeStr)
	}
	return
}

func (sessions *sessions) New(id string, sdpTypeStr string, sdp string) (answer rtc.SessionDesc, err error) {
	sdpType, err := toSDPType(sdpTypeStr)
	if err != nil {
		return
	}
	session, result, err := rtc.NewSession(id, &webrtc.SessionDescription{Type: sdpType, SDP: sdp})
	if err != nil {
		return
	}
	err = sessions.cache.Add(id, session, cache.NoExpiration)
	if err != nil {
		return
	}
	answer.Type = result.Type.String()
	answer.Sdp = result.SDP
	return
}

func (sessions *sessions) Join(id string, sdpTypeStr string, sdp string) (answer rtc.SessionDesc, err error) {
	sdpType, err := toSDPType(sdpTypeStr)
	if err != nil {
		return
	}
	session, err := sessions.cache.Get(id)
	if err != nil {
		return
	}
	if session.IsConnected() {
		err = errors.New("the callee is already connected")
		return
	}
	result, err := session.Connect(&webrtc.SessionDescription{Type: sdpType, SDP: sdp})
	if err != nil {
		return
	}
	answer.Type = result.Type.String()
	answer.Sdp = result.SDP
	return
}

func (sessions *sessions) Close(id string) {
	session, err := sessions.cache.Get(id)
	if err != nil {
		log.Debug().Err(err).Msg("Session Close; Get")
		return
	}
	session.Close()
	err = sessions.cache.Delete(id)
	if err != nil {
		log.Error().Err(err).Msg("Session Close; Delete")
	}
}

func (sessions *sessions) Leave(id string) {
	session, err := sessions.cache.Get(id)
	if err != nil {
		log.Debug().Err(err).Msg("Session Leave; Get")
		return
	}
	session.Leave()
}

func onClientDisconnect(clientID string) {
	session, err := _sessions.cache.Get(clientID)
	if err != nil {
		log.Debug().Err(err).Msg("Client Disconnect; Session Get")
		return
	}
	session.Close()
	err = _sessions.cache.Delete(clientID)
	if err != nil {
		log.Error().Err(err).Msg("Client Disconnect; Session Delete")
	}
}

var (
	_sessions = sessions{cache.New(100_000, 10, time.Hour, time.Hour, nil)}
)

func API() Sessions {
	return &_sessions
}
