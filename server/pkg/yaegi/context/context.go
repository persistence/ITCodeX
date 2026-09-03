package yaegictx

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/traefik/yaegi/interp"
)

type YaegiRequest struct {
	Raw    *http.Request
	Body   []byte
	Path   string
	Method string
}

func (r *YaegiRequest) BindJSON(v interface{}) error {
	if r.Body == nil {
		return nil
	}
	return json.Unmarshal(r.Body, v)
}

func (r *YaegiRequest) GetHeader(key string) string {
	if r.Raw == nil {
		return ""
	}
	return r.Raw.Header.Get(key)
}

type YaegiResponseWriter interface {
	Header() http.Header
	Write(b []byte) (int, error)
	WriteHeader(statusCode int)
}

type YaegiResponse struct {
	Raw    http.ResponseWriter
	Status int
}

func (r *YaegiResponse) JSON(code int, v interface{}) {
	r.Raw.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Status = code
	r.Raw.WriteHeader(code)
	json.NewEncoder(r.Raw).Encode(v)
}

func (r *YaegiResponse) JSONSuccess(data interface{}) {
	r.JSON(200, map[string]interface{}{"code": 0, "data": data})
}

func (r *YaegiResponse) JSONError(code int, msg string) {
	r.JSON(code, map[string]interface{}{"code": code, "message": msg})
}

func (r *YaegiResponse) WriteString(s string) {
	io.WriteString(r.Raw, s)
}

type YaegiHTTPContext struct {
	Request  *YaegiRequest
	Response *YaegiResponse
	Params   map[string]string
	Query    url.Values
	Headers  http.Header
	Locals   map[string]interface{}
}

func NewYaegiHTTPContext(w http.ResponseWriter, req *http.Request, params map[string]string) *YaegiHTTPContext {
	body, _ := io.ReadAll(req.Body)
	if params == nil {
		params = make(map[string]string)
	}
	return &YaegiHTTPContext{
		Request: &YaegiRequest{
			Raw:    req,
			Body:   body,
			Path:   req.URL.Path,
			Method: req.Method,
		},
		Response: &YaegiResponse{
			Raw:    w,
			Status: http.StatusOK,
		},
		Params:  params,
		Query:   req.URL.Query(),
		Headers: req.Header,
		Locals:  make(map[string]interface{}),
	}
}

var Symbols = interp.Exports{
	"itcodex/context/context": map[string]reflect.Value{
		"NewYaegiHTTPContext": reflect.ValueOf(NewYaegiHTTPContext),
		"YaegiRequest":        reflect.ValueOf((*YaegiRequest)(nil)),
		"YaegiResponse":       reflect.ValueOf((*YaegiResponse)(nil)),
		"YaegiHTTPContext":    reflect.ValueOf((*YaegiHTTPContext)(nil)),
	},
}
