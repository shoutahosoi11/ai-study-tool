import axios from 'axios'

type ApiErrorBody = {
  message?: unknown
  error?: unknown
  minVersion?: unknown
  platform?: unknown
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
  if (body?.error && typeof body.error === 'object' && 'message' in body.error) {
    const message = body.error.message
    if (typeof message === 'string') {
      return message.trim()
    }
  }
  if (typeof body?.error === 'string') return body.error.trim()

  return ''
}

export function isAuthenticationRequired(error: unknown) {
  return isApiStatus(error, 401)
}

export function isUpgradeRequired(error: unknown) {
  return isApiStatus(error, 426)
}

export function getUpgradeRequiredInfo(error: unknown) {
  if (!isApiError(error) || error.response?.status !== 426) {
    return null
  }
  const body = error.response?.data
  return {
    minVersion: typeof body?.minVersion === 'string' ? body.minVersion : '',
    platform: typeof body?.platform === 'string' ? body.platform : '',
  }
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
