import { ExtensionApiClient, ExtensionApiError } from './api/client'
import { claimPairing, pollPairingUntilApproved, startPairing } from './auth/pairing'
import { formatTokenExpiry } from './optionsView'
import { clearToken, getSettings, getToken, saveSettings, saveToken } from './storage/tokenStore'
import { API_BASE_URL_PLACEHOLDER, DEFAULT_API_BASE_URL, type PairingState, type StoredSettings, type StoredToken } from './types'

type ViewState = {
  settings: StoredSettings
  token: StoredToken | undefined
  pairing: PairingState | undefined
  status: string
}

const appRoot = requireElement('app')

let state: ViewState = {
  settings: {},
  token: undefined,
  pairing: undefined,
  status: '',
}
let stopPolling: (() => void) | undefined

void initialize()

async function initialize(): Promise<void> {
  state = {
    settings: await getSettings(),
    token: await getToken(),
    pairing: undefined,
    status: '',
  }
  render()
}

function apiBaseUrl(): string {
  return state.settings.apiBaseUrl ?? DEFAULT_API_BASE_URL
}

function render(): void {
  appRoot.replaceChildren()

  const heading = document.createElement('h1')
  heading.textContent = 'AI Study Tool Kindle Import'
  const description = document.createElement('p')
  description.className = 'muted'
  description.textContent = 'Kindle Notebook のハイライトを、ユーザー操作で AI Study Tool に取り込みます。'
  appRoot.append(heading, description)

  appRoot.append(renderConnectionSection())
  appRoot.append(renderPairingSection())
  appRoot.append(renderImportSection())
}

function renderConnectionSection(): HTMLElement {
  const section = document.createElement('section')
  const heading = document.createElement('h2')
  heading.textContent = '接続設定'
  section.appendChild(heading)

  const label = document.createElement('label')
  label.textContent = 'Backend API URL'
  label.htmlFor = 'apiBaseUrl'
  const input = document.createElement('input')
  input.id = 'apiBaseUrl'
  input.value = apiBaseUrl()
  input.placeholder = API_BASE_URL_PLACEHOLDER
  input.required = true
  const saveButton = button('保存', 'secondary', async () => {
    try {
      await saveSettings({ apiBaseUrl: input.value })
      state.settings = await getSettings()
      state.status = 'API URLを保存しました'
    } catch (error) {
      state.status = error instanceof Error ? error.message : 'API URLを保存できませんでした'
    }
    render()
  })

  const connected = document.createElement('p')
  connected.className = 'muted'
  connected.textContent = state.token ? '接続状態: 接続済み' : '接続状態: 未接続。Backend API URLを設定してください。'
  const expires = document.createElement('p')
  expires.className = 'muted'
  expires.textContent = state.token ? `接続期限: ${formatTokenExpiry(state.token.expiresAt)}` : '接続期限: 未接続'

  section.append(label, input, saveButton, connected, expires)
  if (state.status) {
    const status = document.createElement('p')
    status.className = 'status'
    status.textContent = state.status
    section.appendChild(status)
  }
  return section
}

function renderPairingSection(): HTMLElement {
  const section = document.createElement('section')
  const heading = document.createElement('h2')
  heading.textContent = 'Webでログインして接続'
  const note = document.createElement('p')
  note.className = 'muted'
  note.textContent = '接続開始後、Webの /extension/connect で user_code を入力して承認してください。'

  const startButton = button('接続を開始', '', async () => {
    await handleStartPairing()
  })
  section.append(heading, note, startButton)

  if (state.pairing) {
    const code = document.createElement('div')
    code.className = 'code'
    code.textContent = state.pairing.userCode
    const status = document.createElement('p')
    status.className = 'status'
    status.textContent = `Pairing status: ${state.pairing.status}`
    section.append(code, status)
    if (state.pairing.status === 'approved') {
      section.appendChild(
        button('tokenを取得', '', async () => {
          await handleClaimPairing()
        }),
      )
    }
  }
  return section
}

function renderImportSection(): HTMLElement {
  const section = document.createElement('section')
  const heading = document.createElement('h2')
  heading.textContent = '取り込み状態'
  const lastImport = document.createElement('p')
  lastImport.className = 'muted'
  lastImport.textContent = state.settings.lastImportAt
    ? `最終取り込み: ${new Date(state.settings.lastImportAt).toLocaleString()}`
    : '最終取り込み: なし'
  const instruction = document.createElement('p')
  instruction.className = 'muted'
  instruction.textContent = 'Kindle Notebookページ上の「ai-study-toolへ取り込む」ボタンから明示的に実行します。'
  section.append(heading, lastImport, instruction)

  if (state.token) {
    section.appendChild(
      button('revoke / disconnect', 'danger', async () => {
        await handleDisconnect()
      }),
    )
  }
  return section
}

async function handleStartPairing(): Promise<void> {
  stopPolling?.()
  state.status = 'pairingを開始しています'
  render()
  try {
    state.pairing = await startPairing(apiBaseUrl())
    state.status = 'Webでuser_codeを承認してください'
    render()
    stopPolling = pollPairingUntilApproved(apiBaseUrl(), state.pairing.pairingId, (status) => {
      if (!state.pairing) {
        return
      }
      state.pairing = { ...state.pairing, status }
      if (status === 'expired') {
        state.status = '接続コードの有効期限が切れました。もう一度開始してください。'
      }
      render()
      if (status === 'approved') {
        stopPolling?.()
        void handleClaimPairing()
      }
    }, { expiresAt: state.pairing.expiresAt })
  } catch (error) {
    state.status = error instanceof Error ? error.message : 'pairingを開始できませんでした'
    render()
  }
}

async function handleClaimPairing(): Promise<void> {
  if (!state.pairing) {
    return
  }
  try {
    const result = await claimPairing(apiBaseUrl(), state.pairing.pairingId)
    await saveToken(result.token, result.scopes, result.expiresAt)
    state.token = await getToken()
    state.pairing = { ...state.pairing, pairingId: '', status: 'claimed' }
    state.status = '接続が完了しました'
  } catch (error) {
    state.status = error instanceof Error ? error.message : 'tokenを取得できませんでした'
  }
  render()
}

async function handleDisconnect(): Promise<void> {
  const token = state.token?.token
  if (token) {
    try {
      const client = new ExtensionApiClient({ apiBaseUrl: apiBaseUrl() })
      await client.revokeSelf(token)
    } catch (error) {
      if (error instanceof ExtensionApiError && error.code !== 'unauthorized') {
        state.status = 'server revokeに失敗しました。local tokenは削除します。'
      }
    }
  }
  await clearToken()
  state.token = undefined
  if (!state.status) {
    state.status = '切断しました'
  }
  render()
}

function button(label: string, tone: '' | 'secondary' | 'danger', onClick: () => Promise<void>): HTMLButtonElement {
  const element = document.createElement('button')
  element.type = 'button'
  element.textContent = label
  if (tone) {
    element.className = tone
  }
  element.addEventListener('click', () => {
    element.disabled = true
    void onClick().finally(() => {
      element.disabled = false
    })
  })
  return element
}

function requireElement(id: string): HTMLElement {
  const element = document.getElementById(id)
  if (!element) {
    throw new Error(`${id} root not found`)
  }
  return element
}
