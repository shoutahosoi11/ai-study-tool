import { describe, expect, it, vi } from 'vitest'

import { ExtensionApiError } from '../src/api/client'
import { handleMessage } from '../src/backgroundHandler'

describe('background message handler', () => {
  it('rejects unknown message types and invalid payloads', async () => {
    const deps = {
      importHighlights: vi.fn(),
      clearToken: vi.fn(),
    }

    await expect(handleMessage({ type: 'GET_TOKEN' }, deps)).resolves.toMatchObject({ ok: false })
    await expect(handleMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: 'bad' }, deps)).resolves.toMatchObject({ ok: false })
    expect(deps.importHighlights).not.toHaveBeenCalled()
  })

  it('prompts pairing when no token is connected', async () => {
    const deps = {
      importHighlights: vi.fn(async () => {
        throw new ExtensionApiError('missing_token', '拡張機能をAI Study Toolに接続してください')
      }),
      clearToken: vi.fn(),
    }

    await expect(
      handleMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ bookTitle: 'B', content: 'C' }] }, deps),
    ).resolves.toMatchObject({ ok: false, code: 'missing_token' })
  })

  it('clears the token on 401 unauthorized', async () => {
    const deps = {
      importHighlights: vi.fn(async () => {
        throw new ExtensionApiError('unauthorized', '接続が失効しました。再接続してください。')
      }),
      clearToken: vi.fn(async () => undefined),
    }

    await expect(
      handleMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ bookTitle: 'B', content: 'C' }] }, deps),
    ).resolves.toMatchObject({ ok: false, code: 'unauthorized' })
    expect(deps.clearToken).toHaveBeenCalledTimes(1)
  })

  it('maps 403, 429, and 5xx style errors without clearing the token', async () => {
    for (const [code, message] of [
      ['forbidden', '拡張機能の権限が不足しています。'],
      ['rate_limited', '取り込み回数が多すぎます。時間を置いて再試行してください。'],
      ['server_error', 'サーバー側の一時的なエラーです。'],
    ] as const) {
      const deps = {
        importHighlights: vi.fn(async () => {
          throw new ExtensionApiError(code, message)
        }),
        clearToken: vi.fn(),
      }

      await expect(
        handleMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ bookTitle: 'B', content: 'C' }] }, deps),
      ).resolves.toMatchObject({ ok: false, code, error: message })
      expect(deps.clearToken).not.toHaveBeenCalled()
    }
  })

  it('imports valid highlight payloads', async () => {
    const deps = {
      importHighlights: vi.fn(async () => ({
        ok: true as const,
        savedCount: 1,
        duplicateCount: 0,
        skippedCount: 0,
        queuedCount: 0,
      })),
      clearToken: vi.fn(),
    }

    await expect(
      handleMessage({ type: 'IMPORT_HIGHLIGHTS', highlights: [{ bookTitle: 'B', content: 'C' }] }, deps),
    ).resolves.toMatchObject({ ok: true, result: { savedCount: 1 } })
    expect(deps.importHighlights).toHaveBeenCalledWith([{ bookTitle: 'B', content: 'C' }])
  })
})
