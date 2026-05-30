import { JSDOM } from 'jsdom'
import { describe, expect, it, vi } from 'vitest'

import { runImport } from '../src/content/importFlow'

describe('import flow', () => {
  it('does not call the service worker when no highlights are extracted', async () => {
    const dom = new JSDOM('<!doctype html><main><h1>Book</h1></main>', {
      url: 'https://read.amazon.com/notebook?asin=B1',
    })
    const sender = vi.fn(async () => ({ ok: true as const }))

    await expect(runImport(dom.window.document, vi.fn(), sender)).rejects.toThrow(/ハイライトが見つかりませんでした/)
    expect(sender).not.toHaveBeenCalled()
  })

  it('sends a validated non-empty payload to the service worker', async () => {
    const dom = new JSDOM('<!doctype html><h1>Book</h1><div class="kp-notebook-highlight">Text</div>', {
      url: 'https://read.amazon.com/notebook?asin=B1',
    })
    const beforeSend = vi.fn()
    const sender = vi.fn(async () => ({ ok: true as const }))

    await expect(runImport(dom.window.document, beforeSend, sender)).resolves.toMatchObject({ ok: true })
    expect(beforeSend).toHaveBeenCalledWith(1, 1, false)
    expect(sender).toHaveBeenCalledWith([expect.objectContaining({ content: 'Text' })])
  })

  it('reports truncation before sending a capped import', async () => {
    const html = Array.from({ length: 105 }, (_, index) => {
      return `<div class="kp-notebook-highlight">Highlight ${index}</div>`
    }).join('')
    const dom = new JSDOM(`<!doctype html><h1>Book</h1>${html}`, {
      url: 'https://read.amazon.com/notebook?asin=B1',
    })
    const beforeSend = vi.fn()
    const sender = vi.fn(async () => ({ ok: true as const }))

    await runImport(dom.window.document, beforeSend, sender)

    expect(beforeSend).toHaveBeenCalledWith(100, 105, true)
  })
})
