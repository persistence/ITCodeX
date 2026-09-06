import { Menu } from 'antd'
import {
  CodeOutlined,
  DatabaseOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { Link, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { CommandPalette } from '@/components/CommandPalette'

export function AppLayout({ children }: { children: ReactNode }) {
  const location = useLocation()

  const selected = location.pathname.startsWith('/scripts')
    ? 'scripts'
    : location.pathname.startsWith('/settings')
      ? 'settings'
      : 'collections'

  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="app-brand">
          <strong>ITCodeX Meta</strong>
          <span>元数据管理端</span>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selected]}
          style={{ background: 'transparent', border: 'none' }}
          items={[
            {
              key: 'collections',
              icon: <DatabaseOutlined />,
              label: <Link to="/">数据表</Link>,
            },
            {
              key: 'scripts',
              icon: <CodeOutlined />,
              label: <Link to="/scripts">脚本</Link>,
            },
            {
              key: 'settings',
              icon: <SettingOutlined />,
              label: <Link to="/settings">设置</Link>,
            },
          ]}
        />
        <div style={{ marginTop: 'auto', padding: '8px 10px', fontSize: 12, opacity: 0.55 }}>
          <div>Ctrl+K 命令面板</div>
          <div>仅消费 /api/*</div>
        </div>
      </aside>
      <main className="app-main">
        <div className="app-content">{children}</div>
      </main>
      <CommandPalette />
    </div>
  )
}
