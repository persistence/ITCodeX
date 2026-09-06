import { App, Button, Form, Input, Modal, Select, Space, Switch, Table, Tag } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { metaApi } from '@/api'
import type { CollectionItem } from '@/types/metadata'

export function IndexesTab({ collection }: { collection: CollectionItem }) {
  const { message, modal } = App.useApp()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [form] = Form.useForm()

  const { data, isLoading } = useQuery({
    queryKey: ['indexes', collection.name],
    queryFn: () => metaApi.listIndexes(collection.name),
  })

  const { data: fieldsData } = useQuery({
    queryKey: ['fields', collection.name],
    queryFn: () => metaApi.listFields(collection.name),
  })

  const createMut = useMutation({
    mutationFn: async () => {
      const values = await form.validateFields()
      return metaApi.createIndex(collection.name, {
        name: values.name,
        fields: values.fields,
        unique: Boolean(values.unique),
      })
    },
    onSuccess: () => {
      message.success('索引已创建')
      setOpen(false)
      form.resetFields()
      qc.invalidateQueries({ queryKey: ['indexes', collection.name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const deleteMut = useMutation({
    mutationFn: (fields: string[]) => metaApi.deleteIndex(collection.name, fields),
    onSuccess: () => {
      message.success('索引已删除')
      qc.invalidateQueries({ queryKey: ['indexes', collection.name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  return (
    <div>
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          新建索引
        </Button>
      </Space>
      <Table
        rowKey={(r) => r.name || r.fields.join('-')}
        loading={isLoading}
        dataSource={data?.list || []}
        pagination={false}
        columns={[
          { title: '名称', dataIndex: 'name', render: (v) => v || '—' },
          {
            title: '字段',
            dataIndex: 'fields',
            render: (fs: string[]) => fs.map((f) => <Tag key={f}>{f}</Tag>),
          },
          {
            title: '唯一',
            dataIndex: 'unique',
            render: (v) => (v ? <Tag color="gold">唯一</Tag> : '否'),
          },
          {
            title: '操作',
            render: (_, row) => (
              <Button
                danger
                size="small"
                onClick={() => {
                  modal.confirm({
                    title: '删除该索引？',
                    okButtonProps: { danger: true },
                    onOk: () => deleteMut.mutateAsync(row.fields),
                  })
                }}
              >
                删除
              </Button>
            ),
          },
        ]}
      />

      <Modal
        title="新建索引"
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => createMut.mutate()}
        confirmLoading={createMut.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item label="名称" name="name">
            <Input className="mono" placeholder="可选" />
          </Form.Item>
          <Form.Item label="字段" name="fields" rules={[{ required: true, message: '请选择字段' }]}>
            <Select
              mode="multiple"
              options={(fieldsData?.list || []).map((f) => ({
                value: f.name,
                label: `${f.displayName || f.name} (${f.name})`,
              }))}
            />
          </Form.Item>
          <Form.Item label="唯一索引" name="unique" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
