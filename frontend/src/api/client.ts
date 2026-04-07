import axios from 'axios'
import { getIdToken } from './auth'

export const apiClient = axios.create({
  baseURL: '/api',
})

apiClient.interceptors.request.use(async function (config) {
  const token = await getIdToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  function (response) {
    return response
  },
  function (error) {
    return Promise.reject(error)
  }
)
