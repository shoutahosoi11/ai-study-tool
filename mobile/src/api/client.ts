import axios from 'axios'

import { getIdToken } from './auth'
import { apiBaseURL } from '../config'

export const apiClient = axios.create({
  baseURL: apiBaseURL,
})

apiClient.interceptors.request.use(async (config) => {
  const token = await getIdToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  return config
})
