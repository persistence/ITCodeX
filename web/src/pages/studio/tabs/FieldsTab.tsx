import {
  App,
  Button,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Table,
  Tag,
} from 'antd'
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { metaApi } from '@/api'
import { ApiError } from '@/api/client'
import { FIELD_TYPE_OPTIONS, fieldTypeLabel } from '@/lib/fieldTypes'
import type { CollectionItem, CreateFieldInput, FieldItem, UpdateFieldInput } from '@/types/metadata'

export function FieldsTab({ collection }: { collection: CollectionItem }) {
  const { message, modal } = App.useApp()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<FieldItem | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['fields', collection.name],
    queryFn: () => metaApi.listFields(collection.name),
  })

  const fields = data?.list || collection.fields || []

  const typeGroups = useMemo(() => {
    const map = new Map<string, { label: string; value: string }[]>()
    FIELD_TYPE_OPTIONS.forEach((opt) => {
      const list = map.get(opt.group) || []
      list.push({ label: opt.label, value: opt.value })
      map.set(opt.group, list)
    })
    return [...map.entries()].map(([group, options]) => ({ label: group, options }))
  }, [])

  const saveMut = useMutation({
    mutationFn: async () => {
      const values = await form.validateFields()
      const enumRaw = values.enumOptions as string | undefined
      const options: Record<string, unknown> = { ...(values.options || {}) }
      if (enumRaw) {
        options.enum = enumRaw
          .split(/[,，\n]/)
          .map((s) => s.trim())
          .filter(Boolean)
      }
      if (values.target) options.target = values.target

      if (editing) {
        const input: UpdateFieldInput = {
          displayName: values.displayName,
          isRequired: values.isRequired,
          isUnique: values.isUnique,
          isIndexed: values.isIndexed,
          options,
          validation: {
            required: values.isRequired,
            minLength: values.minLength,
            maxLength: values.maxLength,
            pattern: values.pattern,
            min: values.min,
            max: values.max,
          },
        }
        return metaApi.updateField(collection.name, editing.name, input)
      }

      const input: CreateFieldInput = {
        name: values.name,
        displayName: values.displayName,
        type: values.type,
        isRequired: values.isRequired,
        isUnique: values.isUnique,
        isIndexed: values.isIndexed,
        description: values.description,
        defaultValue: values.defaultValue,
        options,
        validation: {
          required: values.isRequired,
          minLength: values.minLength,
          maxLength: values.maxLength,
          pattern: values.pattern,
          min: values.min,
          max: values.max,
        },
        target: values.target,
        foreignKey: values.foreignKey,
        expression: values.expression,
        pattern: values.seqPattern,
      }
      return metaApi.createField(collection.name, input)
    },
    onSuccess: () => {
      message.success(editing ? '字段已更新' : '字段已添加')
      setOpen(false)
      setEditing(null)
      form.resetFields()
      qc.invalidateQueries({ queryKey: ['fields', collection.name] })
      qc.invalidateQueries({ queryKey: ['collection', collection.name] })
    },
    onError: (e: Error) => {
      const err = e as ApiError
      message.error(err.message)
    },
  })

  const deleteMut = useMutation({
    mutationFn: (field: string) => metaApi.deleteField(collection.name, field),
    onSuccess: () => {
      message.success('字段已删除')
      qc.invalidateQueries({ queryKey: ['fields', collection.name] })
      qc.invalidateQueries({ queryKey: ['collection', collection.name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ type: 'string', isRequired: false })
    setOpen(true)
  }

  const openEdit = (field: FieldItem) => {
    setEditing(field)
    const opts = field.options || {}
    const enumVal = Array.isArray(opts.enum) ? (opts.enum as string[]).join(', ') : ''
    form.setFieldsValue({
      name: field.name,
      displayName: field.displayName,
      type: field.type,
      isRequired: field.required,
      isUnique: field.unique,
      isIndexed: field.indexed,
      description: (field as FieldItem & { description?: string }).description,
      enumOptions: enumVal,
      target: opts.target || field.target,
      foreignKey: field.foreignKey,
      minLength: (field.validation as { minLength?: number } | undefined)?.minLength,
      maxLength: (field.validation as { maxLength?: number } | undefined)?.maxLength,
      pattern: (field.validation as { pattern?: string } | undefined)?.pattern,
      min: (field.validation as { min?: number } | undefined)?.min,
      max: (field.validation as { max?: number } | undefined)?.max,
    })
    setOpen(true)
  }

  return (
    <div>
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          添加字段
        </Button>
        <Button onClick={() => refetch()}>刷新</Button>
      </Space>

      <Table
        rowKey="name"
        loading={isLoading}
        size="middle"
        dataSource={fields}
        pagination={false}
        columns={[
          { title: '显示名', dataIndex: 'displayName' },
          {
            title: '标识',
            dataIndex: 'name',
            render: (v) => <span className="mono">{v}</span>,
          },
          {
            title: '类型',
            dataIndex: 'type',
            render: (v) => <Tag>{fieldTypeLabel(v)}</Tag>,
          },
          {
            title: '约束',
            render: (_, row) => (
              <Space size={4}>
                {row.required ? <Tag color="red">必填</Tag> : null}
                {row.unique ? <Tag color="gold">唯一</Tag> : null}
                {row.indexed ? <Tag>索引</Tag> : null}
                {row.isSystem ? <Tag color="default">系统</Tag> : null}
              </Space>
            ),
          },
          {
            title: '操作',
            width: 140,
            render: (_, row) => (
              <Space>
                <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(row)} />
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  disabled={row.isSystem}
                  onClick={() => {
                    modal.confirm({
                      title: `删除字段「${row.displayName || row.name}」？`,
                      content: '可能丢失该列数据，确认后不可恢复。',
                      okButtonProps: { danger: true },
                      onOk: () => deleteMut.mutateAsync(row.name),
                    })
                  }}
                />
              </Space>
            ),
          },
        ]}
      />

      <Drawer
        title={editing ? `编辑字段 · ${editing.name}` : '添加字段'}
        open={open}
        onClose={() => setOpen(false)}
        width={520}
        destroyOnClose
        extra={
          <Button type="primary" loading={saveMut.isPending} onClick={() => saveMut.mutate()}>
            保存
          </Button>
        }
      >
        <Form form={form} layout="vertical">
          {!editing && (
            <Form.Item label="类型" name="type" rules={[{ required: true }]}>
              <Select
                showSearch
                optionFilterProp="label"
                options={typeGroups}
              />
            </Form.Item>
          )}
          <Form.Item label="显示名称" name="displayName" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          {!editing && (
            <Form.Item
              label="标识名称"
              name="name"
              rules={[
                { required: true },
                { pattern: /^[a-zA-Z][a-zA-Z0-9_]*$/, message: '字母开头，仅字母数字下划线' },
              ]}
            >
              <Input className="mono" />
            </Form.Item>
          )}
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Space size="large">
            <Form.Item label="必填" name="isRequired" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item label="唯一" name="isUnique" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item label="索引" name="isIndexed" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>

          <Form.Item noStyle shouldUpdate={(p, c) => p.type !== c.type}>
            {({ getFieldValue }) => {
              const t = editing?.type || getFieldValue('type')
              const choice = ['select', 'radio', 'multiSelect', 'checkboxGroup'].includes(t)
              const relation = ['belongsTo', 'hasOne', 'hasMany', 'belongsToMany', 'belongsToManyArray'].includes(t)
              return (
                <>
                  {choice && (
                    <Form.Item label="选项（逗号或换行分隔）" name="enumOptions">
                      <Input.TextArea rows={3} placeholder="pending, paid, shipped" />
                    </Form.Item>
                  )}
                  {relation && (
                    <>
                      <Form.Item label="目标表 target" name="target">
                        <Input className="mono" placeholder="customers" />
                      </Form.Item>
                      <Form.Item label="外键 foreignKey" name="foreignKey">
                        <Input className="mono" placeholder="customerId" />
                      </Form.Item>
                    </>
                  )}
                  {t === 'formula' && (
                    <Form.Item label="公式表达式" name="expression">
                      <Input.TextArea rows={2} className="mono" />
                    </Form.Item>
                  )}
                  {t === 'sequence' && (
                    <Form.Item label="序列 pattern" name="seqPattern">
                      <Input className="mono" placeholder="ORD-{YYYY}{MM}-{0000}" />
                    </Form.Item>
                  )}
                </>
              )
            }}
          </Form.Item>

          <Form.Item label="最小长度" name="minLength">
            <InputNumber style={{ width: '100%' }} min={0} />
          </Form.Item>
          <Form.Item label="最大长度" name="maxLength">
            <InputNumber style={{ width: '100%' }} min={0} />
          </Form.Item>
          <Form.Item label="正则" name="pattern">
            <Input className="mono" />
          </Form.Item>
          <Form.Item label="最小值" name="min">
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="最大值" name="max">
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  )
}
