export interface Highlight {
  id: string
  book_id?: string
  book_title?: string
  book_author?: string
  asin?: string
  content: string
  explanation?: string
  location?: string
  highlighted_at?: string
  source: 'kindle'
  created_at: string
}

export interface ListBookHighlightsResponse {
  highlights: Highlight[]
}
