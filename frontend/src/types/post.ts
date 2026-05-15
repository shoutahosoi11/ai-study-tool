export interface TimelinePost {
  id: string
  user_id: string
  question_id?: string
  book_id?: string
  field_id?: string
  body?: string
  type: string
  book_title?: string
  question_count: number
  repost_count: number
  like_count: number
  comment_count: number
  created_at: string
  updated_at: string
  score: number
  username: string
  display_name: string
  avatar_url?: string
  field_name?: string
}

export interface TimelineResponse {
  posts: TimelinePost[]
  limit: number
  offset: number
}

export interface CreateQuestionPostQuestionInput {
  question_id: string
  sort_order: number
  note: string
}

export interface CreateQuestionPostInput {
  body: string
  book_title: string
  question_count: number
  questions: CreateQuestionPostQuestionInput[]
  type: "question"
}

export interface CreatedPost {
  id: string
  user_id: string
  body?: string
  book_title?: string
  question_count: number
  type: string
  created_at: string
  updated_at: string
}

export interface PostQuestion {
  id: string
  question_type: "multiple_choice"
  content: string
  options: string[]
  correct_answer: string
  explanation: string
  note: string
  sort_order: number
}

export interface PostQuestionResponse {
  questions: PostQuestion[]
}

export interface PostComment {
  id: string
  post_id: string
  user_id: string
  username: string
  display_name: string
  avatar_url?: string
  content: string
  created_at: string
}

export interface PostCommentsResponse {
  comments: PostComment[]
  limit: number
  offset: number
}
