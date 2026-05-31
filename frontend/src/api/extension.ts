import { apiClient } from './client'

export async function approveExtensionPairing(userCode: string) {
  await apiClient.post('/extension/pairing/approve', {
    user_code: userCode,
  })
}
