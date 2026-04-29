import axios from 'axios'

type ApiErrorBody = {
  message?: unknown
  error?: {
    message?: unknown
  }
}

export function isApiError(error: unknown) {
  return axios.isAxiosError<ApiErrorBody>(error)
}

export function isApiStatus(error: unknown, status: number) {
  return isApiError(error) && error.response?.status === status
}

export function getApiErrorMessage(error: unknown) {
  if (!isApiError(error)) {
    return ''
  }

  const body = error.response?.data
  if (typeof body?.message === 'string') {
    return body.message.trim()
  }
  if (typeof body?.error?.message === 'string') {
    return body.error.message.trim()
  }

  return ''
}

export function serializeApiDebugError(error: unknown) {
  if (!isApiError(error)) {
    return null
  }

  return {
    kind: 'axios',
    message: error.message,
    code: error.code,
    status: error.response?.status,
    data: error.response?.data,
  }
}
