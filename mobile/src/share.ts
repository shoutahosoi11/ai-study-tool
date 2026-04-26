import type { ShareIntent } from 'expo-share-intent'

export type ShareDraft = {
  bookTitle: string
  bookAuthor: string
  content: string
  sourceApp: string
  sourceURL: string
}

export function emptyShareDraft(): ShareDraft {
  return {
    bookTitle: '',
    bookAuthor: '',
    content: '',
    sourceApp: '',
    sourceURL: '',
  }
}

export function draftFromShareIntent(shareIntent: ShareIntent): ShareDraft {
  const text = normalizeText(shareIntent.text)
  const url = normalizeText(shareIntent.webUrl)
  const sourceApp = detectSourceApp(text, url)
  const title = sanitizeMetaTitle(normalizeText(shareIntent.meta?.title), sourceApp)
  const parsedText = parseSharedText(text, url, title, sourceApp)

  return {
    bookTitle: parsedText.bookTitle,
    bookAuthor: parsedText.bookAuthor,
    content: parsedText.content,
    sourceApp,
    sourceURL: url,
  }
}

export function mergeShareDraft(current: ShareDraft, incoming: ShareDraft): ShareDraft {
  return {
    bookTitle: incoming.bookTitle || current.bookTitle,
    bookAuthor: incoming.bookAuthor || current.bookAuthor,
    content: incoming.content || current.content,
    sourceApp: incoming.sourceApp || current.sourceApp,
    sourceURL: incoming.sourceURL || current.sourceURL,
  }
}

export function buildInitialShareDraft(incoming: ShareDraft): ShareDraft {
  if (isShareDraftEmpty(incoming)) {
    return emptyShareDraft()
  }

  return { ...incoming }
}

export function buildShareIntentSignature(shareIntent: ShareIntent): string {
  const parts = [
    normalizeText(shareIntent.text),
    normalizeText(shareIntent.webUrl),
    normalizeText(shareIntent.meta?.title),
    shareIntent.type ?? '',
    String(shareIntent.files?.length ?? 0),
  ]

  return parts.join('|')
}

export function createFallbackUsername(email: string | null | undefined): string {
  const local = (email ?? '').split('@')[0] ?? ''
  const normalized = local.toLowerCase().replace(/[^a-z0-9_]/g, '_').replace(/^_+|_+$/g, '')
  const base = normalized || 'reader'
  return `${base}`.slice(0, 50).padEnd(3, 'x')
}

export function isShareDraftEmpty(draft: ShareDraft): boolean {
  return !draft.bookTitle && !draft.bookAuthor && !draft.content && !draft.sourceApp && !draft.sourceURL
}

function detectSourceApp(text: string, url: string): string {
  const haystack = `${text} ${url}`.toLowerCase()
  if (
    haystack.includes('amazon.') ||
    haystack.includes('kindle') ||
    haystack.includes('a.co/') ||
    haystack.includes('amzn.to/')
  ) {
    return 'kindle'
  }
  if (url) {
    try {
      const host = new URL(url).hostname.replace(/^www\./, '')
      if (host === 'a.co' || host === 'amzn.to' || host.endsWith('.amazon.com')) {
        return 'kindle'
      }
      return host.split('.')[0] ?? 'share_sheet'
    } catch {
      return 'share_sheet'
    }
  }

  return 'share_sheet'
}

function normalizeText(value: string | null | undefined): string {
  return (value ?? '').trim()
}

function parseSharedText(text: string, url: string, title: string, sourceApp: string): ShareDraft {
  const cleanedText = stripSharedURLLines(text, url)
  if (sourceApp !== 'kindle') {
    return {
      bookTitle: title,
      bookAuthor: '',
      content: cleanedText,
      sourceApp,
      sourceURL: url,
    }
  }

  const parsedKindle = parseKindleShareText(cleanedText, title)
  return {
    bookTitle: parsedKindle.bookTitle,
    bookAuthor: parsedKindle.bookAuthor,
    content: parsedKindle.content,
    sourceApp,
    sourceURL: url,
  }
}

