import { App, Button, Descriptions, Form, Input, Space, Switch, Tag } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { dataApi, metaApi } from '@/api'
import type { CollectionItem, CollectionOptions } from '@/types/metadata'

export function OverviewTab({ collection }: { collection: CollectionItem }) {
  const { message } = App.useApp()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [form] = Form.useForm()

  const { data: countData } = useQuery({
    queryKey: ['count', collection.name],
    queryFn: () => dataApi.count(collection.name),
  })

  const updateMut = useMutation({
    mutationFn: async () => {
      const values = await form.validateFields()
      const categories =
        typeof values.categories === 'string'
          ? values.categories
              .split(/[,，]/)
              .map((s: string) => s.trim())
              .filter(Boolean)
          : values.categories || []
      const options: CollectionOptions = {
        ...(collection.options as CollectionOptions),
        simplePagination: Boolean(values.simplePagination),
      }
      return metaApi.updateCollection(collection.name, {
        displayName: values.displayName,
        description: values.description,
        categories,
        options,
      })
    },
    onSuccess: () => {
      message.success('已保存')
      qc.invalidateQueries({ queryKey: ['collection', collection.name] })
      qc.invalidateQueries({ queryKey: ['collections'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const opts = (collection.options || {}) as CollectionOptions

  return (
    <div>
      <Space style={{ marginBottom: 16 }} wrap>
        <Tag color="processing">记录数 {countData?.count ?? '—'}</Tag>
        <Tag>表名 {collection.tableName || collection.name}</Tag>
        <Button type="primary" onClick={() => navigate(`/collections/${collection.name}/records`)}>
          管理数据
        </Button>
        <Button onClick={() => navigate(`/collections/${collection.name}/fields`)}>配置字段</Button>
      </Space>

      <Descriptions column={2} size="small" style={{ marginBottom: 20 }} bordered>
        <Descriptions.Item label="标识名">
          <span className="mono">{collection.name}</span>
        </Descriptions.Item>
        <Descriptions.Item label="类型">{collection.type}</Descriptions.Item>
        <Descriptions.Item label="字段数">
          {collection.fieldCount ?? collection.fields?.length ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label="简单分页">{opts.simplePagination ? '是' : '否'}</Descriptions.Item>
      </Descriptions>

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          displayName: collection.displayName,
          description: collection.description,
          categories: (collection.categories || []).join(', '),
          simplePagination: Boolean(opts.simplePagination),
        }}
        style={{ maxWidth: 560 }}
      >
        <Form.Item label="显示名称" name="displayName" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label="分类" name="categories">
          <Input placeholder="逗号分隔" />
        </Form.Item>
        <Form.Item label="描述" name="description">
          <Input.TextArea rows={3} />
        </Form.Item>
        <Form.Item label="简单分页" name="simplePagination" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Button type="primary" loading={updateMut.isPending} onClick={() => updateMut.mutate()}>
          保存基本信息
        </Button>
      </Form>
    </div>
  )
}
