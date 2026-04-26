import AsyncStorage from '@react-native-async-storage/async-storage'

import type { HighlightResponse } from './api/highlights'

const STORAGE_PREFIX = 'mobile.saved-highlights'
const MAX_ITEMS = 50

export async function loadSavedHighlights(userKey: string): Promise<HighlightResponse[]> {
  if (!userKey.trim()) {
    return []
  }

  const raw = await AsyncStorage.getItem(buildStorageKey(userKey))
  if (!raw) {
    return []
  }

  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) {
      return []
    }

    return parsed.filter(isHighlightResponse)
  } catch {
    return []
  }
}

export async function saveSavedHighlights(userKey: string, items: HighlightResponse[]): Promise<void> {
  if (!userKey.trim()) {
    return
  }

  await AsyncStorage.setItem(buildStorageKey(userKey), JSON.stringify(items.slice(0, MAX_ITEMS)))
}

export function prependSavedHighlight(
  items: HighlightResponse[],
  highlight: HighlightResponse
): HighlightResponse[] {
  const deduped = items.filter((item) => item.id !== highlight.id)
  return [highlight, ...deduped].slice(0, MAX_ITEMS)
}

function buildStorageKey(userKey: string): string {
  return `${STORAGE_PREFIX}:${userKey}`
}

function isHighlightResponse(value: unknown): value is HighlightResponse {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  return 'id' in value && 'content' in value && 'source' in value && 'created_at' in value
}
