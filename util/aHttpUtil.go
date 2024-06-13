package util

import (
	"fmt"
	"github.com/valyala/fasthttp"
)

const (
	jsonContentType = "application/json"
)

func SendResponse(ctx *fasthttp.RequestCtx, status int, response string) {
	ctx.SetStatusCode(status)
	if status == fasthttp.StatusOK {
		ctx.Response.Header.Set("Pragma", "no-cache")
		ctx.Response.Header.Set("Expires", "Thu, 19 Nov 1981 08:52:00 GMT")
		ctx.Response.Header.Set("cache-control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
		ctx.Response.Header.Set("vary", "Accept-Language")
		ctx.Response.Header.Set("content-type", "application/json; charset=UTF-8")
		ctx.Response.Header.Set("access-control-allow-origin", "*")
		ctx.Response.Header.Set("x-frame-options", "SAMEORIGIN")
		ctx.Response.Header.Set("x-xss-protection", " 1; mode=block")
	}
	//For more info: http://craigwickesser.com/2015/01/golang-http-to-many-open-files/
	ctx.Response.Header.Set("Connection", "close")
	fmt.Fprint(ctx, response)
}





