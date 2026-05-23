import { apiClient } from './client'

export type TokenBalance = {
  available_tokens: number
  free_used_today: number
  free_limit: number
  ad_views_today: number
  ad_views_limit: number
  plan: string
}

export async function fetchTokenBalance(): Promise<TokenBalance> {
  const response = await apiClient.get<TokenBalance>('/tokens/balance')
  return response.data
}
