import type { ApiResponse, ValidationErrorData } from '@/types/metadata'
import { getSettings } from '@/lib/settings'

export class ApiError extends Error {
  code: number
  data: unknown

  constructor(code: number, message: string, data?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.data = data
  }

  get validation(): ValidationErrorData | null {
    if (this.code === 422 && this.data && typeof this.data === 'object') {
      return this.data as ValidationErrorData
    }
    return null
  }
}

function getBaseUrl(): string {
  const settings = getSettings()
  const base = settings.apiBaseUrl?.trim()
  if (base) return base.replace(/\/$/, '')
  // 开发环境走 Vite proxy：相对路径 /api
  return ''
}

function withQuery(path: string, query?: Record<string, unknown>): string {
  if (!query) return path
  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    if (typeof value === 'object') {
      params.set(key, JSON.stringify(value))
    } else {
      params.set(key, String(value))
    }
  })
  const qs = params.toString()
  return qs ? `${path}?${qs}` : path
}

export async function request<T>(
  path: string,
  options: RequestInit & { query?: Record<string, unknown> } = {},
): Promise<T> {
  const { query, headers, ...rest } = options
  const base = getBaseUrl()
  const url = `${base}${withQuery(path, query)}`

  const res = await fetch(url, {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      ...(headers || {}),
    },
  })

  let body: ApiResponse<T>
  try {
    body = (await res.json()) as ApiResponse<T>
  } catch {
    throw new ApiError(res.status || 1, `请求失败：HTTP ${res.status}`)
  }

  if (body.code !== 0) {
    throw new ApiError(body.code, body.message || '请求失败', body.data)
  }

  return body.data
}

export const http = {
  get: <T>(path: string, query?: Record<string, unknown>) =>
    request<T>(path, { method: 'GET', query }),
  post: <T>(path: string, data?: unknown, query?: Record<string, unknown>) =>
    request<T>(path, {
      method: 'POST',
      body: data !== undefined ? JSON.stringify(data) : undefined,
      query,
    }),
  put: <T>(path: string, data?: unknown, query?: Record<string, unknown>) =>
    request<T>(path, {
      method: 'PUT',
      body: data !== undefined ? JSON.stringify(data) : undefined,
      query,
    }),
  delete: <T>(path: string, data?: unknown, query?: Record<string, unknown>) =>
    request<T>(path, {
      method: 'DELETE',
      body: data !== undefined ? JSON.stringify(data) : undefined,
      query,
    }),
}
