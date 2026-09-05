package middleware

import (
	"net/http"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/net/ghttp"
)

// HandlerResponse writes {code,message,data}. Skips if the handler already wrote a body (dynamic CRUD).
func HandlerResponse(r *ghttp.Request) {
	r.Middleware.Next()
	if r.Response.BufferLength() > 0 {
		return
	}

	var (
		msg  = "success"
		code = 0
		data = r.GetHandlerResponse()
		err  = r.GetError()
	)
	if err != nil {
		code = 1
		msg = err.Error()
		data = nil
		if e := gerror.Code(err); e != nil && e != gcode.CodeNil {
			code = e.Code()
			if e.Message() != "" {
				msg = e.Message()
			}
			if e.Detail() != nil {
				data = e.Detail()
			}
		}
		status := http.StatusOK
		switch code {
		case 404:
			status = http.StatusNotFound
		case 409:
			status = http.StatusConflict
		case 403:
			status = http.StatusForbidden
		case 422:
			status = http.StatusUnprocessableEntity
		case 1:
			status = http.StatusInternalServerError
		}
		if status != http.StatusOK {
			r.Response.WriteHeader(status)
		}
	}
	r.Response.WriteJson(map[string]interface{}{
		"code":    code,
		"message": msg,
		"data":    data,
	})
}
