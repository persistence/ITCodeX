// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// YaegiScriptsDao is the data access object for the table c_yaegi_scripts.
type YaegiScriptsDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  YaegiScriptsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// YaegiScriptsColumns defines and stores column names for the table c_yaegi_scripts.
type YaegiScriptsColumns struct {
	Id             string //
	CollectionName string //
	Name           string //
	HookPoint      string //
	Content        string //
	ApiPath        string //
	HttpMethod     string //
	Enabled        string //
	Priority       string //
	Options        string //
	CreatedAt      string //
	UpdatedAt      string //
}

// yaegiScriptsColumns holds the columns for the table c_yaegi_scripts.
var yaegiScriptsColumns = YaegiScriptsColumns{
	Id:             "id",
	CollectionName: "collection_name",
	Name:           "name",
	HookPoint:      "hook_point",
	Content:        "content",
	ApiPath:        "api_path",
	HttpMethod:     "http_method",
	Enabled:        "enabled",
	Priority:       "priority",
	Options:        "options",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
}

// NewYaegiScriptsDao creates and returns a new DAO object for table data access.
func NewYaegiScriptsDao(handlers ...gdb.ModelHandler) *YaegiScriptsDao {
	return &YaegiScriptsDao{
		group:    "default",
		table:    "c_yaegi_scripts",
		columns:  yaegiScriptsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *YaegiScriptsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *YaegiScriptsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *YaegiScriptsDao) Columns() YaegiScriptsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *YaegiScriptsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *YaegiScriptsDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *YaegiScriptsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
