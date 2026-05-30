import { describe, expect, it, vi } from 'vitest'

import { ExtensionApiClient, errorForStatus } from '../src/api/client'

describe('ExtensionApiClient', () => {
  it('attaches Authorization header for highlight import', async () => {
    const fetchImpl = vi.fn(async (_url: string | URL | Request, init?: RequestInit) => {
      const headers = new Headers(init?.headers)
      expect(headers.get('Authorization')).toBe('Bearer ext_secret')
      return new Response(JSON.stringify({ saved_count: 2, duplicate_count: 1 }), { status: 200 })
    }) as unknown as typeof fetch
    const client = new ExtensionApiClient({ apiBaseUrl: 'https://api.example.com', fetchImpl })

    await expect(
      client.importHighlights('ext_secret', [
        { asin: 'B1', bookTitle: 'Book', bookAuthor: 'Author', content: 'Highlight' },
      ]),
    ).resolves.toMatchObject({ ok: true, savedCount: 2, duplicateCount: 1 })
  })

  it('maps common backend statuses to extension errors', () => {
    expect(errorForStatus(401)).toMatchObject({ code: 'unauthorized' })
    expect(errorForStatus(403)).toMatchObject({ code: 'forbidden' })
    expect(errorForStatus(429)).toMatchObject({ code: 'rate_limited' })
    expect(errorForStatus(503)).toMatchObject({ code: 'server_error' })
  })
})
