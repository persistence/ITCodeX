import type { ReactNode } from 'react'
import {
  Checkbox,
  ColorPicker,
  DatePicker,
  Input,
  InputNumber,
  Radio,
  Select,
  Switch,
  TimePicker,
} from 'antd'
import dayjs, { type Dayjs } from 'dayjs'
import type { FieldItem } from '@/types/metadata'
import { isRelationType, READONLY_ON_FORM } from '@/lib/fieldTypes'

function enumOptions(field: FieldItem): { label: string; value: string | number }[] {
  const opts = field.options || {}
  const raw = (opts.enum || opts.options || []) as unknown
  if (!Array.isArray(raw)) return []
  return raw.map((item) => {
    if (typeof item === 'string' || typeof item === 'number') {
      return { label: String(item), value: item }
    }
    const obj = item as { label?: string; value?: string | number; name?: string }
    return {
      label: obj.label || obj.name || String(obj.value ?? ''),
      value: obj.value ?? obj.label ?? '',
    }
  })
}

export function renderCellValue(field: FieldItem, value: unknown): ReactNode {
  if (value === null || value === undefined || value === '') return '—'
  const type = field.type

  if (type === 'password' || type === 'encrypted') return '••••••'
  if (type === 'boolean') return value ? '是' : '否'
  if (type === 'color') {
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
        <span
          style={{
            width: 14,
            height: 14,
            borderRadius: 4,
            background: String(value),
            border: '1px solid #d9d9d9',
          }}
        />
        {String(value)}
      </span>
    )
  }
  if (type === 'percent') return `${value}%`
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

interface FieldControlProps {
  field: FieldItem
  value?: unknown
  onChange?: (v: unknown) => void
  disabled?: boolean
}

export function FieldControl({ field, value, onChange, disabled }: FieldControlProps) {
  const type = field.type
  const readOnly = disabled || READONLY_ON_FORM.has(type) || field.isSystem

  if (type === 'boolean') {
    return <Switch checked={Boolean(value)} disabled={readOnly} onChange={(v) => onChange?.(v)} />
  }

  if (type === 'text' || type === 'markdown' || type === 'richText' || type === 'markdownVditor') {
    return (
      <Input.TextArea
        rows={4}
        value={(value as string) ?? ''}
        disabled={readOnly}
        onChange={(e) => onChange?.(e.target.value)}
        placeholder={field.description}
      />
    )
  }

  if (type === 'number' || type === 'integer' || type === 'percent' || type === 'sort') {
    return (
      <InputNumber
        style={{ width: '100%' }}
        value={value as number | null}
        disabled={readOnly}
        precision={type === 'integer' || type === 'sort' ? 0 : undefined}
        addonAfter={type === 'percent' ? '%' : undefined}
        onChange={(v) => onChange?.(v)}
      />
    )
  }

  if (type === 'color') {
    return (
      <ColorPicker
        value={typeof value === 'string' ? value : undefined}
        disabled={readOnly}
        showText
        onChange={(_, hex) => onChange?.(hex)}
      />
    )
  }

  if (type === 'select' || type === 'radio') {
    const options = enumOptions(field)
    if (type === 'radio') {
      return (
        <Radio.Group
          options={options}
          value={value}
          disabled={readOnly}
          onChange={(e) => onChange?.(e.target.value)}
        />
      )
    }
    return (
      <Select
        allowClear
        style={{ width: '100%' }}
        options={options}
        value={value as string | number | undefined}
        disabled={readOnly}
        onChange={(v) => onChange?.(v)}
      />
    )
  }

  if (type === 'multiSelect' || type === 'checkboxGroup') {
    const options = enumOptions(field)
    if (type === 'checkboxGroup') {
      return (
        <Checkbox.Group
          options={options}
          value={(value as (string | number)[]) || []}
          disabled={readOnly}
          onChange={(v) => onChange?.(v)}
        />
      )
    }
    return (
      <Select
        mode="multiple"
        allowClear
        style={{ width: '100%' }}
        options={options}
        value={(value as (string | number)[]) || []}
        disabled={readOnly}
        onChange={(v) => onChange?.(v)}
      />
    )
  }

  if (type === 'date' || type === 'dateTime' || type === 'dateTimeTz') {
    const showTime = type !== 'date'
    let day: Dayjs | null = null
    if (value) day = dayjs(value as string)
    return (
      <DatePicker
        showTime={showTime}
        style={{ width: '100%' }}
        value={day}
        disabled={readOnly}
        onChange={(d) => onChange?.(d ? d.toISOString() : null)}
      />
    )
  }

  if (type === 'time') {
    return (
      <TimePicker
        style={{ width: '100%' }}
        value={value ? dayjs(String(value), 'HH:mm:ss') : null}
        disabled={readOnly}
        onChange={(d) => onChange?.(d ? d.format('HH:mm:ss') : null)}
      />
    )
  }

  if (type === 'unixTimestamp') {
    const day = typeof value === 'number' ? dayjs.unix(value > 1e12 ? value / 1000 : value) : null
    return (
      <DatePicker
        showTime
        style={{ width: '100%' }}
        value={day}
        disabled={readOnly}
        onChange={(d) => onChange?.(d ? d.unix() : null)}
      />
    )
  }

  if (type === 'json' || type === 'point' || type === 'lineString' || type === 'circle' || type === 'polygon') {
    const text = typeof value === 'string' ? value : value != null ? JSON.stringify(value, null, 2) : ''
    return (
      <Input.TextArea
        rows={6}
        value={text}
        disabled={readOnly}
        onChange={(e) => {
          const t = e.target.value
          try {
            onChange?.(t ? JSON.parse(t) : null)
          } catch {
            onChange?.(t)
          }
        }}
        placeholder="JSON"
      />
    )
  }

  if (type === 'password' || type === 'encrypted') {
    return (
      <Input.Password
        value={(value as string) ?? ''}
        disabled={readOnly}
        onChange={(e) => onChange?.(e.target.value)}
      />
    )
  }

  if (isRelationType(type)) {
    // MVP：关系字段以降级为 ID / JSON 输入
    return (
      <Input
        value={value == null ? '' : typeof value === 'object' ? JSON.stringify(value) : String(value)}
        disabled={readOnly}
        placeholder={type === 'belongsTo' ? '外键 ID' : '关联数据（ID 或 JSON）'}
        onChange={(e) => {
          const t = e.target.value.trim()
          if (!t) {
            onChange?.(null)
            return
          }
          if (/^\d+$/.test(t)) {
            onChange?.(Number(t))
            return
          }
          try {
            onChange?.(JSON.parse(t))
          } catch {
            onChange?.(t)
          }
        }}
      />
    )
  }

  return (
    <Input
      value={(value as string) ?? ''}
      disabled={readOnly}
      placeholder={field.description}
      onChange={(e) => onChange?.(e.target.value)}
    />
  )
}

export function buildFormRules(field: FieldItem) {
  const rules: { required?: boolean; message?: string; type?: 'email' | 'url'; pattern?: RegExp }[] = []
  if (field.required) {
    rules.push({ required: true, message: `请填写${field.displayName || field.name}` })
  }
  if (field.type === 'email') {
    rules.push({ type: 'email', message: '请输入有效邮箱' })
  }
  if (field.type === 'url') {
    rules.push({ type: 'url', message: '请输入有效 URL' })
  }
  const validation = field.validation as { pattern?: string; minLength?: number; maxLength?: number } | undefined
  if (validation?.pattern) {
    try {
      rules.push({ pattern: new RegExp(validation.pattern), message: '格式不正确' })
    } catch {
      /* ignore invalid pattern */
    }
  }
  return rules
}
