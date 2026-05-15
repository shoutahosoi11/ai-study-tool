import axios from 'axios'

import { getIdToken } from './auth'
import { apiBaseURL } from '../config'

export const apiClient = axios.create({
  baseURL: apiBaseURL,
  timeout: 30_000,
})

apiClient.interceptors.request.use(async (config) => {
  const token = await getIdToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  return config
})
