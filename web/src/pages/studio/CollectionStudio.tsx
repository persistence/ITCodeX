import { App, Button, Skeleton, Space, Tabs, Tag, Typography } from 'antd'
import { ArrowLeftOutlined, SyncOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Navigate, Route, Routes, useLocation, useNavigate, useParams } from 'react-router-dom'
import { metaApi } from '@/api'
import { COLLECTION_TYPES } from '@/lib/fieldTypes'
import { OverviewTab } from './tabs/OverviewTab'
import { FieldsTab } from './tabs/FieldsTab'
import { RecordsTab } from './tabs/RecordsTab'
import { IndexesTab } from './tabs/IndexesTab'
import { ScriptsTab } from './tabs/ScriptsTab'
import { AdvancedTab } from './tabs/AdvancedTab'

export function CollectionStudio() {
  const { name = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const { message } = App.useApp()
  const qc = useQueryClient()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['collection', name],
    queryFn: () => metaApi.getCollection(name),
    enabled: Boolean(name),
  })

  const syncMut = useMutation({
    mutationFn: () => metaApi.syncCollection(name),
    onSuccess: () => {
      message.success('同步成功')
      qc.invalidateQueries({ queryKey: ['collection', name] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const tabKey = location.pathname.includes('/fields')
    ? 'fields'
    : location.pathname.includes('/records')
      ? 'records'
      : location.pathname.includes('/indexes')
        ? 'indexes'
        : location.pathname.includes('/scripts')
          ? 'scripts'
          : location.pathname.includes('/advanced')
            ? 'advanced'
            : 'overview'

  if (!name) return <Navigate to="/" replace />

  if (isLoading) return <Skeleton active paragraph={{ rows: 10 }} />

  if (isError || !data) {
    return (
      <div className="panel">
        <Typography.Title level={4}>表不存在或已删除</Typography.Title>
        <p className="muted">{(error as Error)?.message}</p>
        <Button type="primary" onClick={() => navigate('/')}>
          返回广场
        </Button>
      </div>
    )
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <Space size={8} style={{ marginBottom: 6 }}>
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/')} />
            <h1 style={{ margin: 0 }}>{data.displayName || data.name}</h1>
            <Tag color="blue">
              {COLLECTION_TYPES.find((t) => t.value === data.type)?.label || data.type}
            </Tag>
            <Tag>{data.fieldCount ?? data.fields?.length ?? 0} 字段</Tag>
          </Space>
          <p className="mono" style={{ marginLeft: 40 }}>
            {data.name}
            {data.description ? ` · ${data.description}` : ''}
          </p>
        </div>
        <Space>
          <Button icon={<SyncOutlined />} loading={syncMut.isPending} onClick={() => syncMut.mutate()}>
            同步结构
          </Button>
        </Space>
      </div>

      <div className="panel studio-tabs">
        <Tabs
          activeKey={tabKey}
          onChange={(key) => {
            const map: Record<string, string> = {
              overview: `/collections/${name}`,
              fields: `/collections/${name}/fields`,
              records: `/collections/${name}/records`,
              indexes: `/collections/${name}/indexes`,
              scripts: `/collections/${name}/scripts`,
              advanced: `/collections/${name}/advanced`,
            }
            navigate(map[key] || `/collections/${name}`)
          }}
          items={[
            { key: 'overview', label: '概览' },
            { key: 'fields', label: '结构' },
            { key: 'records', label: '数据' },
            { key: 'indexes', label: '索引' },
            { key: 'scripts', label: '脚本' },
            { key: 'advanced', label: '高级' },
          ]}
        />

        <Routes>
          <Route index element={<OverviewTab collection={data} />} />
          <Route path="fields" element={<FieldsTab collection={data} />} />
          <Route path="records" element={<RecordsTab collection={data} />} />
          <Route path="indexes" element={<IndexesTab collection={data} />} />
          <Route path="scripts" element={<ScriptsTab collectionName={data.name} />} />
          <Route path="advanced" element={<AdvancedTab collection={data} />} />
          <Route path="*" element={<Navigate to={`/collections/${name}`} replace />} />
        </Routes>
      </div>
    </div>
  )
}
