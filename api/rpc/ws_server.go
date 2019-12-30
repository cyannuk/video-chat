package rpc

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"

	"video-chat/api/rpc/websocket"
)

type Server struct {
	*fasthttp.Server
	service *websocket.Service
}

func onShutdown(f func()) {
	once := &sync.Once{}
	signalsChannel := make(chan os.Signal, 3)
	signal.Notify(signalsChannel, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-signalsChannel
		once.Do(f)
	}()
}

func NewServer(service *websocket.Service) Server {
	s := Server{&fasthttp.Server{Handler: service.ServeWS, NoDefaultServerHeader: true, NoDefaultContentType: true}, service}
	onShutdown(func() {
		err := s.Shutdown()
		if err != nil {
			log.Error().Err(err).Msg("New Ws Server")
		}
	})
	return s
}
