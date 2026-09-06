import { App, Button, Card, Checkbox, Form, Input, Space, Switch, Steps, Alert } from 'antd'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { metaApi } from '@/api'
import { COLLECTION_TYPES, PRESET_FIELDS, suggestNameFromDisplay } from '@/lib/fieldTypes'
import type { CollectionType, CreateCollectionInput } from '@/types/metadata'

export function CreateCollectionPage() {
  const [step, setStep] = useState(0)
  const [type, setType] = useState<CollectionType>('general')
  const [syncAfter, setSyncAfter] = useState(true)
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const { message } = App.useApp()
  const watchedName = Form.useWatch('name', form)
  const watchedDisplayName = Form.useWatch('displayName', form)

  const createMut = useMutation({
    mutationFn: async (input: CreateCollectionInput) => {
      const created = await metaApi.createCollection(input)
      if (syncAfter) {
        try {
          await metaApi.syncCollection(created.name || input.name)
        } catch (e) {
          message.warning(`表已创建，但同步失败：${(e as Error).message}`)
        }
      }
      return created
    },
    onSuccess: (data) => {
      message.success('创建成功')
      navigate(`/collections/${data.name}/fields`)
    },
    onError: (e: Error) => message.error(e.message),
  })

  const goBasicInfo = () => setStep(1)

  const goPresets = async () => {
    try {
      await form.validateFields(['displayName', 'name', 'filterTargetKey'])
      setStep(2)
    } catch {
      setStep(1)
    }
  }

  const onFinish = async () => {
    try {
      await form.validateFields(['displayName', 'name'])
    } catch {
      message.error('请先完善基本信息：标识名称必填，且须以字母开头')
      setStep(1)
      return
    }

    const values = form.getFieldsValue(true)
    const name = String(values.name || '').trim()
    if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(name)) {
      message.error('标识名称无效：字母开头，仅字母数字下划线')
      setStep(1)
      return
    }

    const categories =
      typeof values.categories === 'string'
        ? values.categories
            .split(/[,，]/)
            .map((s: string) => s.trim())
            .filter(Boolean)
        : values.categories || []

    const input: CreateCollectionInput = {
      name,
      displayName: values.displayName,
      type,
      description: values.description,
      categories,
      autoGenId: values.autoGenId !== false,
      filterTargetKey: values.filterTargetKey || 'id',
      presetFields: values.presetFields || ['id', 'createdAt', 'updatedAt', 'createdBy', 'updatedBy'],
      options: {
        simplePagination: Boolean(values.simplePagination),
        ...(type === 'tree' ? { treeParentKey: 'parentId' } : {}),
      },
    }
    createMut.mutate(input)
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>新建数据表</h1>
          <p>选择类型、填写基本信息，创建后可继续配置字段</p>
        </div>
        <Button onClick={() => navigate('/')}>返回</Button>
      </div>

      <div className="panel" style={{ marginBottom: 16 }}>
        <Steps
          current={step}
          items={[{ title: '选择类型' }, { title: '基本信息' }, { title: '预设字段' }]}
          onChange={async (s) => {
            if (s >= 2) {
              await goPresets()
              return
            }
            setStep(s)
          }}
        />
      </div>

      {/* Form 始终挂载，避免回退步骤丢值 */}
      <Form
        form={form}
        layout="vertical"
        preserve
        initialValues={{
          autoGenId: true,
          filterTargetKey: 'id',
          presetFields: ['id', 'createdAt', 'updatedAt', 'createdBy', 'updatedBy'],
          simplePagination: false,
        }}
        onValuesChange={(changed) => {
          if ('displayName' in changed && !form.isFieldTouched('name')) {
            const suggested = suggestNameFromDisplay(changed.displayName || '')
            if (suggested) form.setFieldValue('name', suggested)
          }
        }}
      >
        <div style={{ display: step === 0 ? 'block' : 'none' }}>
          <div className="collection-grid">
            {COLLECTION_TYPES.map((t) => (
              <div
                key={t.value}
                className={`type-card ${type === t.value ? 'active' : ''}`}
                onClick={() => setType(t.value)}
              >
                <h3 style={{ margin: '0 0 6px' }}>{t.label}</h3>
                <div className="muted" style={{ fontSize: 13 }}>
                  {t.desc}
                </div>
              </div>
            ))}
            <div style={{ gridColumn: '1 / -1' }}>
              <Button type="primary" onClick={goBasicInfo}>
                下一步
              </Button>
            </div>
          </div>
        </div>

        <Card style={{ display: step === 1 || step === 2 ? 'block' : 'none' }}>
          <div style={{ display: step === 1 ? 'block' : 'none' }}>
            <Form.Item
              label="显示名称"
              name="displayName"
              rules={[{ required: true, message: '请输入显示名称' }]}
            >
              <Input placeholder="例如：订单" />
            </Form.Item>
            <Form.Item
              label="标识名称"
              name="name"
              rules={[
                { required: true, message: '请输入标识名称' },
                {
                  pattern: /^[a-zA-Z][a-zA-Z0-9_]*$/,
                  message: '字母开头，仅字母数字下划线（中文显示名需单独填写英文标识）',
                },
              ]}
              extra="创建后不可修改，用于 API 路径。纯中文显示名不会自动生成标识，请手动填写如 orders"
            >
              <Input className="mono" placeholder="例如：orders" />
            </Form.Item>
            <Form.Item label="分类" name="categories" extra="多个分类用逗号分隔">
              <Input placeholder="销售, 系统" />
            </Form.Item>
            <Form.Item label="描述" name="description">
              <Input.TextArea rows={3} />
            </Form.Item>
            <Form.Item
              label="记录定位键"
              name="filterTargetKey"
              extra="当前后端默认使用 id；此配置预留展示"
            >
              <Input placeholder="id" disabled />
            </Form.Item>
            <Form.Item label="自动生成 ID" name="autoGenId" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item label="简单分页模式" name="simplePagination" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Space>
              <Button onClick={() => setStep(0)}>上一步</Button>
              <Button type="primary" onClick={goPresets}>
                下一步
              </Button>
            </Space>
          </div>

          <div style={{ display: step === 2 ? 'block' : 'none' }}>
            {(watchedDisplayName || watchedName) && (
              <Alert
                style={{ marginBottom: 16 }}
                type="info"
                showIcon
                message={`即将创建：${watchedDisplayName || '未命名'}（${watchedName || '缺少标识'}）`}
                description={`类型：${COLLECTION_TYPES.find((t) => t.value === type)?.label || type}`}
              />
            )}
            <Form.Item label="预设字段" name="presetFields">
              <Checkbox.Group
                options={PRESET_FIELDS.map((f) => ({ label: f.label, value: f.value }))}
              />
            </Form.Item>
            {type !== 'general' && (
              <p className="muted">
                当前类型为「{COLLECTION_TYPES.find((t) => t.value === type)?.label}」，保存后可能写入类型专属字段（如
                parentId、起止时间），请勿随意删除。
              </p>
            )}
            <Form.Item label="创建后同步到数据库">
              <Switch checked={syncAfter} onChange={setSyncAfter} />
            </Form.Item>
            <Space>
              <Button onClick={() => setStep(1)}>上一步</Button>
              <Button type="primary" loading={createMut.isPending} onClick={onFinish}>
                创建
              </Button>
            </Space>
          </div>
        </Card>
      </Form>
    </div>
  )
}
