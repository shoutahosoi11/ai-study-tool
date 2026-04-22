import { apiClient } from './client'
import type { Highlight, ListBookHighlightsResponse } from '../types/highlight'

export async function listBookHighlights(asin: string): Promise<ListBookHighlightsResponse> {
  const res = await apiClient.get<ListBookHighlightsResponse>(`/highlights/books/${encodeURIComponent(asin)}/items`)
  return res.data
}

export async function listBookHighlightsByMetadata(bookTitle: string, bookAuthor?: string): Promise<ListBookHighlightsResponse> {
  const res = await apiClient.get<ListBookHighlightsResponse>('/highlights/books/search/items', {
    params: {
      title: bookTitle,
      author: bookAuthor ?? '',
    },
  })
  return res.data
}

export async function updateHighlightExplanation(id: string, explanation: string): Promise<Highlight> {
  const res = await apiClient.put<Highlight>(`/highlights/${id}/explanation`, {
    explanation,
  })
  return res.data
}
