import axios, { AxiosHeaders } from 'axios'
import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { createWebSession, getIdToken, getStoredCSRFToken } from './auth'

type RetriableRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
}

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30_000,
  withCredentials: true,
})

apiClient.interceptors.request.use(async function (config) {
  const token = await getIdToken()
  const headers = AxiosHeaders.from(config.headers)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  } else {
    headers.delete('Authorization')
  }
  const method = (config.method ?? 'get').toLowerCase()
  if (['post', 'put', 'patch', 'delete'].includes(method)) {
    const csrfToken = getStoredCSRFToken()
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken)
    }
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

    if (error.response?.status === 403 && config && !config._retry) {
      config._retry = true
      try {
        const sessionCreated = await createWebSession(true)
        if (sessionCreated) {
          return apiClient.request(config)
        }
      } catch {
        return Promise.reject(error)
      }
    }

    return Promise.reject(error)
  }
)
