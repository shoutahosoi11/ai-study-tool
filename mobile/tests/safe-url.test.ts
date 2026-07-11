import { safeHttpUrl } from '../src/utils/safe-url'

describe('safeHttpUrl', () => {
  it('accepts http and https URLs', () => {
    expect(safeHttpUrl('https://example.com/page')).toBe('https://example.com/page')
    expect(safeHttpUrl('http://example.com')).toBe('http://example.com/')
  })

  it('rejects javascript: and data: URLs', () => {
    expect(safeHttpUrl('javascript:alert(1)')).toBeUndefined()
    expect(safeHttpUrl('data:text/html,x')).toBeUndefined()
  })

  it('rejects relative URLs', () => {
    expect(safeHttpUrl('/settings')).toBeUndefined()
  })

  it('returns undefined for empty input', () => {
    expect(safeHttpUrl('')).toBeUndefined()
    expect(safeHttpUrl(null)).toBeUndefined()
    expect(safeHttpUrl(undefined)).toBeUndefined()
  })
})
