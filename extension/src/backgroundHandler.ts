import { ExtensionApiError } from './api/client'
import type { ImportErrorCode, ImportResult, KindleHighlight, RuntimeResponse } from './types'
import { isRuntimeMessage } from './utils/safeMessage'

export type BackgroundHandlerDeps = {
  importHighlights: (highlights: KindleHighlight[]) => Promise<ImportResult>
  clearToken: () => Promise<void>
}

export async function handleMessage(message: unknown, deps: BackgroundHandlerDeps): Promise<RuntimeResponse> {
  if (!isRuntimeMessage(message)) {
    return { ok: false, error: 'Unsupported message' }
  }

  if (message.type === 'START_IMPORT') {
    return { ok: true }
  }

  try {
    const result = await deps.importHighlights(message.highlights)
    return { ok: true, result }
  } catch (error: unknown) {
    const mapped = mapImportError(error)
    if (mapped.code === 'unauthorized') {
      await deps.clearToken()
    }
    return { ok: false, error: mapped.message, code: mapped.code }
  }
}

export function mapImportError(error: unknown): { code: ImportErrorCode; message: string } {
  if (error instanceof ExtensionApiError) {
    return { code: error.code, message: error.message }
  }
  if (typeof error === 'object' && error !== null && 'code' in error && 'message' in error) {
    const candidate = error as { code: ImportErrorCode; message: string }
    return { code: candidate.code, message: candidate.message }
  }
  return { code: 'network_error', message: '取り込みに失敗しました' }
}