function parseKindleShareText(
  text: string,
  knownTitle: string
): Pick<ShareDraft, 'bookTitle' | 'bookAuthor' | 'content'> {
  const lines = splitPreservingEmptyLines(text)
  const inlineCitation = extractTrailingInlineKindleCitation(lines)

  if (inlineCitation) {
    const nextLines = [...lines]
    nextLines[inlineCitation.index] = ''

    return {
      bookTitle: inlineCitation.bookTitle || knownTitle,
      bookAuthor: inlineCitation.bookAuthor,
      content: collapseEmptyLines(nextLines),
    }
  }

  const titleIndex = findTrailingTitleLine(lines, knownTitle)

  if (titleIndex >= 0) {
    const authorIndex = findNextNonEmptyLine(lines, titleIndex + 1)
    const nextLines = [...lines]
    const bookTitle = lines[titleIndex] ?? knownTitle
    let bookAuthor = ''

    nextLines[titleIndex] = ''
    if (authorIndex >= 0 && looksLikeAuthorCandidate(lines[authorIndex] ?? '')) {
      bookAuthor = normalizeAuthorLine(lines[authorIndex] ?? '')
      nextLines[authorIndex] = ''
    }

    return {
      bookTitle,
      bookAuthor,
      content: collapseEmptyLines(nextLines),
    }
  }

  const metadataIndexes = findTrailingMetadataIndexes(lines)
  if (metadataIndexes) {
    const nextLines = [...lines]
    const [bookTitleIndex, bookAuthorIndex] = metadataIndexes
    const bookTitle = (lines[bookTitleIndex] ?? knownTitle).trim()
    const bookAuthor = normalizeAuthorLine(lines[bookAuthorIndex] ?? '')

    nextLines[bookTitleIndex] = ''
    nextLines[bookAuthorIndex] = ''

    return {
      bookTitle,
      bookAuthor,
      content: collapseEmptyLines(nextLines),
    }
  }

  return {
    bookTitle: knownTitle,
    bookAuthor: '',
    content: collapseEmptyLines(lines),
  }
}

function extractTrailingInlineKindleCitation(
  lines: string[]
): { index: number; bookTitle: string; bookAuthor: string } | null {
  for (let index = lines.length - 1; index >= 0; index -= 1) {
    const line = lines[index] ?? ''
    if (!line) {
      continue
    }

    const parsed = parseInlineKindleCitationLine(line)
    if (parsed) {
      return {
        index,
        ...parsed,
      }
    }

    break
  }

  return null
}

function stripSharedURLLines(text: string, url: string): string {
  const normalizedURL = normalizeURL(url)
  const keptLines = splitPreservingEmptyLines(text).filter((line) => {
    const trimmed = line.trim()
    if (!trimmed) {
      return true
    }
    if (!looksLikeURL(trimmed)) {
      return true
    }

    return normalizedURL ? normalizeURL(trimmed) !== normalizedURL : false
  })

  return collapseEmptyLines(keptLines)
}

function sanitizeMetaTitle(title: string, sourceApp: string): string {
  if (!title) {
    return ''
  }

  const normalized = normalizeLoose(title)
  if (!normalized) {
    return ''
  }

  const genericTitles = new Set(['kindle', 'amazonkindle', 'share', '共有'])
  if (genericTitles.has(normalized)) {
    return ''
  }

  if (sourceApp === 'kindle' && normalized === 'amazon') {
    return ''
  }

  return title
}

function splitPreservingEmptyLines(text: string): string[] {
  return text
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.trim())
}

function collapseEmptyLines(lines: string[]): string {
  const joined = lines.join('\n')
  return joined.replace(/\n{3,}/g, '\n\n').trim()
}

