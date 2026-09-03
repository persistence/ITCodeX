# Markdown（Vditor 编辑器）

> 来源：NocoBase 文档整理
> 整理时间：2026-09-02

## 介绍

在 NocoBase 中，**Markdown（Vditor）** 属于多媒体字段类型，使用 Vditor 编辑器来编辑和渲染 Markdown 内容。

相比基础的 Markdown 字段，Vditor 版本提供更强大的编辑体验：支持所见即所得、即时渲染、分屏预览三种模式，支持上传图片、录音、表情、代码块高亮、列表、引用、表格等丰富的 Markdown 语法，体验更接近主流 Markdown 编辑器。

适合需要良好 Markdown 编辑体验的内容创作场景。

## 适用场景

Markdown（Vditor）适合这些业务场景：

* 知识库文章、博客正文、帮助文档
* 产品说明、操作手册、技术文档
* 处理方案、排障记录、变更记录
* 需要图片上传、代码高亮、丰富格式的 Markdown 内容

## 创建配置

在数据表的「Configure fields」页面中，点击「Add field」，在多媒体类型中选择「Markdown(Vditor)」可以创建该字段。

| 配置 | 说明 |
|------|------|
| Field interface | 字段的界面类型。Markdown(Vditor) 对应 markdownVditor，使用 Vditor 编辑器组件。 |
| Field display name | 字段在界面中显示的名称，比如「文章正文」「帮助内容」「处理方案」。建议使用业务人员能直接理解的名称。 |
| Field name | 字段标识名称，用于 API、权限、工作流等内部引用。创建后通常不再修改。 |
| Field type | 字段在数据层的类型。Vditor Markdown 通常使用 text 类型保存内容。 |
| Default value | 默认值。新增记录时，如果用户没有填写，可以自动带出默认值。 |
| Upload settings | 图片上传配置，可以配置图片存储位置、大小限制。 |
| Toolbar | 编辑器工具栏配置，可以选择显示哪些功能按钮。 |
| Validation rules | 校验规则。可以限制最小长度、最大长度或是否必填。 |
| Description | 字段说明。适合写编辑规范、内容要求。 |

> **注意**：Vditor 编辑器支持图片上传，需要先配置好存储引擎才能正常上传图片。

## 字段特性

| 特性 | 说明 |
|------|------|
| 默认 Field interface | markdownVditor |
| 默认 Field type | text |
| 编辑器模式 | 支持所见即所得、即时渲染、分屏预览三种模式切换 |
| 支持功能 | Markdown 语法、图片上传、录音、表情、代码高亮、表格、列表、引用等 |
| 页面组件 | 编辑模式使用 Vditor 编辑器，阅读模式按 Markdown 渲染 |
| 筛选 | 支持文本类筛选，比如包含、为空、不为空 |
| 排序 | 通常不用于排序 |

## 编辑配置

创建后，点击字段右侧的「Edit」可以编辑 Markdown(Vditor) 字段配置。

| 配置 | 允许编辑 | 说明 |
|------|----------|------|
| Field display name | 是 | 修改字段在界面中的显示名称。 |
| Field name | 否 | 字段标识名称创建后通常不能修改。 |
| Upload settings | 是 | 调整图片上传配置。 |
| Toolbar | 是 | 调整工具栏显示的功能按钮。 |
| Default value | 是 | 调整新增记录时的默认值。 |
| Validation rules | 是 | 调整字段校验规则。 |
| Description | 是 | 补充字段说明。 |

## 删除字段

点击字段右侧的「Delete」可以删除 Markdown(Vditor) 字段。

> ⚠️ **警告**：删除字段会删除所有已保存的 Markdown 内容，删除前确认内容已备份。

## 页面配置使用

Markdown(Vditor) 适合内容创作场景，编辑体验比基础 Markdown 更好。

| 场景 | 用途 |
|------|------|
| 表单区块 | 使用 Vditor 编辑器编辑内容，支持三种编辑模式。 |
| 详情区块 | 按 Markdown 格式渲染展示内容，支持代码高亮。 |
| 内容管理 | 知识库、文章、文档类内容的正文编辑。 |
| 图片插入 | 直接在编辑器中粘贴或上传图片。 |

## 与基础 Markdown 的区别

| 对比项 | 基础 Markdown | Markdown(Vditor) |
|--------|---------------|------------------|
| 编辑器 | 简易文本编辑 | 功能完整的 Vditor 编辑器 |
| 编辑模式 | 纯文本编辑 | 所见即所得/即时渲染/分屏预览 |
| 图片上传 | 需要手动插入链接 | 支持直接拖拽/粘贴上传 |
| 功能丰富度 | 基础语法 | 支持表情、录音、代码块、大纲等 |
| 加载体积 | 轻量 | 相对较大，功能更全 |

## 相关链接

* Markdown — 基础 Markdown 字段
* 富文本 — 所见即所得富文本编辑
