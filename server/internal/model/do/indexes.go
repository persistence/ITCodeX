// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Indexes is the golang structure of table c_indexes for DAO operations like Where/Data.
type Indexes struct {
	g.Meta         `orm:"table:c_indexes, do:true"`
	Id             any         //
	CollectionName any         //
	Name           any         //
	Fields         any         //
	Unique         any         //
	Options        any         //
	CreatedAt      *gtime.Time //
}
