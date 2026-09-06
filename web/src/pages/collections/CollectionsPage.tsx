import { App, Button, Empty, Input, Select, Skeleton, Space, Tag, Dropdown } from 'antd'
import {
  DeleteOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { metaApi } from '@/api'
import { ApiError } from '@/api/client'
import { COLLECTION_TYPES } from '@/lib/fieldTypes'

export function CollectionsPage() {
  const navigate = useNavigate()
  const { message, modal } = App.useApp()
  const qc = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [type, setType] = useState<string | undefined>()
  const [category, setCategory] = useState<string | undefined>()

  const { data, isLoading, isError, error, refetch, isFetching } = useQuery({
    queryKey: ['collections', keyword, category],
    queryFn: () =>
      metaApi.listCollections({
        keyword: keyword || undefined,
        category: category || undefined,
        pageSize: 200,
      }),
  })

  const syncMut = useMutation({
    mutationFn: (name: string) => metaApi.syncCollection(name),
    onSuccess: () => message.success('同步成功'),
    onError: (e: Error) => message.error(e.message),
  })

  const deleteMut = useMutation({
    mutationFn: (name: string) => metaApi.deleteCollection(name),
    onSuccess: () => {
      message.success('已删除')
      qc.invalidateQueries({ queryKey: ['collections'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const list = useMemo(() => {
    const items = data?.list || []
    if (!type) return items
    return items.filter((x) => x.type === type)
  }, [data, type])

  const categories = useMemo(() => {
    const set = new Set<string>()
    ;(data?.list || []).forEach((c) => (c.categories || []).forEach((x) => set.add(x)))
    return [...set]
  }, [data])

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>数据表</h1>
          <p>管理 Collection 结构，并进入工作室维护字段与记录</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/collections/new')}>
            新建数据表
          </Button>
        </Space>
      </div>

      <div className="panel" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="搜索显示名或标识"
            style={{ width: 260 }}
            onSearch={setKeyword}
            onChange={(e) => {
              if (!e.target.value) setKeyword('')
            }}
          />
          <Select
            allowClear
            placeholder="表类型"
            style={{ width: 160 }}
            value={type}
            onChange={setType}
            options={COLLECTION_TYPES.map((t) => ({ value: t.value, label: t.label }))}
          />
          <Select
            allowClear
            placeholder="分类"
            style={{ width: 160 }}
            value={category}
            onChange={setCategory}
            options={categories.map((c) => ({ value: c, label: c }))}
          />
          <span className="muted">共 {list.length} 张表</span>
        </Space>
      </div>

      {isLoading ? (
        <Skeleton active paragraph={{ rows: 8 }} />
      ) : isError ? (
        <div className="panel">
          <Empty
            description={(error as ApiError)?.message || '加载失败，请检查 API 地址'}
          >
            <Button onClick={() => refetch()}>重试</Button>
            <Button type="link" onClick={() => navigate('/settings')}>
              去设置
            </Button>
          </Empty>
        </div>
      ) : list.length === 0 ? (
        <div className="panel">
          <Empty description="还没有数据表">
            <Button type="primary" onClick={() => navigate('/collections/new')}>
              新建第一张表
            </Button>
          </Empty>
        </div>
      ) : (
        <div className="collection-grid">
          {list.map((item) => (
            <div
              key={item.name}
              className="collection-card"
              onClick={() => navigate(`/collections/${item.name}`)}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
                <div>
                  <h3>{item.displayName || item.name}</h3>
                  <div className="meta">{item.name}</div>
                </div>
                <Dropdown
                  menu={{
                    items: [
                      {
                        key: 'open',
                        label: '打开',
                        onClick: () => navigate(`/collections/${item.name}`),
                      },
                      {
                        key: 'data',
                        label: '数据',
                        onClick: () => navigate(`/collections/${item.name}/records`),
                      },
                      {
                        key: 'sync',
                        icon: <SyncOutlined />,
                        label: '同步结构',
                        onClick: () => syncMut.mutate(item.name),
                      },
                      { type: 'divider' },
                      {
                        key: 'delete',
                        danger: true,
                        icon: <DeleteOutlined />,
                        label: '删除',
                        onClick: () => {
                          modal.confirm({
                            title: `删除数据表「${item.displayName || item.name}」？`,
                            content:
                              '将删除元数据与真实数据表及其中数据，此操作不可恢复。',
                            okText: '删除',
                            okButtonProps: { danger: true },
                            onOk: () => deleteMut.mutateAsync(item.name),
                          })
                        },
                      },
                    ],
                  }}
                  trigger={['click']}
                >
                  <Button
                    type="text"
                    size="small"
                    icon={<MoreOutlined />}
                    onClick={(e) => e.stopPropagation()}
                  />
                </Dropdown>
              </div>
              <div style={{ marginTop: 12, display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                <Tag color="blue">
                  {COLLECTION_TYPES.find((t) => t.value === item.type)?.label || item.type}
                </Tag>
                <Tag>{item.fieldCount ?? item.fields?.length ?? 0} 字段</Tag>
                {(item.categories || []).map((c) => (
                  <Tag key={c}>{c}</Tag>
                ))}
              </div>
              {item.description ? (
                <p className="muted" style={{ margin: '10px 0 0', fontSize: 13 }}>
                  {item.description}
                </p>
              ) : null}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
