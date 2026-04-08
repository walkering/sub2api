/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey, PaginatedResponse } from '@/types'

export interface APIKeyListFilters {
  search?: string
  status?: string
  group_id?: number
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: APIKeyListFilters,
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<ApiKey>> {
  const { data } = await apiClient.get<PaginatedResponse<ApiKey>>('/admin/api-keys', {
    params: {
      page,
      page_size: pageSize,
      search: filters?.search,
      status: filters?.status,
      group_id: filters?.group_id
    },
    signal: options?.signal
  })
  return data
}

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId
  })
  return data
}

export const apiKeysAPI = {
  list,
  updateApiKeyGroup
}

export default apiKeysAPI
