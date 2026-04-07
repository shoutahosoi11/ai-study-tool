import { apiClient } from './client'
import type { Highlight, ListHighlightsResponse } from '../types/highlight'

export async function createHighlight(req: Partial<Highlight>): Promise<Highlight> {
  const res = await apiClient.post<Highlight>('/highlights', req)
  return res.data
}

export async function listHighlights(page = 1, limit = 20): Promise<ListHighlightsResponse> {
  const res = await apiClient.get<ListHighlightsResponse>('/highlights', {
    params: { page, limit },
  })
  return res.data
}

export async function getHighlight(id: string): Promise<Highlight> {
  const res = await apiClient.get<Highlight>(`/highlights/${id}`)
  return res.data
}

export async function deleteHighlight(id: string): Promise<void> {
  await apiClient.delete(`/highlights/${id}`)
}
