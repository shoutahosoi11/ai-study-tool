import { apiClient } from './client'

export type ImportSharedHighlightRequest = {
  content: string
  book_title?: string
  book_author?: string
  source_app?: string
  source_url?: string
}

export type HighlightResponse = {
  id: string
  book_id?: string
  book_title?: string
  book_author?: string
  asin?: string
  content: string
  explanation?: string
  location?: string
  highlighted_at?: string
  source: string
  source_app?: string
  source_url?: string
  created_at: string
}

export type ImportSharedHighlightResponse = {
  saved: boolean
  duplicate: boolean
  highlight?: HighlightResponse
}

export type ListBookHighlightsResponse = {
  highlights: HighlightResponse[]
}

export type ImportKindleHighlightItem = {
  asin?: string
  book_title?: string
  book_author?: string
  content: string
  location?: string
  highlighted_at?: string | null
}

export type ImportKindleHighlightsRequest = {
  highlights: ImportKindleHighlightItem[]
}

export type ImportKindleHighlightsResponse = {
  saved_count: number
  duplicate_count: number
  copy_protected_count: number
  resolved_asin: string
  highlights: HighlightResponse[]
  warning?: string
}

export async function importSharedHighlight(
  payload: ImportSharedHighlightRequest
): Promise<ImportSharedHighlightResponse> {
  const response = await apiClient.post<ImportSharedHighlightResponse>('/highlights/share', payload)
  return response.data
}

export async function importKindleHighlights(
  payload: ImportKindleHighlightsRequest
): Promise<ImportKindleHighlightsResponse> {
  const response = await apiClient.post<ImportKindleHighlightsResponse>('/highlights/import', payload)
  return response.data
}

export async function listBookHighlights(asin: string): Promise<HighlightResponse[]> {
  const response = await apiClient.get<ListBookHighlightsResponse>(`/highlights/books/${encodeURIComponent(asin)}/items`)
  return response.data.highlights ?? []
}

export async function listBookHighlightsByMetadata(bookTitle: string, bookAuthor?: string): Promise<HighlightResponse[]> {
  const response = await apiClient.get<ListBookHighlightsResponse>('/highlights/books/search/items', {
    params: {
      title: bookTitle,
      author: bookAuthor ?? '',
    },
  })
  return response.data.highlights ?? []
}

export async function updateHighlightExplanation(id: string, explanation: string): Promise<HighlightResponse> {
  const response = await apiClient.put<HighlightResponse>(`/highlights/${id}/explanation`, {
    explanation,
  })
  return response.data
}
