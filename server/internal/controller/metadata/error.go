package metadata

import (
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	md "itcodex/server/internal/service/metadata"
)

func wrapSvcErr(err error) error {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *md.ValidationError:
		return gerror.NewCode(gcode.New(422, "数据校验失败", g.Map{
			"fieldErrors": e.FieldErrors,
			"tableErrors": e.TableErrors,
		}), "数据校验失败")
	case *md.NotFoundError:
		return gerror.NewCode(gcode.New(404, e.Error(), nil), e.Error())
	case *md.AlreadyExistsError:
		return gerror.NewCode(gcode.New(409, e.Error(), nil), e.Error())
	case *md.ForbiddenError:
		return gerror.NewCode(gcode.New(403, e.Error(), nil), e.Error())
	default:
		return gerror.Wrap(err, err.Error())
	}
}
