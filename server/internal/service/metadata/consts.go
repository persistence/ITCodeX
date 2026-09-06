package metadata

type CollectionType string

const (
	CollectionTypeGeneral  CollectionType = "general"
	CollectionTypeTree     CollectionType = "tree"
	CollectionTypeCalendar CollectionType = "calendar"
	CollectionTypeComment  CollectionType = "comment"
	CollectionTypeFile     CollectionType = "file"
)

type HookPoint string

const (
	HookPointBeforeValidate HookPoint = "beforeValidate"
	HookPointAfterValidate  HookPoint = "afterValidate"
	HookPointBeforeCreate   HookPoint = "beforeCreate"
	HookPointAfterCreate    HookPoint = "afterCreate"
	HookPointBeforeUpdate   HookPoint = "beforeUpdate"
	HookPointAfterUpdate    HookPoint = "afterUpdate"
	HookPointBeforeDelete   HookPoint = "beforeDelete"
	HookPointAfterDelete    HookPoint = "afterDelete"
	HookPointAfterCommit    HookPoint = "afterCommit"
	HookPointBeforeFind     HookPoint = "beforeFind"
	HookPointAfterFind      HookPoint = "afterFind"
	HookPointCustomAPI      HookPoint = "customAPI"
)

type FieldType string

const (
	FieldTypeString             FieldType = "string"
	FieldTypeText               FieldType = "text"
	FieldTypePhone              FieldType = "phone"
	FieldTypeEmail              FieldType = "email"
	FieldTypeUrl                FieldType = "url"
	FieldTypeNumber             FieldType = "number"
	FieldTypeInteger            FieldType = "integer"
	FieldTypePercent            FieldType = "percent"
	FieldTypeColor              FieldType = "color"
	FieldTypeIcon               FieldType = "icon"
	FieldTypePassword           FieldType = "password"
	FieldTypeBoolean            FieldType = "boolean"
	FieldTypeSelect             FieldType = "select"
	FieldTypeRadio              FieldType = "radio"
	FieldTypeMultiSelect        FieldType = "multiSelect"
	FieldTypeCheckboxGroup      FieldType = "checkboxGroup"
	FieldTypeChinaRegion        FieldType = "chinaRegion"
	FieldTypeMarkdown           FieldType = "markdown"
	FieldTypeRichText           FieldType = "richText"
	FieldTypeMarkdownVditor     FieldType = "markdownVditor"
	FieldTypeAttachmentRelation FieldType = "attachmentRelation"
	FieldTypeAttachmentUrl      FieldType = "attachmentUrl"
	FieldTypeDateTime           FieldType = "dateTime"
	FieldTypeDateTimeTz         FieldType = "dateTimeTz"
	FieldTypeTime               FieldType = "time"
	FieldTypeDate               FieldType = "date"
	FieldTypeUnixTimestamp      FieldType = "unixTimestamp"
	FieldTypeBelongsTo          FieldType = "belongsTo"
	FieldTypeHasMany            FieldType = "hasMany"
	FieldTypeHasOne             FieldType = "hasOne"
	FieldTypeBelongsToMany      FieldType = "belongsToMany"
	FieldTypeBelongsToManyArray FieldType = "belongsToManyArray"
	FieldTypePoint              FieldType = "point"
	FieldTypeLineString         FieldType = "lineString"
	FieldTypeCircle             FieldType = "circle"
	FieldTypePolygon            FieldType = "polygon"
	FieldTypeUUID               FieldType = "uuid"
	FieldTypeNanoID             FieldType = "nanoId"
	FieldTypeSort               FieldType = "sort"
	FieldTypeFormula            FieldType = "formula"
	FieldTypeSequence           FieldType = "sequence"
	FieldTypeJSON               FieldType = "json"
	FieldTypeTableSelector      FieldType = "tableSelector"
	FieldTypeEncrypted          FieldType = "encrypted"
	FieldTypeCreatedAt          FieldType = "createdAt"
	FieldTypeUpdatedAt          FieldType = "updatedAt"
	FieldTypeCreatedBy          FieldType = "createdBy"
	FieldTypeUpdatedBy          FieldType = "updatedBy"
	FieldTypeTableOID           FieldType = "tableOid"
)

type DataType string

const (
	DataTypeVarchar   DataType = "varchar"
	DataTypeText      DataType = "text"
	DataTypeLongText  DataType = "longtext"
	DataTypeInteger   DataType = "integer"
	DataTypeBigInt    DataType = "bigInt"
	DataTypeFloat     DataType = "float"
	DataTypeDouble    DataType = "double"
	DataTypeDecimal   DataType = "decimal"
	DataTypeBoolean   DataType = "boolean"
	DataTypeDate      DataType = "date"
	DataTypeTime      DataType = "time"
	DataTypeDateTime  DataType = "datetime"
	DataTypeTimestamp DataType = "timestamp"
	DataTypeJSON      DataType = "json"
	DataTypeJSONB     DataType = "jsonb"
	DataTypeUUID      DataType = "uuid"
	DataTypeBlob      DataType = "blob"
)

const (
	DefaultPrimaryKey   = "id"
	DefaultPage         = 1
	DefaultPageSize     = 20
	MaxPageSize         = 1000
	ActionCreate        = "create"
	ActionUpdate        = "update"
	ErrorValidationCode = 422
	ErrorNotFoundCode   = 404
	ErrorForbiddenCode  = 403
	ErrorInternalCode   = 500
)
