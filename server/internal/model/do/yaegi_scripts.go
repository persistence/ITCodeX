// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// YaegiScripts is the golang structure of table c_yaegi_scripts for DAO operations like Where/Data.
type YaegiScripts struct {
	g.Meta         `orm:"table:c_yaegi_scripts, do:true"`
	Id             any         //
	CollectionName any         //
	Name           any         //
	HookPoint      any         //
	Content        any         //
	ApiPath        any         //
	HttpMethod     any         //
	Enabled        any         //
	Priority       any         //
	Options        any         //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
}
