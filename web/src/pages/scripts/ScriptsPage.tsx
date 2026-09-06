import { ScriptsTab } from '@/pages/studio/tabs/ScriptsTab'

export function ScriptsPage() {
  return (
    <div>
      <div className="page-header">
        <div>
          <h1>脚本</h1>
          <p>管理 Yaegi 钩子与自定义 API（语法校验、启用禁用、热加载由后端完成）</p>
        </div>
      </div>
      <div className="panel">
        <ScriptsTab />
      </div>
    </div>
  )
}
