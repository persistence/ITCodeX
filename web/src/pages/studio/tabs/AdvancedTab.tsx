import { Alert, App, Button, Form, Input, Space } from 'antd'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { metaApi } from '@/api'
import type { CollectionItem, CollectionOptions } from '@/types/metadata'

export function AdvancedTab({ collection }: { collection: CollectionItem }) {
  const { message, modal } = App.useApp()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [form] = Form.useForm()
  const opts = (collection.options || {}) as CollectionOptions

  const saveMut = useMutation({
    mutationFn: async () => {
      const values = await form.validateFields()
      return metaApi.updateCollection(collection.name, {
        options: {
          ...opts,
          treeParentKey: values.treeParentKey || opts.treeParentKey,
          calendarStartField: values.calendarStartField || opts.calendarStartField,
          calendarEndField: values.calendarEndField || opts.calendarEndField,
          commentForeignKey: values.commentForeignKey || opts.commentForeignKey,
        },
      })
    },
    onSuccess: () => {
      message.success('已保存')
      qc.invalidateQueries({ queryKey: ['collection', collection.name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const syncMut = useMutation({
    mutationFn: () => metaApi.syncCollection(collection.name),
    onSuccess: () => message.success('同步成功'),
    onError: (e: Error) => message.error(e.message),
  })

  const deleteMut = useMutation({
    mutationFn: () => metaApi.deleteCollection(collection.name),
    onSuccess: () => {
      message.success('表已删除')
      qc.invalidateQueries({ queryKey: ['collections'] })
      navigate('/')
    },
    onError: (e: Error) => message.error(e.message),
  })

  return (
    <div>
      <Alert
        style={{ marginBottom: 16 }}
        type="info"
        showIcon
        message="高级操作"
        description="同步会将元数据结构应用到数据库；删除表不可恢复。"
      />

      <Space style={{ marginBottom: 24 }}>
        <Button type="primary" loading={syncMut.isPending} onClick={() => syncMut.mutate()}>
          同步到数据库
        </Button>
        <Button
          danger
          onClick={() => {
            let confirmName = ''
            modal.confirm({
              title: '危险：删除整个数据表',
              content: (
                <div>
                  <p>将删除元数据、真实表及全部数据。请输入表标识名确认：</p>
                  <Input
                    className="mono"
                    placeholder={collection.name}
                    onChange={(e) => {
                      confirmName = e.target.value
                    }}
                  />
                </div>
              ),
              okButtonProps: { danger: true },
              onOk: () => {
                if (confirmName !== collection.name) {
                  message.error('标识名不匹配')
                  return Promise.reject()
                }
                return deleteMut.mutateAsync()
              },
            })
          }}
        >
          删除数据表
        </Button>
      </Space>

      {(collection.type === 'tree' ||
        collection.type === 'calendar' ||
        collection.type === 'comment') && (
        <Form
          form={form}
          layout="vertical"
          style={{ maxWidth: 480 }}
          initialValues={{
            treeParentKey: opts.treeParentKey || 'parentId',
            calendarStartField: opts.calendarStartField,
            calendarEndField: opts.calendarEndField,
            commentForeignKey: opts.commentForeignKey,
          }}
        >
          {collection.type === 'tree' && (
            <Form.Item label="树表父键 treeParentKey" name="treeParentKey">
              <Input className="mono" />
            </Form.Item>
          )}
          {collection.type === 'calendar' && (
            <>
              <Form.Item label="开始时间字段" name="calendarStartField">
                <Input className="mono" />
              </Form.Item>
              <Form.Item label="结束时间字段" name="calendarEndField">
                <Input className="mono" />
              </Form.Item>
            </>
          )}
          {collection.type === 'comment' && (
            <Form.Item label="评论外键 commentForeignKey" name="commentForeignKey">
              <Input className="mono" />
            </Form.Item>
          )}
          <Button type="primary" loading={saveMut.isPending} onClick={() => saveMut.mutate()}>
            保存类型配置
          </Button>
        </Form>
      )}

      {collection.type === 'file' && (
        <Alert
          type="warning"
          showIcon
          message="文件表仅管理元数据"
          description="本端不包含对象存储引擎，请通过 URL / 元信息字段维护附件信息。"
        />
      )}
    </div>
  )
}
