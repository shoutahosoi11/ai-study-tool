export interface Highlight {
  id: string
  book_id?: string
  book_title?: string
  book_author?: string
  asin?: string
  content: string
  location?: string
  highlighted_at?: string
  source: 'kindle' | 'manual'
  created_at: string
}

export interface ListHighlightsResponse {
  highlights: Highlight[]
  total: number
  page: number
  limit: number
}
