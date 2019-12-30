package rtc

import (
	"sync"

	"github.com/pion/webrtc/v2"
)

type webrtcApi struct {
	*webrtc.API
	config webrtc.Configuration
}

var (
	audioCodec *webrtc.RTPCodec
	videoCodec *webrtc.RTPCodec
	mediaEngine webrtc.MediaEngine
	api *webrtcApi
	once sync.Once
)

func API() *webrtcApi {
	once.Do(func() {
		mediaEngine = webrtc.MediaEngine{}
		// Setup the codecs you want to use
		audioCodec = webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000)
		mediaEngine.RegisterCodec(audioCodec)
		videoCodec = webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000)
		mediaEngine.RegisterCodec(videoCodec)
		// Create the API object with the MediaEngine
		api = &webrtcApi{
			webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine)),
			webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{"stun:stun.l.google.com:19302"},
					},
				},
			},
		}
	})
	return api
}

func getAudioCodec() *webrtc.RTPCodec {
	return audioCodec
}

func getVideoCodec() *webrtc.RTPCodec {
	return videoCodec
}
