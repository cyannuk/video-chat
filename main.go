package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"video-chat/api"
	"video-chat/api/rpc"
)

var bindAddr string

func init() {
	// init logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: true})
	flag.StringVar(&bindAddr, "bind", "0.0.0.0:8080", "binding address")
	flag.Parse()
}

func main() {
	log.Info().Msg("Starting..")

	s := rpc.NewServer(api.NewRpcService())
	if err := s.ListenAndServe(bindAddr); err != nil {
		log.Error().Err(err).Msg("RPC Server")
	}
}
