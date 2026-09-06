import { http } from './client'
import type {
  CollectionItem,
  CreateCollectionInput,
  CreateFieldInput,
  FieldItem,
  IndexItem,
  ListRecordsParams,
  PageResult,
  RecordData,
  UpdateCollectionInput,
  UpdateFieldInput,
  YaegiScript,
} from '@/types/metadata'

export const metaApi = {
  listCollections: (params?: { keyword?: string; category?: string; page?: number; pageSize?: number }) =>
    http.get<PageResult<CollectionItem>>('/api/meta/collections', params),

  getCollection: (name: string) =>
    http.get<CollectionItem>(`/api/meta/collections/${encodeURIComponent(name)}`),

  createCollection: (input: CreateCollectionInput) =>
    http.post<CollectionItem>('/api/meta/collections', input),

  updateCollection: (name: string, input: UpdateCollectionInput) =>
    http.put<CollectionItem>(`/api/meta/collections/${encodeURIComponent(name)}`, input),

  deleteCollection: (name: string) =>
    http.delete<unknown>(`/api/meta/collections/${encodeURIComponent(name)}`),

  syncCollection: (name: string) =>
    http.post<CollectionItem>(`/api/meta/collections/${encodeURIComponent(name)}/sync`),

  listFields: (collection: string) =>
    http.get<{ list: FieldItem[] }>(
      `/api/meta/collections/${encodeURIComponent(collection)}/fields`,
    ),

  createField: (collection: string, input: CreateFieldInput) =>
    http.post<{ list: FieldItem[] }>(
      `/api/meta/collections/${encodeURIComponent(collection)}/fields`,
      input,
    ),

  updateField: (collection: string, field: string, input: UpdateFieldInput) =>
    http.put<{ list: FieldItem[] }>(
      `/api/meta/collections/${encodeURIComponent(collection)}/fields/${encodeURIComponent(field)}`,
      input,
    ),

  deleteField: (collection: string, field: string) =>
    http.delete<unknown>(
      `/api/meta/collections/${encodeURIComponent(collection)}/fields/${encodeURIComponent(field)}`,
    ),

  listIndexes: (collection: string) =>
    http.get<{ list: IndexItem[] }>(
      `/api/meta/collections/${encodeURIComponent(collection)}/indexes`,
    ),

  createIndex: (collection: string, index: IndexItem) =>
    http.post<{ index: IndexItem }>(
      `/api/meta/collections/${encodeURIComponent(collection)}/indexes`,
      index,
    ),

  deleteIndex: (collection: string, fields: string[]) =>
    http.delete<unknown>(`/api/meta/collections/${encodeURIComponent(collection)}/indexes`, {
      fields,
    }),

  listScripts: (params?: { collection?: string; hook?: string }) =>
    http.get<{ list: YaegiScript[] }>('/api/meta/scripts', params),

  saveScript: (script: YaegiScript) =>
    http.post<YaegiScript>('/api/meta/scripts', script),

  disableScript: (id: number) =>
    http.post<unknown>(`/api/meta/scripts/${id}/disable`),

  deleteScript: (id: number) =>
    http.delete<unknown>(`/api/meta/scripts/${id}`),

  validateScript: (content: string) =>
    http.post<{ valid: boolean; error?: string }>('/api/meta/scripts/validate', { content }),
}

export const dataApi = {
  list: (collection: string, params?: ListRecordsParams) =>
    http.get<PageResult<RecordData>>(`/api/c/${encodeURIComponent(collection)}`, params as Record<string, unknown>),

  get: (collection: string, id: string | number, params?: { fields?: string; except?: string; appends?: string }) =>
    http.get<RecordData>(`/api/c/${encodeURIComponent(collection)}/${id}`, params),

  create: (collection: string, values: RecordData) =>
    http.post<RecordData>(`/api/c/${encodeURIComponent(collection)}`, values),

  update: (collection: string, id: string | number, values: RecordData) =>
    http.put<RecordData>(`/api/c/${encodeURIComponent(collection)}/${id}`, values),

  remove: (collection: string, id: string | number) =>
    http.delete<unknown>(`/api/c/${encodeURIComponent(collection)}/${id}`),

  removeMany: (collection: string, filter: Record<string, unknown>) =>
    http.delete<unknown>(`/api/c/${encodeURIComponent(collection)}`, undefined, { filter }),

  count: (collection: string, filter?: Record<string, unknown>) =>
    http.get<{ count: number }>(`/api/c/${encodeURIComponent(collection)}/count`, filter ? { filter } : undefined),

  associationList: (collection: string, id: string | number, association: string) =>
    http.get<unknown>(`/api/c/${encodeURIComponent(collection)}/${id}/${encodeURIComponent(association)}`),

  associationAdd: (collection: string, id: string | number, association: string, body: unknown) =>
    http.post<unknown>(
      `/api/c/${encodeURIComponent(collection)}/${id}/${encodeURIComponent(association)}`,
      body,
    ),

  associationSet: (collection: string, id: string | number, association: string, body: unknown) =>
    http.put<unknown>(
      `/api/c/${encodeURIComponent(collection)}/${id}/${encodeURIComponent(association)}`,
      body,
    ),

  associationRemove: (collection: string, id: string | number, association: string, body: unknown) =>
    http.delete<unknown>(
      `/api/c/${encodeURIComponent(collection)}/${id}/${encodeURIComponent(association)}`,
      body,
    ),
}
