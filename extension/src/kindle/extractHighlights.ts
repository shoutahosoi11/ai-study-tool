import { MAX_IMPORT_HIGHLIGHTS, type ExtractResult, type KindleHighlight } from '../types'

const highlightSelectors = [
  '.kp-notebook-highlight',
  '[id^="kp-notebook-annotated"]',
  '[data-annotation-id]',
  '.kp-notebook-selected-text',
]

const noteSelectors = [
  '.kp-notebook-note',
  '.kp-notebook-note-text',
  '.kp-notebook-note-content',
  '[data-annotation-note]',
]

const asinFallbackSelectors = [
  '[data-asin]',
  '[data-book-asin]',
  '#kp-notebook-annotations',
]

function normalizeText(value: string | null | undefined): string {
  return (value ?? '').replace(/\s+/g, ' ').trim()
}

function firstText(root: ParentNode, selectors: string[]): string {
  for (const selector of selectors) {
    const element = root.querySelector(selector)
    const text = normalizeText(element?.textContent)
    if (text) {
      return text
    }
  }
  return ''
}

function textFromMeta(documentRef: Document, selectors: string[]): string {
  return firstText(documentRef, selectors)
}

export function isKindleNotebookUrl(rawUrl: string): boolean {
  try {
    const url = new URL(rawUrl)
    const host = url.hostname.toLowerCase()
    if (host !== 'read.amazon.co.jp' && host !== 'read.amazon.com') {
      return false
    }
    return url.pathname.toLowerCase().includes('/notebook')
  } catch {
    return false
  }
}

export function extractHighlights(documentRef: Document = document): ExtractResult {
  if (!isKindleNotebookUrl(documentRef.location.href)) {
    throw new Error('Kindle Notebook ページでのみ取り込みできます')
  }

  const bookTitle =
    textFromMeta(documentRef, ['#annotation-scroller h3', '.kp-notebook-searchable', '.kp-notebook-title', 'h1']) ||
    normalizeText(documentRef.title.replace(/Kindle.*$/i, ''))
  const bookAuthor = textFromMeta(documentRef, ['.kp-notebook-author', '.a-color-secondary'])
  const asin = extractAsin(documentRef)
  const nodes = Array.from(documentRef.querySelectorAll(highlightSelectors.join(', ')))
  const seen = new Set<string>()
  const highlights: KindleHighlight[] = []

  // Explicit user imports should remain bounded even if Kindle changes the DOM
  // and exposes unexpectedly many matching nodes. The backend performs its own
  // limits as a second line of defense.
  for (const node of nodes.slice(0, MAX_IMPORT_HIGHLIGHTS)) {
    const container = node.closest('[data-annotation-id], .kp-notebook-row-separator, .a-row') ?? node.parentElement ?? node
    const content = normalizeText(node.textContent)
    if (!content || seen.has(content)) {
      continue
    }
    seen.add(content)

    const note = firstText(container, noteSelectors)
    const metadata = normalizeText(
      firstText(container, ['#annotationHighlightHeader', '.kp-notebook-metadata', '.a-color-secondary']) ||
        container.textContent,
    )
    const locationMatch = metadata.match(/(?:Location|位置|ページ|Page)\s*[:：]?\s*([0-9,\-]+)/i)
    highlights.push({
      asin,
      bookTitle: bookTitle || 'Unknown Kindle Book',
      bookAuthor,
      content,
      note,
      location: locationMatch?.[1] ?? '',
    })
  }

  if (highlights.length === 0) {
    throw new Error(
      'ハイライトが見つかりませんでした。Kindle Notebookの表示形式が変わった可能性があります。対象ページがKindle Notebookか確認してください。',
    )
  }
  return {
    highlights,
    totalFound: nodes.length,
    truncated: nodes.length > highlights.length,
  }
}

function extractAsin(documentRef: Document): string {
  const fromURL = new URL(documentRef.location.href).searchParams.get('asin')
  if (fromURL) {
    return fromURL.trim()
  }

  // Prefer stable data attributes before text parsing; Kindle layouts vary, so
  // only simple B/ASIN-like identifiers are accepted as fallback.
  for (const selector of asinFallbackSelectors) {
    const element = documentRef.querySelector(selector)
    const candidate =
      element?.getAttribute('data-asin') ??
      element?.getAttribute('data-book-asin') ??
      element?.getAttribute('data-book-id') ??
      ''
    const normalized = candidate.trim()
    if (/^[A-Z0-9]{10}$/.test(normalized)) {
      return normalized
    }
  }

  const bodyText = normalizeText(documentRef.body?.textContent)
  const match = bodyText.match(/\b(?:ASIN|asin)\s*[:：]?\s*([A-Z0-9]{10})\b/)
  return match?.[1] ?? ''
}
