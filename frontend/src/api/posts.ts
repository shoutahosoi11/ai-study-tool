import { apiClient } from './client'
import type {
  CreateQuestionPostInput,
  CreatedPost,
  PostComment,
  PostCommentsResponse,
  PostQuestion,
  PostQuestionResponse,
  TimelineResponse,
} from '../types/post'

export async function fetchTimeline(limit = 20, offset = 0): Promise<TimelineResponse> {
  const res = await apiClient.get<TimelineResponse>('/posts/timeline', {
    params: { limit, offset },
  })
  return res.data
}

export async function createQuestionPost(input: CreateQuestionPostInput): Promise<CreatedPost> {
  const res = await apiClient.post<CreatedPost>('/posts', input)
  return res.data
}

export async function fetchPostQuestions(postId: string): Promise<PostQuestion[]> {
  const res = await apiClient.get<PostQuestionResponse>(`/posts/${postId}/questions`)
  return res.data.questions ?? []
}

export async function listPostComments(postId: string, limit = 20, offset = 0): Promise<PostComment[]> {
  const res = await apiClient.get<PostCommentsResponse>(`/posts/${postId}/comments`, {
    params: { limit, offset },
  })
  return res.data.comments ?? []
}

export async function createPostComment(postId: string, content: string): Promise<PostComment> {
  const res = await apiClient.post<PostComment>(`/posts/${postId}/comments`, { content })
  return res.data
}

export async function likePost(postId: string): Promise<void> {
  await apiClient.post(`/posts/${postId}/like`)
}

export async function unlikePost(postId: string): Promise<void> {
  await apiClient.delete(`/posts/${postId}/like`)
}

export async function repostPost(postId: string): Promise<void> {
  await apiClient.post(`/posts/${postId}/repost`)
}

export async function unrepostPost(postId: string): Promise<void> {
  await apiClient.delete(`/posts/${postId}/repost`)
}
