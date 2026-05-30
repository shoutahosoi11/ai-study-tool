import { JSDOM } from 'jsdom'
import { describe, expect, it } from 'vitest'

import { extractHighlights, isKindleNotebookUrl } from '../src/kindle/extractHighlights'
import { MAX_IMPORT_HIGHLIGHTS } from '../src/types'

describe('extractHighlights', () => {
  it('extracts book metadata and highlights from Kindle Notebook HTML', () => {
    const dom = new JSDOM(
      `<!doctype html>
      <title>Sample Book - Kindle</title>
      <main>
        <h1 class="kp-notebook-title">Sample Book</h1>
        <div class="kp-notebook-author">Jane Author</div>
        <div class="kp-notebook-row-separator">
          <div id="annotationHighlightHeader">Location 123</div>
          <div class="kp-notebook-highlight"> First highlight text </div>
          <div class="kp-notebook-note">Important note</div>
        </div>
        <div class="kp-notebook-row-separator">
          <div class="kp-notebook-highlight">Second highlight text</div>
        </div>
      </main>`,
      { url: 'https://read.amazon.co.jp/notebook?asin=B000TEST01' },
    )

    const result = extractHighlights(dom.window.document)

    expect(result).toMatchObject({ totalFound: 2, truncated: false })
    expect(result.highlights).toHaveLength(2)
    expect(result.highlights[0]).toMatchObject({
      asin: 'B000TEST01',
      bookTitle: 'Sample Book',
      bookAuthor: 'Jane Author',
      content: 'First highlight text',
      note: 'Important note',
      location: '123',
    })
  })

  it('only accepts Kindle Notebook pages', () => {
    expect(isKindleNotebookUrl('https://read.amazon.co.jp/notebook')).toBe(true)
    expect(isKindleNotebookUrl('https://read.amazon.com/notebook?asin=B1')).toBe(true)
    expect(isKindleNotebookUrl('https://example.com/notebook')).toBe(false)
  })

  it('fails clearly and does not return an empty import payload', () => {
    const dom = new JSDOM('<!doctype html><main><h1>Book</h1></main>', {
      url: 'https://read.amazon.com/notebook?asin=B1',
    })

    expect(() => extractHighlights(dom.window.document)).toThrow(/ハイライトが見つかりませんでした/)
  })

  it('caps one explicit import to the configured item limit', () => {
    const html = Array.from({ length: MAX_IMPORT_HIGHLIGHTS + 20 }, (_, index) => {
      return `<div class="kp-notebook-highlight">Highlight ${index}</div>`
    }).join('')
    const dom = new JSDOM(`<!doctype html><h1>Book</h1>${html}`, {
      url: 'https://read.amazon.com/notebook?asin=B1',
    })

    const result = extractHighlights(dom.window.document)

    expect(result.highlights).toHaveLength(MAX_IMPORT_HIGHLIGHTS)
    expect(result.totalFound).toBe(MAX_IMPORT_HIGHLIGHTS + 20)
    expect(result.truncated).toBe(true)
  })

  it('uses DOM ASIN fallback when the query parameter is missing', () => {
    const dom = new JSDOM(
      '<!doctype html><h1>Book</h1><div data-asin="B012345678"></div><div class="kp-notebook-highlight">Text</div>',
      { url: 'https://read.amazon.com/notebook' },
    )

    expect(extractHighlights(dom.window.document).highlights[0]?.asin).toBe('B012345678')
  })

  it('keeps ASIN empty when no stable identifier is found', () => {
    const dom = new JSDOM('<!doctype html><h1>Book</h1><div class="kp-notebook-highlight">Text</div>', {
      url: 'https://read.amazon.com/notebook',
    })

    expect(extractHighlights(dom.window.document).highlights[0]?.asin).toBe('')
  })
})
