package rtc

import (
	"math/rand"

	"github.com/dchest/uniuri"
	"github.com/pion/webrtc/v2"
	"github.com/satori/go.uuid"
)

type Peer struct {
	*webrtc.PeerConnection
	audioTrack *webrtc.Track
	videoTrack *webrtc.Track
}

func NewPeer() (peer Peer, err error) {
	api := API()
	peer.PeerConnection, err = api.NewPeerConnection(api.config)
	if err != nil {
		return
	}
	return
}

func (p *Peer) InitPeer() (err error) {
	api := API()
	p.PeerConnection, err = api.NewPeerConnection(api.config)
	if err != nil {
		return
	}
	_, err = p.PeerConnection.AddTrack(p.videoTrack)
	if err != nil {
		return
	}
	_, err = p.PeerConnection.AddTrack(p.audioTrack)
	return
}

func (p *Peer) Close() (err error) {
	if p.PeerConnection != nil {
		err = p.PeerConnection.Close()
	}
	return
}

func newAudioTrack() (*webrtc.Track, error) {
	codec := getAudioCodec()
	return webrtc.NewTrack(codec.PayloadType, rand.Uint32(), uniuri.New(), uuid.NewV4().String(), codec)
}

func newVideoTrack() (*webrtc.Track, error) {
	codec := getVideoCodec()
	return webrtc.NewTrack(codec.PayloadType, rand.Uint32(), uniuri.New(), uuid.NewV4().String(), codec)
}

func (p *Peer) addNewTrack(codec *webrtc.RTPCodec) (track *webrtc.Track, err error) {
	track, err = webrtc.NewTrack(codec.PayloadType, rand.Uint32(), uniuri.New(), uuid.NewV4().String(), codec)
	if err != nil {
		return
	}
	_, err = p.PeerConnection.AddTransceiverFromTrack(track)
	return
}

func (p *Peer) AddAudioTrack() error {
	track, err := p.addNewTrack(getAudioCodec())
	if err != nil {
		return err
	}
	p.audioTrack = track
	return nil
}

func (p *Peer) AddVideoTrack() (err error) {
	track, err := p.addNewTrack(getVideoCodec())
	if err != nil {
		return err
	}
	p.videoTrack = track
	return nil
}
