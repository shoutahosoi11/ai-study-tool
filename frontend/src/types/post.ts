export interface TimelinePost {
  id: string
  user_id: string
  question_id?: string
  note_id?: string
  book_id?: string
  field_id?: string
  type: string
  repost_count: number
  like_count: number
  comment_count: number
  created_at: string
  updated_at: string
  score: number
  username: string
  display_name: string
  avatar_url?: string
  book_title?: string
  field_name?: string
}

export interface TimelineResponse {
  posts: TimelinePost[]
  limit: number
  offset: number
}
