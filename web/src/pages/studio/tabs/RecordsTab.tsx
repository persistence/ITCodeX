import {
  Alert,
  App,
  Button,
  Descriptions,
  Drawer,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tag,
} from 'antd'
import { DeleteOutlined, EditOutlined, EyeOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState, type Key } from 'react'
import { dataApi, metaApi } from '@/api'
import { ApiError } from '@/api/client'
import { buildFormRules, FieldControl, renderCellValue } from '@/components/FieldRenderer'
import { isSearchableType, READONLY_ON_FORM } from '@/lib/fieldTypes'
import { getSettings } from '@/lib/settings'
import type { CollectionItem, FieldItem, RecordData } from '@/types/metadata'

export function RecordsTab({ collection }: { collection: CollectionItem }) {
  const { message, modal } = App.useApp()
  const qc = useQueryClient()
  const settings = getSettings()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(settings.defaultPageSize)
  const [keyword, setKeyword] = useState('')
  const [sort, setSort] = useState<string | undefined>('-createdAt')
  const [selected, setSelected] = useState<Key[]>([])
  const [drawer, setDrawer] = useState<'create' | 'edit' | 'view' | null>(null)
  const [current, setCurrent] = useState<RecordData | null>(null)
  const [tableErrors, setTableErrors] = useState<string[]>([])
  const [form] = Form.useForm()

  const { data: fieldsData } = useQuery({
    queryKey: ['fields', collection.name],
    queryFn: () => metaApi.listFields(collection.name),
  })

  const fields = fieldsData?.list || collection.fields || []
  const editableFields = fields.filter((f) => !READONLY_ON_FORM.has(f.type) && f.name !== 'id')
  const pk = collection.filterTargetKey || 'id'

  const filter = useMemo(() => {
    const q = keyword.trim()
    if (!q) return undefined
    const searchable = fields.filter((f) => isSearchableType(f.type))
    if (searchable.length === 0) {
      return { [pk]: q }
    }
    return {
      $or: searchable.map((f) => ({ [f.name]: { $like: `%${q}%` } })),
    }
  }, [keyword, fields, pk])

  const listQuery = useQuery({
    queryKey: ['records', collection.name, page, pageSize, keyword, sort],
    queryFn: () =>
      dataApi.list(collection.name, {
        page,
        pageSize,
        sort,
        filter,
        except: fields.some((f) => f.type === 'password') ? 'password' : undefined,
      }),
  })

  const openCreate = () => {
    setCurrent(null)
    setTableErrors([])
    form.resetFields()
    setDrawer('create')
  }

  const openEdit = (record: RecordData) => {
    setCurrent(record)
    setTableErrors([])
    form.setFieldsValue(record)
    setDrawer('edit')
  }

  const openView = (record: RecordData) => {
    setCurrent(record)
    setDrawer('view')
  }

  const saveMut = useMutation({
    mutationFn: async () => {
      const values = await form.validateFields()
      setTableErrors([])
      if (drawer === 'edit' && current) {
        return dataApi.update(collection.name, current[pk] as string | number, values)
      }
      return dataApi.create(collection.name, values)
    },
    onSuccess: () => {
      message.success(drawer === 'edit' ? '已更新' : '已创建')
      setDrawer(null)
      qc.invalidateQueries({ queryKey: ['records', collection.name] })
      qc.invalidateQueries({ queryKey: ['count', collection.name] })
    },
    onError: (e: Error) => {
      const err = e as ApiError
      const validation = err.validation
      if (validation) {
        const fieldErrors = Object.entries(validation.fieldErrors || {}).map(([name, msgs]) => ({
          name,
          errors: msgs,
        }))
        form.setFields(fieldErrors)
        setTableErrors(validation.tableErrors || [])
      }
      message.error(err.message)
    },
  })

  const deleteMut = useMutation({
    mutationFn: (id: string | number) => dataApi.remove(collection.name, id),
    onSuccess: () => {
      message.success('已删除')
      qc.invalidateQueries({ queryKey: ['records', collection.name] })
      qc.invalidateQueries({ queryKey: ['count', collection.name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const batchDeleteMut = useMutation({
    mutationFn: (ids: (string | number)[]) =>
      dataApi.removeMany(collection.name, { [pk]: { $in: ids } }),
    onSuccess: () => {
      message.success('批量删除完成')
      setSelected([])
      qc.invalidateQueries({ queryKey: ['records', collection.name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const columns = [
    ...fields
      .filter((f) => f.type !== 'password' && f.type !== 'encrypted')
      .slice(0, 8)
      .map((f) => ({
        title: f.displayName || f.name,
        dataIndex: f.name,
        sorter: true,
        ellipsis: true,
        render: (v: unknown) => renderCellValue(f, v),
      })),
    {
      title: '操作',
      key: 'actions',
      width: 150,
      fixed: 'right' as const,
      render: (_: unknown, row: RecordData) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => openView(row)} />
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(row)} />
          <Button
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => {
              modal.confirm({
                title: '删除这条记录？',
                okButtonProps: { danger: true },
                onOk: () => deleteMut.mutateAsync(row[pk] as string | number),
              })
            }}
          />
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space wrap style={{ marginBottom: 12 }}>
        <Input.Search
          allowClear
          placeholder="搜索文本字段"
          style={{ width: 240 }}
          onSearch={(v) => {
            setPage(1)
            setKeyword(v)
          }}
        />
        <Select
          allowClear
          placeholder="排序"
          style={{ width: 180 }}
          value={sort}
          onChange={setSort}
          options={[
            { value: '-createdAt', label: '创建时间 ↓' },
            { value: 'createdAt', label: '创建时间 ↑' },
            { value: `-${pk}`, label: `${pk} ↓` },
            { value: pk, label: `${pk} ↑` },
          ]}
        />
        <Button icon={<ReloadOutlined />} onClick={() => listQuery.refetch()}>
          刷新
        </Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} disabled={fields.length === 0}>
          新增记录
        </Button>
        {selected.length > 0 && (
          <Button
            danger
            onClick={() => {
              modal.confirm({
                title: `批量删除 ${selected.length} 条记录？`,
                okButtonProps: { danger: true },
                onOk: () => batchDeleteMut.mutateAsync(selected as (string | number)[]),
              })
            }}
          >
            批量删除 ({selected.length})
          </Button>
        )}
      </Space>

      {fields.length === 0 ? (
        <Alert type="info" showIcon message="请先添加字段后再录入数据" />
      ) : (
        <Table
          rowKey={(r) => String(r[pk])}
          loading={listQuery.isLoading}
          dataSource={listQuery.data?.list || []}
          columns={columns}
          size={settings.tableDensity === 'compact' ? 'small' : 'middle'}
          rowSelection={{
            selectedRowKeys: selected,
            onChange: setSelected,
          }}
          pagination={{
            current: page,
            pageSize,
            total: listQuery.data?.total,
            showSizeChanger: true,
            pageSizeOptions: [8, 20, 50, 100],
            onChange: (p, ps) => {
              setPage(p)
              setPageSize(ps)
            },
          }}
          scroll={{ x: true }}
          onChange={(_pag, _f, sorter) => {
            const s = Array.isArray(sorter) ? sorter[0] : sorter
            if (s?.field && s.order) {
              const field = String(s.field)
              setSort(s.order === 'descend' ? `-${field}` : field)
            }
          }}
        />
      )}

      <Drawer
        title={
          drawer === 'create' ? '新增记录' : drawer === 'edit' ? '编辑记录' : '记录详情'
        }
        open={drawer !== null}
        onClose={() => setDrawer(null)}
        width={560}
        destroyOnClose
        extra={
          drawer !== 'view' ? (
            <Button type="primary" loading={saveMut.isPending} onClick={() => saveMut.mutate()}>
              保存
            </Button>
          ) : (
            <Button
              type="primary"
              onClick={() => {
                if (current) openEdit(current)
              }}
            >
              编辑
            </Button>
          )
        }
      >
        {drawer === 'view' && current ? (
          <Descriptions column={1} bordered size="small">
            {fields.map((f) => (
              <Descriptions.Item key={f.name} label={f.displayName || f.name}>
                {renderCellValue(f, current[f.name])}
              </Descriptions.Item>
            ))}
          </Descriptions>
        ) : (
          <>
            {tableErrors.length > 0 && (
              <Alert
                type="error"
                showIcon
                style={{ marginBottom: 12 }}
                message={tableErrors.join('；')}
              />
            )}
            <Form form={form} layout="vertical">
              {editableFields.map((field: FieldItem) => (
                <Form.Item
                  key={field.name}
                  name={field.name}
                  label={
                    <Space>
                      {field.displayName || field.name}
                      {field.required ? <Tag color="red">必填</Tag> : null}
                    </Space>
                  }
                  rules={buildFormRules(field)}
                >
                  <FieldControl field={field} />
                </Form.Item>
              ))}
            </Form>
          </>
        )}
      </Drawer>
    </div>
  )
}
