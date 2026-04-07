import { apiClient } from './client'
import type { TimelineResponse } from '../types/post'

export async function fetchTimeline(limit = 20, offset = 0): Promise<TimelineResponse> {
  const res = await apiClient.get<TimelineResponse>('/posts/timeline', {
    params: { limit, offset },
  })
  return res.data
}

export async function likePost(postId: string): Promise<void> {
  await apiClient.post(`/posts/${postId}/like`)
}

export async function unlikePost(postId: string): Promise<void> {
  await apiClient.delete(`/posts/${postId}/like`)
}
