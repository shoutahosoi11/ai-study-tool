import type { RuntimeMessage } from '../types'

const allowedTypes = new Set(['START_IMPORT', 'IMPORT_HIGHLIGHTS'])
const allowedMessageKeys = new Set(['type', 'highlights'])
const allowedHighlightKeys = new Set(['asin', 'bookTitle', 'bookAuthor', 'content', 'note', 'location', 'page'])

export function isRuntimeMessage(value: unknown): value is RuntimeMessage {
  if (!value || typeof value !== 'object') {
    return false
  }
  const candidate = value as { type?: unknown; highlights?: unknown }
  if (!Object.keys(candidate).every((key) => allowedMessageKeys.has(key))) {
    return false
  }
  if (typeof candidate.type !== 'string' || !allowedTypes.has(candidate.type)) {
    return false
  }
  if (candidate.type === 'START_IMPORT') {
    return true
  }
  if (!Array.isArray(candidate.highlights)) {
    return false
  }
  return candidate.highlights.every((item) => {
    if (!item || typeof item !== 'object') {
      return false
    }
    const highlight = item as { bookTitle?: unknown; content?: unknown }
    if (!Object.keys(highlight).every((key) => allowedHighlightKeys.has(key))) {
      return false
    }
    return typeof highlight.bookTitle === 'string' && typeof highlight.content === 'string'
  })
}
