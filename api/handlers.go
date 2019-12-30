package api

import (
	"io/ioutil"
	"mime"
	"path/filepath"

	"github.com/hprose/hprose-golang/rpc"
	"github.com/valyala/fasthttp"

	"video-chat/api/rpc/websocket"
)

func debugAsset(path string) ([]byte, string, string, error) {
	if path == "/" {
		path = "/index.html"
	}
	file := "web" + path
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, "", "", ErrFileNotFound
	}
	return data, "", mime.TypeByExtension(filepath.Ext(file)), nil
}

func serveStaticAssets(ctx *fasthttp.RequestCtx) {
	if !ctx.IsGet() {
		ctx.Error(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed), fasthttp.StatusMethodNotAllowed)
		return
	}
	content, _, contentType, err := Asset(string(ctx.Path()))
	// content, _, contentType, err := debugAsset(string(ctx.Path()))
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusNotFound)
	} else {
		ctx.Response.Header.Set("Content-Encoding", "gzip")
		ctx.Response.Header.SetContentType(contentType)
		//ctx.SetContentType(contentType)
		ctx.Response.SetBody(content)
		//ctx.Response.Header.Reset()
		//ctx.Response.Header.Add("Content-Type", contentType)
	}
}

func NewRpcService() *websocket.Service {
	service := websocket.NewService(serveStaticAssets, onClientDisconnect)
	service.AddAllMethods(API(), rpc.Options{Simple: true, NameSpace: "Sessions"})
	return service
}
