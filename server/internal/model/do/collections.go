// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Collections is the golang structure of table c_collections for DAO operations like Where/Data.
type Collections struct {
	g.Meta      `orm:"table:c_collections, do:true"`
	Id          any         //
	Name        any         //
	DisplayName any         //
	Type        any         //
	Options     any         //
	CreatedAt   *gtime.Time //
	UpdatedAt   *gtime.Time //
}
