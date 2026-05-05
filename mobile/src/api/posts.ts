import { apiClient } from './client'

export type TimelinePost = {
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

export type TimelineResponse = {
  posts: TimelinePost[]
  limit: number
  offset: number
}

export type CreateQuestionPostQuestionInput = {
  question_id: string
  sort_order: number
  note: string
}

export type CreateQuestionPostInput = {
  body: string
  book_title: string
  question_count: number
  questions: CreateQuestionPostQuestionInput[]
  type: 'question'
}

export type CreatedPost = {
  id: string
  user_id: string
  body?: string
  book_title?: string
  question_count: number
  type: string
  created_at: string
  updated_at: string
}

export type PostQuestion = {
  id: string
  question_type: 'multiple_choice'
  content: string
  options: string[]
  correct_answer: string
  explanation: string
  note?: string
  sort_order?: number
}

export type PostQuestionResponse = {
  questions: PostQuestion[]
}

export type PostComment = {
  id: string
  post_id: string
  user_id: string
  username: string
  display_name: string
  avatar_url?: string
  content: string
  created_at: string
}

export type PostCommentsResponse = {
  comments: PostComment[]
  limit: number
  offset: number
}

export async function fetchTimeline(limit = 20, offset = 0): Promise<TimelineResponse> {
  const response = await apiClient.get<TimelineResponse>('/posts/timeline', {
    params: { limit, offset },
  })
  return response.data
}

export async function createQuestionPost(input: CreateQuestionPostInput): Promise<CreatedPost> {
  const response = await apiClient.post<CreatedPost>('/posts', input)
  return response.data
}

export async function fetchPostQuestions(postID: string): Promise<PostQuestion[]> {
  const response = await apiClient.get<PostQuestionResponse>(`/posts/${postID}/questions`)
  return response.data.questions ?? []
}

export async function listPostComments(postID: string, limit = 20, offset = 0): Promise<PostComment[]> {
  const response = await apiClient.get<PostCommentsResponse>(`/posts/${postID}/comments`, {
    params: { limit, offset },
  })
  return response.data.comments ?? []
}

export async function createPostComment(postID: string, content: string): Promise<PostComment> {
  const response = await apiClient.post<PostComment>(`/posts/${postID}/comments`, { content })
  return response.data
}

export async function likePost(postID: string): Promise<void> {
  await apiClient.post(`/posts/${postID}/like`)
}

export async function unlikePost(postID: string): Promise<void> {
  await apiClient.delete(`/posts/${postID}/like`)
}

export async function repostPost(postID: string): Promise<void> {
  await apiClient.post(`/posts/${postID}/repost`)
}

export async function unrepostPost(postID: string): Promise<void> {
  await apiClient.delete(`/posts/${postID}/repost`)
}
