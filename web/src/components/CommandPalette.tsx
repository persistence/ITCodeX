import { useEffect, useMemo, useState } from 'react'
import { Modal, Input, List, Empty } from 'antd'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { metaApi } from '@/api'

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState('')
  const navigate = useNavigate()

  const { data } = useQuery({
    queryKey: ['collections', 'cmd'],
    queryFn: () => metaApi.listCollections({ pageSize: 200 }),
    enabled: open,
  })

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen(true)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const actions = useMemo(() => {
    const base = [
      { key: 'new', title: '新建数据表', subtitle: '创建 Collection', run: () => navigate('/collections/new') },
      { key: 'scripts', title: '脚本管理', subtitle: 'Yaegi 钩子与自定义 API', run: () => navigate('/scripts') },
      { key: 'settings', title: '设置', subtitle: 'API 地址与主题', run: () => navigate('/settings') },
      { key: 'home', title: '数据表广场', subtitle: '返回首页', run: () => navigate('/') },
    ]
    const cols = (data?.list || []).map((c) => ({
      key: `c-${c.name}`,
      title: c.displayName || c.name,
      subtitle: `${c.name} · ${c.type}`,
      run: () => navigate(`/collections/${c.name}`),
    }))
    const all = [...base, ...cols]
    const keyword = q.trim().toLowerCase()
    if (!keyword) return all
    return all.filter(
      (x) => x.title.toLowerCase().includes(keyword) || x.subtitle.toLowerCase().includes(keyword),
    )
  }, [data, navigate, q])

  return (
    <Modal
      open={open}
      onCancel={() => {
        setOpen(false)
        setQ('')
      }}
      footer={null}
      title="命令面板"
      destroyOnClose
      width={560}
    >
      <Input
        autoFocus
        placeholder="搜索数据表或动作…"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        onPressEnter={() => {
          if (actions[0]) {
            actions[0].run()
            setOpen(false)
            setQ('')
          }
        }}
        style={{ marginBottom: 12 }}
      />
      {actions.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无匹配项" />
      ) : (
        <List
          size="small"
          dataSource={actions.slice(0, 12)}
          renderItem={(item) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => {
                item.run()
                setOpen(false)
                setQ('')
              }}
            >
              <List.Item.Meta title={item.title} description={item.subtitle} />
            </List.Item>
          )}
        />
      )}
    </Modal>
  )
}
