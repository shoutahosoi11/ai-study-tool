import { apiClient } from './client'

export type SignUpBackendUserRequest = {
  username: string
}

export type MeResponse = {
  id: string
  username: string
  display_name: string
  avatar_url?: string
  bio?: string
  university?: string
  faculty?: string
  grade?: number
  country?: string
  plan: string
  default_question_count: number
}

export async function signUpBackendUser(payload: SignUpBackendUserRequest): Promise<MeResponse> {
  const response = await apiClient.post<MeResponse>('/users/signup', payload)
  return response.data
}

export async function getMe(): Promise<MeResponse> {
  const response = await apiClient.get<MeResponse>('/users/me')
  return response.data
}

export async function updateQuestionSettings(defaultQuestionCount: number): Promise<MeResponse> {
  const response = await apiClient.put<MeResponse>('/users/me/question-settings', {
    default_question_count: defaultQuestionCount,
  })
  return response.data
}

// 事前に reauthenticateWithPassword で再認証しておくこと
// （サーバーは5分以内の再ログインを要求する）。
export async function deleteAccount(): Promise<void> {
  await apiClient.delete('/users/me')
}
