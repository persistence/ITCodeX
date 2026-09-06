package metadata

import (
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	md "itcodex/server/internal/service/metadata"
)

func writeOK(r *ghttp.Request, data any) {
	r.Response.WriteJson(g.Map{"code": 0, "message": "success", "data": data})
}

func writeCreated(r *ghttp.Request, data any) {
	r.Response.WriteHeader(http.StatusCreated)
	r.Response.WriteJson(g.Map{"code": 0, "message": "success", "data": data})
}

func writeFail(r *ghttp.Request, status, code int, message string, data any) {
	r.Response.WriteHeader(status)
	r.Response.WriteJson(g.Map{"code": code, "message": message, "data": data})
}

func writeLogicError(r *ghttp.Request, err error) {
	switch e := err.(type) {
	case *md.ValidationError:
		writeFail(r, http.StatusUnprocessableEntity, 422, "数据校验失败", g.Map{
			"fieldErrors": e.FieldErrors,
			"tableErrors": e.TableErrors,
		})
	case *md.NotFoundError:
		writeFail(r, http.StatusNotFound, 404, e.Error(), nil)
	case *md.AlreadyExistsError:
		writeFail(r, http.StatusConflict, 409, e.Error(), nil)
	case *md.ForbiddenError:
		writeFail(r, http.StatusForbidden, 403, e.Error(), nil)
	default:
		writeFail(r, http.StatusInternalServerError, 1, err.Error(), nil)
	}
}
