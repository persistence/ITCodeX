/** 与 server/api/metadata/v1 及 consts.go 对齐 */

export type CollectionType = 'general' | 'tree' | 'calendar' | 'comment' | 'file'

export type FieldType =
  | 'string'
  | 'text'
  | 'phone'
  | 'email'
  | 'url'
  | 'number'
  | 'integer'
  | 'percent'
  | 'color'
  | 'icon'
  | 'password'
  | 'boolean'
  | 'select'
  | 'radio'
  | 'multiSelect'
  | 'checkboxGroup'
  | 'chinaRegion'
  | 'markdown'
  | 'richText'
  | 'markdownVditor'
  | 'attachmentRelation'
  | 'attachmentUrl'
  | 'dateTime'
  | 'dateTimeTz'
  | 'time'
  | 'date'
  | 'unixTimestamp'
  | 'belongsTo'
  | 'hasMany'
  | 'hasOne'
  | 'belongsToMany'
  | 'belongsToManyArray'
  | 'point'
  | 'lineString'
  | 'circle'
  | 'polygon'
  | 'uuid'
  | 'nanoId'
  | 'sort'
  | 'formula'
  | 'sequence'
  | 'json'
  | 'tableSelector'
  | 'encrypted'
  | 'createdAt'
  | 'updatedAt'
  | 'createdBy'
  | 'updatedBy'
  | 'tableOid'

export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageResult<T> {
  list: T[]
  total: number
  page?: number
  pageSize?: number
  totalPages?: number
}

export interface ValidationErrorData {
  fieldErrors?: Record<string, string[]>
  tableErrors?: string[]
}

export interface FieldValidationConfig {
  required?: boolean
  nullable?: boolean
  unique?: boolean
  minLength?: number
  maxLength?: number
  length?: number
  pattern?: string
  min?: number
  max?: number
  exclusiveMin?: number
  exclusiveMax?: number
  multipleOf?: number
  integer?: boolean
  format?: string
  rules?: CELRule[]
}

export interface CELRule {
  name: string
  expression: string
  errorMessage: string
}

export interface CollectionOptions {
  simplePagination?: boolean
  treeParentKey?: string
  calendarStartField?: string
  calendarEndField?: string
  commentForeignKey?: string
  extra?: Record<string, unknown>
}

export interface FieldItem {
  name: string
  displayName: string
  type: FieldType | string
  required?: boolean
  unique?: boolean
  indexed?: boolean
  isSystem?: boolean
  options?: Record<string, unknown>
  validation?: FieldValidationConfig | unknown
  dataType?: string
  description?: string
  sort?: number
  defaultValue?: unknown
  target?: string
  foreignKey?: string
  sourceKey?: string
  through?: string
  otherKey?: string
  targetKey?: string
}

export interface CollectionItem {
  name: string
  displayName: string
  tableName?: string
  type: CollectionType | string
  description?: string
  categories?: string[]
  options?: CollectionOptions | Record<string, unknown>
  fieldCount?: number
  fields?: FieldItem[]
  indexes?: IndexItem[]
  filterTargetKey?: string
  autoGenId?: boolean
}

export interface IndexItem {
  id?: number
  name?: string
  fields: string[]
  unique?: boolean
  options?: Record<string, unknown>
}

export interface CreateFieldInput {
  name: string
  displayName: string
  type: FieldType | string
  dataType?: string
  interfaceType?: string
  isSystem?: boolean
  isRequired?: boolean
  isUnique?: boolean
  isIndexed?: boolean
  defaultValue?: unknown
  description?: string
  sort?: number
  options?: Record<string, unknown>
  validation?: FieldValidationConfig
  validationRules?: CELRule[]
  target?: string
  foreignKey?: string
  sourceKey?: string
  through?: string
  otherKey?: string
  targetKey?: string
  autoGenerate?: boolean
  length?: number
  scopeKey?: string
  expression?: string
  pattern?: string
  startsAt?: number
  incrementBy?: number
  algorithm?: string
  targetCollection?: string
}

export interface CreateCollectionInput {
  name: string
  displayName: string
  type?: CollectionType | string
  description?: string
  categories?: string[]
  options?: CollectionOptions
  autoGenId?: boolean
  filterTargetKey?: string
  presetFields?: string[]
  fields?: CreateFieldInput[]
  indexes?: IndexItem[]
}

export interface UpdateCollectionInput {
  displayName?: string
  description?: string
  categories?: string[]
  options?: CollectionOptions
}

export interface UpdateFieldInput {
  displayName?: string
  isRequired?: boolean
  isUnique?: boolean
  isIndexed?: boolean
  options?: Record<string, unknown>
  validation?: FieldValidationConfig
  sort?: number
}

export interface YaegiScript {
  id?: number
  collectionName?: string
  name: string
  hookPoint: string
  content: string
  apiPath?: string
  httpMethod?: string
  enabled?: boolean
  priority?: number
  options?: string
  createdAt?: string
  updatedAt?: string
}

export type RecordData = Record<string, unknown>

export interface ListRecordsParams {
  filter?: Record<string, unknown> | string
  sort?: string
  fields?: string
  except?: string
  appends?: string
  page?: number
  pageSize?: number
  limit?: number
  offset?: number
}
