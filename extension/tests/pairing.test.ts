import { afterEach, describe, expect, it, vi } from 'vitest'

import { pollPairingUntilApproved } from '../src/auth/pairing'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('pairing polling', () => {
  it('stops immediately when the client-side deadline has passed', async () => {
    const statuses: string[] = []
    const fetchImpl = vi.fn()
    vi.stubGlobal('fetch', fetchImpl)

    pollPairingUntilApproved('https://api.example.com', 'pairing-secret', (status) => {
      statuses.push(status)
    }, { expiresAt: '2026-05-28T00:00:00.000Z', now: () => Date.parse('2026-05-28T00:00:01.000Z') })

    expect(statuses).toEqual(['expired'])
    expect(fetchImpl).not.toHaveBeenCalled()
  })

  it('reports approved before the deadline', async () => {
    const statuses: string[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(JSON.stringify({ status: 'approved' }), { status: 200 })),
    )

    pollPairingUntilApproved('https://api.example.com', 'pairing-secret', (status) => {
      statuses.push(status)
    }, { expiresAt: '2026-05-28T00:10:00.000Z', now: () => Date.parse('2026-05-28T00:00:00.000Z') })
    await new Promise((resolve) => globalThis.setTimeout(resolve, 0))
    await new Promise((resolve) => globalThis.setTimeout(resolve, 0))

    expect(statuses).toContain('approved')
  })
})
