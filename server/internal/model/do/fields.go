// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Fields is the golang structure of table c_fields for DAO operations like Where/Data.
type Fields struct {
	g.Meta         `orm:"table:c_fields, do:true"`
	Id             any         //
	CollectionName any         //
	Name           any         //
	Type           any         //
	DisplayName    any         //
	IsRequired     any         //
	IsUnique       any         //
	IsIndexed      any         //
	Validation     any         //
	Options        any         //
	Sort           any         //
	CreatedAt      *gtime.Time //
}
