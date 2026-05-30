import { DEFAULT_API_BASE_URL } from '../types'

export function normalizeApiBaseUrl(value: string | undefined): string {
  const trimmed = (value ?? '').trim()
  if (!trimmed) {
    return DEFAULT_API_BASE_URL
  }

  let parsed: URL
  try {
    parsed = new URL(trimmed)
  } catch {
    throw new Error('API URL must be a valid http/https URL')
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('API URL must use http or https')
  }
  parsed.pathname = parsed.pathname.replace(/\/+$/, '')
  parsed.search = ''
  parsed.hash = ''
  return parsed.toString().replace(/\/+$/, '')
}

export function apiV1BaseUrl(value: string | undefined): string {
  const base = normalizeApiBaseUrl(value)
  if (!base) {
    throw new Error('Backend API URLを設定してください')
  }
  if (base.endsWith('/api/v1')) {
    return base
  }
  if (base.endsWith('/api')) {
    return `${base}/v1`
  }
  return `${base}/api/v1`
}
