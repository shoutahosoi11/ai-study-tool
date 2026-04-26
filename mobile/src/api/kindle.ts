import { apiClient } from './client'

export type KindleBook = {
  asin: string
  book_title: string
  book_author: string
  highlight_count: number
  source: string
}

export type ListKindleBooksResponse = {
  books: KindleBook[]
}

export async function listKindleBooks(): Promise<ListKindleBooksResponse> {
  const response = await apiClient.get<ListKindleBooksResponse>('/highlights/books')
  return response.data
}
