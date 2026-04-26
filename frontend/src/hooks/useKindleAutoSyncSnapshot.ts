import { useEffect, useState } from 'react'
import {
  KINDLE_AUTO_SYNC_EVENT,
  readKindleAutoSyncSnapshot,
  type KindleAutoSyncSnapshot,
} from '../lib/kindleAutoSync'

export function useKindleAutoSyncSnapshot() {
  const [snapshot, setSnapshot] = useState<KindleAutoSyncSnapshot | null>(() => readKindleAutoSyncSnapshot())

  useEffect(function () {
    function handleUpdate(event: Event) {
      if (event instanceof CustomEvent && event.detail && typeof event.detail === 'object') {
        setSnapshot(event.detail as KindleAutoSyncSnapshot)
        return
      }

      setSnapshot(readKindleAutoSyncSnapshot())
    }

    window.addEventListener(KINDLE_AUTO_SYNC_EVENT, handleUpdate)
    return function () {
      window.removeEventListener(KINDLE_AUTO_SYNC_EVENT, handleUpdate)
    }
  }, [])

  return snapshot
}

