import axios, { AxiosHeaders } from 'axios'
import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { getIdToken } from './auth'

type RetriableRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
}

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30_000,
})

apiClient.interceptors.request.use(async function (config) {
  const token = await getIdToken()
  const headers = AxiosHeaders.from(config.headers)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  } else {
    headers.delete('Authorization')
  }
  config.headers = headers
  return config
})

apiClient.interceptors.response.use(
  function (response) {
    return response
  },
  async function (error: AxiosError) {
    const config = error.config as RetriableRequestConfig | undefined
    if (error.response?.status === 401 && config && !config._retry) {
      config._retry = true
      const refreshedToken = await getIdToken(true)
      if (refreshedToken) {
        const headers = AxiosHeaders.from(config.headers)
        headers.set('Authorization', `Bearer ${refreshedToken}`)
        config.headers = headers
        return apiClient.request(config)
      }
    }

    return Promise.reject(error)
  }
)