function findTrailingTitleLine(lines: string[], knownTitle: string): number {
  if (!knownTitle) {
    return -1
  }

  const normalizedKnownTitle = normalizeLoose(knownTitle)
  for (let index = lines.length - 1; index >= 0; index -= 1) {
    const line = lines[index] ?? ''
    if (!line) {
      continue
    }
    if (normalizeLoose(line) !== normalizedKnownTitle) {
      continue
    }

    const trailingNonEmptyCount = lines.slice(index + 1).filter(Boolean).length
    if (trailingNonEmptyCount <= 1) {
      return index
    }
  }

  return -1
}

function findTrailingMetadataIndexes(lines: string[]): [number, number] | null {
  const nonEmptyIndexes = lines
    .map((line, index) => ({ line, index }))
    .filter((entry) => Boolean(entry.line))
    .map((entry) => entry.index)

  if (nonEmptyIndexes.length < 3) {
    return null
  }

  const bookAuthorIndex = nonEmptyIndexes[nonEmptyIndexes.length - 1] ?? -1
  const bookTitleIndex = nonEmptyIndexes[nonEmptyIndexes.length - 2] ?? -1
  if (bookAuthorIndex < 0 || bookTitleIndex < 0) {
    return null
  }

  if (!looksLikeAuthorCandidate(lines[bookAuthorIndex] ?? '')) {
    return null
  }

  if (!looksLikeTitleCandidate(lines[bookTitleIndex] ?? '')) {
    return null
  }

  const separatorSlice = lines.slice(bookTitleIndex === 0 ? 0 : bookTitleIndex - 1, bookTitleIndex)
  const hasBlankSeparator = separatorSlice.some((line) => !line)
  if (!hasBlankSeparator) {
    return null
  }

  return [bookTitleIndex, bookAuthorIndex]
}

function findNextNonEmptyLine(lines: string[], startIndex: number): number {
  for (let index = startIndex; index < lines.length; index += 1) {
    if (lines[index]) {
      return index
    }
  }

  return -1
}

function looksLikeAuthorCandidate(line: string): boolean {
  const normalized = line.trim()
  if (!normalized || looksLikeURL(normalized)) {
    return false
  }

  if (/^(by|著者|author)\b[:\s-]*/i.test(normalized)) {
    return true
  }

  return normalized.length <= 60 && !/[。.!?]"?$/.test(normalized) && !looksLikeQuotedExcerpt(normalized)
}

function looksLikeTitleCandidate(line: string): boolean {
  const normalized = line.trim()
  if (!normalized || looksLikeURL(normalized)) {
    return false
  }

  return normalized.length <= 120 && !looksLikeQuotedExcerpt(normalized)
}

function normalizeAuthorLine(line: string): string {
  return line
    .replace(/^(by|著者|author)\b[:\s-]*/i, '')
    .replace(/著$/, '')
    .trim()
}

function looksLikeQuotedExcerpt(line: string): boolean {
  return /["“”「」『』]/.test(line)
}

function looksLikeURL(line: string): boolean {
  return /^https?:\/\//i.test(line)
}

function normalizeURL(value: string): string {
  if (!value) {
    return ''
  }

  try {
    return new URL(value).toString()
  } catch {
    return value.trim()
  }
}

function normalizeLoose(value: string): string {
  return value
    .toLowerCase()
    .replace(/\s+/g, '')
    .replace(/[「」『』"“”'’`~!@#$%^&*()_+=\-[\]{}|\\:;<>,.?/]/g, '')
}

function parseInlineKindleCitationLine(line: string): { bookTitle: string; bookAuthor: string } | null {
  const normalized = line.trim()
  if (!normalized) {
    return null
  }

  const match = normalized.match(/^[—\-]\s*[『「](.+?)[』」]\s*(.+)$/)
  if (!match) {
    return null
  }

  const bookTitle = (match[1] ?? '').trim()
  const bookAuthor = normalizeAuthorLine((match[2] ?? '').trim())
  if (!bookTitle) {
    return null
  }

  return {
    bookTitle,
    bookAuthor,
  }
}
