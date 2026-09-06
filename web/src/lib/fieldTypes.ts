import type { FieldType } from '@/types/metadata'

export interface FieldTypeOption {
  value: FieldType
  label: string
  group: string
}

export const FIELD_TYPE_OPTIONS: FieldTypeOption[] = [
  { value: 'string', label: '单行文本', group: '基础' },
  { value: 'text', label: '多行文本', group: '基础' },
  { value: 'phone', label: '手机号', group: '基础' },
  { value: 'email', label: '邮箱', group: '基础' },
  { value: 'url', label: 'URL', group: '基础' },
  { value: 'number', label: '数字', group: '基础' },
  { value: 'integer', label: '整数', group: '基础' },
  { value: 'percent', label: '百分比', group: '基础' },
  { value: 'color', label: '颜色', group: '基础' },
  { value: 'icon', label: '图标', group: '基础' },
  { value: 'password', label: '密码', group: '基础' },
  { value: 'boolean', label: '勾选', group: '选择' },
  { value: 'select', label: '下拉单选', group: '选择' },
  { value: 'radio', label: '单选框组', group: '选择' },
  { value: 'multiSelect', label: '下拉多选', group: '选择' },
  { value: 'checkboxGroup', label: '复选框组', group: '选择' },
  { value: 'chinaRegion', label: '中国行政区', group: '选择' },
  { value: 'markdown', label: 'Markdown', group: '媒体' },
  { value: 'richText', label: '富文本', group: '媒体' },
  { value: 'markdownVditor', label: 'Markdown(Vditor)', group: '媒体' },
  { value: 'attachmentRelation', label: '附件(关联)', group: '媒体' },
  { value: 'attachmentUrl', label: '附件(URL)', group: '媒体' },
  { value: 'dateTime', label: '日期时间', group: '日期时间' },
  { value: 'dateTimeTz', label: '日期时间(时区)', group: '日期时间' },
  { value: 'date', label: '日期', group: '日期时间' },
  { value: 'time', label: '时间', group: '日期时间' },
  { value: 'unixTimestamp', label: 'Unix时间戳', group: '日期时间' },
  { value: 'belongsTo', label: '多对一', group: '关系' },
  { value: 'hasOne', label: '一对一', group: '关系' },
  { value: 'hasMany', label: '一对多', group: '关系' },
  { value: 'belongsToMany', label: '多对多', group: '关系' },
  { value: 'belongsToManyArray', label: '多对多(数组)', group: '关系' },
  { value: 'point', label: '点', group: '几何' },
  { value: 'lineString', label: '线', group: '几何' },
  { value: 'circle', label: '圆', group: '几何' },
  { value: 'polygon', label: '多边形', group: '几何' },
  { value: 'uuid', label: 'UUID', group: '高级' },
  { value: 'nanoId', label: 'NanoID', group: '高级' },
  { value: 'sort', label: '排序', group: '高级' },
  { value: 'formula', label: '公式', group: '高级' },
  { value: 'sequence', label: '自动序列', group: '高级' },
  { value: 'json', label: 'JSON', group: '高级' },
  { value: 'tableSelector', label: '表选择器', group: '高级' },
  { value: 'encrypted', label: '加密', group: '高级' },
  { value: 'createdAt', label: '创建时间', group: '系统' },
  { value: 'updatedAt', label: '更新时间', group: '系统' },
  { value: 'createdBy', label: '创建人', group: '系统' },
  { value: 'updatedBy', label: '更新人', group: '系统' },
  { value: 'tableOid', label: '表OID', group: '系统' },
]

export const COLLECTION_TYPES = [
  {
    value: 'general' as const,
    label: '普通表',
    desc: '客户、订单、合同等常规业务数据',
  },
  {
    value: 'tree' as const,
    label: '树表',
    desc: '组织架构、分类、目录等层级数据',
  },
  {
    value: 'calendar' as const,
    label: '日历表',
    desc: '预约、排期、日程等时间范围数据',
  },
  {
    value: 'comment' as const,
    label: '评论表',
    desc: '挂到业务记录的讨论与反馈',
  },
  {
    value: 'file' as const,
    label: '文件表',
    desc: '文件元信息（不含对象存储引擎）',
  },
]

export const PRESET_FIELDS = [
  { value: 'id', label: '主键 ID' },
  { value: 'createdAt', label: '创建时间' },
  { value: 'updatedAt', label: '更新时间' },
  { value: 'createdBy', label: '创建人' },
  { value: 'updatedBy', label: '更新人' },
]

export const HOOK_POINTS = [
  'beforeValidate',
  'afterValidate',
  'beforeCreate',
  'afterCreate',
  'beforeUpdate',
  'afterUpdate',
  'beforeDelete',
  'afterDelete',
  'afterCommit',
  'beforeFind',
  'afterFind',
  'customAPI',
]

export const SYSTEM_FIELD_TYPES = new Set([
  'createdAt',
  'updatedAt',
  'createdBy',
  'updatedBy',
  'tableOid',
  'uuid',
  'nanoId',
  'sequence',
  'formula',
])

export const READONLY_ON_FORM = new Set([
  'createdAt',
  'updatedAt',
  'createdBy',
  'updatedBy',
  'tableOid',
  'uuid',
  'nanoId',
  'sequence',
  'formula',
])

export function fieldTypeLabel(type: string): string {
  return FIELD_TYPE_OPTIONS.find((x) => x.value === type)?.label || type
}

export function isSearchableType(type: string): boolean {
  return ['string', 'text', 'email', 'phone', 'url'].includes(type)
}

export function isRelationType(type: string): boolean {
  return ['belongsTo', 'hasOne', 'hasMany', 'belongsToMany', 'belongsToManyArray', 'attachmentRelation'].includes(
    type,
  )
}

export function suggestNameFromDisplay(displayName: string): string {
  const ascii = displayName
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_\s-]/g, '')
    .replace(/[\s-]+/g, '_')
    .replace(/^_+|_+$/g, '')
  if (ascii && /^[a-z]/.test(ascii)) return ascii
  return ''
}
