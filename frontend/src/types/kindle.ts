import type { Highlight } from './highlight'

export interface KindleBook {
  asin: string
  book_title: string
  book_author: string
  highlight_count: number
  source: string
}

export interface ListKindleBooksResponse {
  books: KindleBook[]
}

export interface ImportHighlightsResponse {
  saved_count: number
  duplicate_count: number
  copy_protected_count: number
  resolved_asin: string
  highlights: Highlight[]
  warning?: string
}

export interface ExtensionKindleBook {
  id: string
  asin: string
  book_title: string
  book_author: string
  notebook_url?: string
}
