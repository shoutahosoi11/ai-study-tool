import { describe, expect, it } from 'vitest'

import { formatTokenExpiry } from '../src/optionsView'

describe('options view helpers', () => {
  it('formats known token expiry without throwing', () => {
    expect(formatTokenExpiry('2026-06-01T00:00:00Z')).not.toBe('不明')
  })

  it('handles missing token expiry', () => {
    expect(formatTokenExpiry(undefined)).toBe('不明')
    expect(formatTokenExpiry('not-a-date')).toBe('不明')
  })
})
