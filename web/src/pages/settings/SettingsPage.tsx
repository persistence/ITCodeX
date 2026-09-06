import { Alert, App, Button, Form, Input, InputNumber, Radio, Space } from 'antd'
import { useState } from 'react'
import { metaApi } from '@/api'
import { getSettings, saveSettings, type AppSettings } from '@/lib/settings'

export function SettingsPage() {
  const { message } = App.useApp()
  const [form] = Form.useForm<AppSettings>()
  const [testing, setTesting] = useState(false)
  const initial = getSettings()

  const onSave = async () => {
    const values = await form.validateFields()
    saveSettings(values)
    message.success('设置已保存到本地')
  }

  const onTest = async () => {
    const values = await form.validateFields()
    saveSettings({ apiBaseUrl: values.apiBaseUrl })
    setTesting(true)
    try {
      await metaApi.listCollections({ pageSize: 1 })
      message.success('已连接')
    } catch (e) {
      message.error((e as Error).message || '连接失败')
    } finally {
      setTesting(false)
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>设置</h1>
          <p>仅保存在浏览器 localStorage，不调用后端配置接口</p>
        </div>
      </div>

      <div className="panel" style={{ maxWidth: 560 }}>
        <Alert
          style={{ marginBottom: 16 }}
          type="info"
          showIcon
          message="开发环境可将 API Base URL 留空，走 Vite 代理 /api → http://127.0.0.1:8000"
        />
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          message="Yaegi 脚本在服务端沙箱执行，仅可信管理员使用。"
        />
        <Form form={form} layout="vertical" initialValues={initial}>
          <Form.Item
            label="API Base URL"
            name="apiBaseUrl"
            extra="例如 http://127.0.0.1:8000 ；留空则使用相对路径 /api"
          >
            <Input className="mono" placeholder="http://127.0.0.1:8000" />
          </Form.Item>
          <Form.Item label="主题" name="theme">
            <Radio.Group
              options={[
                { value: 'light', label: '浅色' },
                { value: 'dark', label: '深色' },
                { value: 'system', label: '跟随系统' },
              ]}
            />
          </Form.Item>
          <Form.Item label="默认页大小" name="defaultPageSize">
            <InputNumber min={8} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="表格密度" name="tableDensity">
            <Radio.Group
              options={[
                { value: 'default', label: '默认' },
                { value: 'compact', label: '紧凑' },
              ]}
            />
          </Form.Item>
          <Space>
            <Button type="primary" onClick={onSave}>
              保存
            </Button>
            <Button loading={testing} onClick={onTest}>
              测试连接
            </Button>
          </Space>
        </Form>
      </div>
    </div>
  )
}
