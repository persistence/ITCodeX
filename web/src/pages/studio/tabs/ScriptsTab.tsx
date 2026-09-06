import { Alert, App, Button, Drawer, Form, Input, InputNumber, Select, Space, Switch, Table, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { metaApi } from '@/api'
import { HOOK_POINTS } from '@/lib/fieldTypes'
import type { YaegiScript } from '@/types/metadata'

const HOOK_TEMPLATE = `package main

import (
\t"context"
)

func BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {
\t// TODO: 在此修改 data
\treturn data, nil
}
`

const API_TEMPLATE = `package main

import (
\t"itcodex/context"
)

func Handle(ctx *context.YaegiHTTPContext) {
\tctx.Response.JSONSuccess(map[string]string{"message": "ok"})
}
`

export function ScriptsTab({ collectionName }: { collectionName?: string }) {
  const { message, modal } = App.useApp()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<YaegiScript | null>(null)
  const [validateMsg, setValidateMsg] = useState<string | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading } = useQuery({
    queryKey: ['scripts', collectionName || 'all'],
    queryFn: () => metaApi.listScripts(collectionName ? { collection: collectionName } : undefined),
  })

  const { data: collections } = useQuery({
    queryKey: ['collections', 'for-scripts'],
    queryFn: () => metaApi.listCollections({ pageSize: 200 }),
  })

  const saveMut = useMutation({
    mutationFn: async () => {
      const values = await form.validateFields()
      const payload: YaegiScript = {
        ...(editing?.id ? { id: editing.id } : {}),
        name: values.name,
        collectionName: values.collectionName,
        hookPoint: values.hookPoint,
        content: values.content,
        apiPath: values.apiPath,
        httpMethod: values.httpMethod,
        priority: values.priority ?? 0,
        enabled: values.enabled !== false,
      }
      return metaApi.saveScript(payload)
    },
    onSuccess: () => {
      message.success('脚本已保存')
      setOpen(false)
      setEditing(null)
      qc.invalidateQueries({ queryKey: ['scripts'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const disableMut = useMutation({
    mutationFn: (id: number) => metaApi.disableScript(id),
    onSuccess: () => {
      message.success('已禁用')
      qc.invalidateQueries({ queryKey: ['scripts'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const enableMut = useMutation({
    mutationFn: (script: YaegiScript) =>
      metaApi.saveScript({
        ...script,
        enabled: true,
      }),
    onSuccess: () => {
      message.success('已启用')
      qc.invalidateQueries({ queryKey: ['scripts'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => metaApi.deleteScript(id),
    onSuccess: () => {
      message.success('已删除')
      qc.invalidateQueries({ queryKey: ['scripts'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const openCreate = () => {
    setEditing(null)
    setValidateMsg(null)
    form.resetFields()
    form.setFieldsValue({
      collectionName,
      hookPoint: 'beforeCreate',
      enabled: true,
      priority: 0,
      content: HOOK_TEMPLATE,
      httpMethod: 'POST',
    })
    setOpen(true)
  }

  const openEdit = (script: YaegiScript) => {
    setEditing(script)
    setValidateMsg(null)
    form.setFieldsValue(script)
    setOpen(true)
  }

  return (
    <div>
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 12 }}
        message="脚本在服务端沙箱执行，具有库表访问能力；仅可信管理员使用。本前端不执行脚本。"
      />
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建脚本
        </Button>
      </Space>
      <Table
        rowKey={(r) => String(r.id)}
        loading={isLoading}
        dataSource={data?.list || []}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '表', dataIndex: 'collectionName', render: (v) => v || '—' },
          { title: '钩子点', dataIndex: 'hookPoint', render: (v) => <Tag>{v}</Tag> },
          {
            title: '方法/路径',
            render: (_, r) =>
              r.hookPoint === 'customAPI' ? (
                <span className="mono">
                  {r.httpMethod} {r.apiPath}
                </span>
              ) : (
                '—'
              ),
          },
          {
            title: '状态',
            dataIndex: 'enabled',
            render: (v) => (v ? <Tag color="green">启用</Tag> : <Tag>禁用</Tag>),
          },
          { title: '优先级', dataIndex: 'priority' },
          {
            title: '操作',
            render: (_, row) => (
              <Space>
                <Button size="small" onClick={() => openEdit(row)}>
                  编辑
                </Button>
                {row.id ? (
                  row.enabled ? (
                    <Button size="small" onClick={() => disableMut.mutate(row.id!)}>
                      禁用
                    </Button>
                  ) : (
                    <Button size="small" onClick={() => enableMut.mutate(row)}>
                      启用
                    </Button>
                  )
                ) : null}
                <Button
                  size="small"
                  danger
                  onClick={() => {
                    modal.confirm({
                      title: '删除脚本？',
                      okButtonProps: { danger: true },
                      onOk: () => deleteMut.mutateAsync(row.id!),
                    })
                  }}
                >
                  删除
                </Button>
              </Space>
            ),
          },
        ]}
      />

      <Drawer
        title={editing ? '编辑脚本' : '新建脚本'}
        open={open}
        onClose={() => setOpen(false)}
        width={820}
        destroyOnClose
        extra={
          <Space>
            <Button
              onClick={async () => {
                const content = form.getFieldValue('content')
                try {
                  const res = await metaApi.validateScript(content || '')
                  setValidateMsg(res.valid ? '语法正确' : res.error || '语法错误')
                  if (res.valid) message.success('语法正确')
                  else message.error(res.error || '语法错误')
                } catch (e) {
                  message.error((e as Error).message)
                }
              }}
            >
              校验语法
            </Button>
            <Button type="primary" loading={saveMut.isPending} onClick={() => saveMut.mutate()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Space style={{ width: '100%' }} size="middle" align="start">
            <div style={{ width: 280 }}>
              <Form.Item label="名称" name="name" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item label="绑定表" name="collectionName">
                <Select
                  allowClear
                  showSearch
                  options={(collections?.list || []).map((c) => ({
                    value: c.name,
                    label: `${c.displayName || c.name} (${c.name})`,
                  }))}
                />
              </Form.Item>
              <Form.Item label="钩子点" name="hookPoint" rules={[{ required: true }]}>
                <Select options={HOOK_POINTS.map((h) => ({ value: h, label: h }))} />
              </Form.Item>
              <Form.Item noStyle shouldUpdate={(p, c) => p.hookPoint !== c.hookPoint}>
                {({ getFieldValue }) =>
                  getFieldValue('hookPoint') === 'customAPI' ? (
                    <>
                      <Form.Item label="HTTP 方法" name="httpMethod">
                        <Select options={['GET', 'POST', 'PUT', 'DELETE'].map((m) => ({ value: m, label: m }))} />
                      </Form.Item>
                      <Form.Item label="API Path" name="apiPath" rules={[{ required: true }]}>
                        <Input className="mono" placeholder="/c/users/:id/change-password" />
                      </Form.Item>
                    </>
                  ) : null
                }
              </Form.Item>
              <Form.Item label="优先级" name="priority">
                <InputNumber style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item label="启用" name="enabled" valuePropName="checked">
                <Switch />
              </Form.Item>
              <Space>
                <Button
                  size="small"
                  onClick={() => form.setFieldValue('content', HOOK_TEMPLATE)}
                >
                  钩子模板
                </Button>
                <Button size="small" onClick={() => form.setFieldValue('content', API_TEMPLATE)}>
                  API 模板
                </Button>
              </Space>
              {validateMsg && (
                <p className="muted" style={{ marginTop: 12 }}>
                  {validateMsg}
                </p>
              )}
            </div>
            <div style={{ flex: 1 }}>
              <Form.Item label="脚本内容" name="content" rules={[{ required: true }]}>
                <Input.TextArea
                  rows={24}
                  className="mono"
                  style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' }}
                />
              </Form.Item>
            </div>
          </Space>
        </Form>
      </Drawer>
    </div>
  )
}
