package websocket

import (
	"bytes"
	"reflect"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/hprose/hprose-golang/rpc"
	"github.com/hprose/hprose-golang/util"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

var contextType = reflect.TypeOf((*Context)(nil))
var strWebsocket = []byte("websocket")

type ClientDisconnectHandler = func(clientID string)

type Context struct {
	rpc.BaseServiceContext
	WebSocket *websocket.Conn
	ClientID  string
	mutex     sync.Mutex
}

type Service struct {
	rpc.BaseService
	fallback           fasthttp.RequestHandler
	upgrader           websocket.FastHTTPUpgrader
	onClientDisconnect ClientDisconnectHandler
}

func websocketFixArguments(args []reflect.Value, context rpc.ServiceContext) {
	i := len(args) - 1
	switch args[i].Type() {
	case contextType:
		if c, ok := context.(*Context); ok {
			args[i] = reflect.ValueOf(c)
		}
	default:
		rpc.DefaultFixArguments(args, context)
	}
}

func checkOrigin(_ *fasthttp.RequestCtx) bool {
	return true
}

func clientID(context *Context ) string {
	return context.ClientID
}

// NewService is the constructor of Service
func NewService(fallback fasthttp.RequestHandler, onClientDisconnect ClientDisconnectHandler) *Service {
	service := Service{fallback: fallback, onClientDisconnect: onClientDisconnect}
	service.InitBaseService()
	service.AddFunction("#", clientID, rpc.Options{Simple: true})
	service.upgrader.CheckOrigin = checkOrigin
	service.FixArguments = websocketFixArguments
	return &service
}

func hasUpgradeHeader(headers *fasthttp.RequestHeader) bool {
	header := headers.Peek("Upgrade")
	if header == nil {
		return false
	}
	return bytes.Contains(bytes.ToLower(header), strWebsocket)
}

func (service *Service) ServeWS(ctx *fasthttp.RequestCtx) {
	if (ctx.IsGet() && !hasUpgradeHeader(&ctx.Request.Header)) || ctx.IsPost() || ctx.IsDelete() || ctx.IsPut() {
		if service.fallback != nil {
			service.fallback(ctx)
		} else {
			ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound), fasthttp.StatusNotFound)
		}
	} else {
		service.serve(ctx)
	}
}

func (service *Service) serve(ctx *fasthttp.RequestCtx) {
	_ = service.upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		context := service.newContext(conn)
		for {
			msgType, data, err := context.read()
			if err != nil {
				log.Debug().Err(err).Msg("Websocket Service Serve")
				if _, ok := err.(*websocket.CloseError); ok {
					if service.onClientDisconnect != nil {
						go service.onClientDisconnect(context.ClientID)
					}
				}
				break
			}
			if msgType == websocket.BinaryMessage {
				go service.handle(data, context)
			}
		}
		context.close()
	})
}

func (service *Service) newContext(conn *websocket.Conn) *Context {
	context := Context{}
	context.InitServiceContext(service)
	context.WebSocket = conn
	context.ClientID = util.UUIDv4()
	return &context
}

func (context *Context) read() (int, []byte, error) {
	return context.WebSocket.ReadMessage()
}

func (context *Context) write(id []byte, data []byte) (err error) {
	context.mutex.Lock()
	defer context.mutex.Unlock()
	writer, err := context.WebSocket.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return
	}
	_, err = writer.Write(id)
	if err != nil {
		return
	}
	_, err = writer.Write(data)
	if err != nil {
		return
	}
	err = writer.Close()
	return
}

func (context *Context) close() {
	_ = context.WebSocket.Close()
}

func (service *Service) handle(data []byte, context *Context) {
	err := context.write(data[0:4], service.Handle(data[4:], context))
	if err != nil {
		_ = rpc.FireErrorEvent(service.Event, err, context)
	}
}
