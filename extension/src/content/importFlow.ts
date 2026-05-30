import { extractHighlights } from '../kindle/extractHighlights'
import type { RuntimeResponse } from '../types'

export type ImportSender = (highlights: unknown) => Promise<RuntimeResponse>

export async function runImport(
  documentRef: Document,
  onBeforeSend: (count: number, totalFound: number, truncated: boolean) => void,
  sender: ImportSender,
): Promise<RuntimeResponse> {
  const result = extractHighlights(documentRef)
  onBeforeSend(result.highlights.length, result.totalFound, result.truncated)
  return sender(result.highlights)
}
