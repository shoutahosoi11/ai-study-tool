import { ExtensionApiClient } from '../api/client'
import type { PairingState } from '../types'

const pollingIntervalMs = 3000
const defaultPairingTimeoutMs = 10 * 60 * 1000

export type PairingPollOptions = {
  expiresAt?: string
  timeoutMs?: number
  now?: () => number
}

export async function startPairing(apiBaseUrl: string): Promise<PairingState> {
  const client = new ExtensionApiClient({ apiBaseUrl })
  const result = await client.startPairing()
  return {
    pairingId: result.pairing_id,
    userCode: result.user_code,
    expiresAt: result.expires_at,
    status: 'pending',
  }
}

export function pollPairingUntilApproved(
  apiBaseUrl: string,
  pairingId: string,
  onStatus: (status: PairingState['status']) => void,
  options: PairingPollOptions = {},
): () => void {
  const client = new ExtensionApiClient({ apiBaseUrl })
  let stopped = false
  let timer: number | undefined
  const now = options.now ?? Date.now
  const deadlineMs = parseDeadline(options.expiresAt, now(), options.timeoutMs ?? defaultPairingTimeoutMs)

  async function tick(): Promise<void> {
    if (stopped) {
      return
    }
    if (now() >= deadlineMs) {
      onStatus('expired')
      stopped = true
      return
    }
    try {
      const result = await client.pairingStatus(pairingId)
      if (result.status === 'approved') {
        onStatus('approved')
        return
      }
      if (result.status === 'used') {
        onStatus('claimed')
        return
      }
      onStatus('pending')
    } catch {
      onStatus('error')
    }

    timer = globalThis.setTimeout(() => {
      void tick()
    }, pollingIntervalMs) as unknown as number
  }

  void tick()
  return () => {
    stopped = true
    if (timer !== undefined) {
      globalThis.clearTimeout(timer)
    }
  }
}

function parseDeadline(expiresAt: string | undefined, startMs: number, fallbackTimeoutMs: number): number {
  if (expiresAt) {
    const parsed = Date.parse(expiresAt)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return startMs + fallbackTimeoutMs
}

export async function claimPairing(apiBaseUrl: string, pairingId: string): Promise<{ token: string; scopes: string[]; expiresAt?: string }> {
  const client = new ExtensionApiClient({ apiBaseUrl })
  const result = await client.claimPairing(pairingId)
  if (!result.token || !result.token.startsWith('ext_')) {
    throw new Error('pairing token was not returned')
  }
  const claimed: { token: string; scopes: string[]; expiresAt?: string } = {
    token: result.token,
    scopes: result.scopes ?? [],
  }
  if (result.expires_at) {
    claimed.expiresAt = result.expires_at
  }
  return claimed
}
