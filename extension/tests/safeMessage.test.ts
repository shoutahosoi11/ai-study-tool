import { describe, expect, it } from 'vitest'

import { isRuntimeMessage } from '../src/utils/safeMessage'
import { apiV1BaseUrl, normalizeApiBaseUrl } from '../src/utils/url'

describe('safe messages and URLs', () => {
  it('allows only known runtime message types', () => {
    expect(isRuntimeMessage({ type: 'START_IMPORT' })).toBe(true)
    expect(isRuntimeMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ bookTitle: 'B', content: 'C' }] })).toBe(true)
    expect(isRuntimeMessage({ type: 'GET_TOKEN' })).toBe(false)
    expect(isRuntimeMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ content: 'C' }] })).toBe(false)
    expect(isRuntimeMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: 'not-array' })).toBe(false)
    expect(isRuntimeMessage({ type: 'IMPORT_HIGHLIGHTS', token: 'ext_secret', highlights: [] })).toBe(false)
    expect(isRuntimeMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ bookTitle: 'B', content: 'C', token: 'ext_secret' }] })).toBe(false)
  })

  it('accepts only http and https API URLs', () => {
    expect(normalizeApiBaseUrl('https://api.example.com/')).toBe('https://api.example.com')
    expect(normalizeApiBaseUrl('http://localhost:8080')).toBe('http://localhost:8080')
    expect(apiV1BaseUrl('https://api.example.com')).toBe('https://api.example.com/api/v1')
    expect(() => apiV1BaseUrl('')).toThrow(/API URL/)
    expect(() => normalizeApiBaseUrl('javascript:alert(1)')).toThrow(/http/)
    expect(() => normalizeApiBaseUrl('data:text/plain,hi')).toThrow(/http/)
    expect(() => normalizeApiBaseUrl('file:///tmp/api')).toThrow(/http/)
  })
})
