import { apiClient } from './client'

export type CheckoutSessionResponse = {
  url: string
}

export async function createCheckoutSession(): Promise<CheckoutSessionResponse> {
  const response = await apiClient.post<CheckoutSessionResponse>('/checkout/session', {})
  return response.data
}
