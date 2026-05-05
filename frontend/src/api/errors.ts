import axios from 'axios'

type ApiErrorBody = {
  message?: unknown
  error?: {
    message?: unknown
  }
}

export function getApiErrorMessage(error: unknown) {
  if (!axios.isAxiosError<ApiErrorBody>(error)) {
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
