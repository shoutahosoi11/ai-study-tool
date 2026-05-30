import { isKindleNotebookUrl } from './kindle/extractHighlights'
import type { RuntimeResponse } from './types'
import { runImport } from './content/importFlow'

const buttonID = 'ai-study-tool-import-button'
const statusID = 'ai-study-tool-import-status'

if (isKindleNotebookUrl(window.location.href)) {
  injectImportButton()
}

function injectImportButton(): void {
  if (document.getElementById(buttonID)) {
    return
  }
  const wrapper = document.createElement('div')
  wrapper.style.position = 'fixed'
  wrapper.style.right = '16px'
  wrapper.style.bottom = '16px'
  wrapper.style.zIndex = '2147483647'
  wrapper.style.fontFamily = 'system-ui, sans-serif'

  const button = document.createElement('button')
  button.id = buttonID
  button.type = 'button'
  button.textContent = 'ai-study-toolへ取り込む'
  button.style.border = '0'
  button.style.borderRadius = '6px'
  button.style.background = '#2563eb'
  button.style.color = '#fff'
  button.style.padding = '10px 12px'
  button.style.boxShadow = '0 4px 14px rgba(0,0,0,0.18)'
  button.style.cursor = 'pointer'

  const status = document.createElement('div')
  status.id = statusID
  status.style.marginTop = '8px'
  status.style.maxWidth = '260px'
  status.style.borderRadius = '6px'
  status.style.background = '#fff'
  status.style.color = '#1f2933'
  status.style.padding = '8px 10px'
  status.style.boxShadow = '0 4px 14px rgba(0,0,0,0.14)'
  status.style.fontSize = '12px'
  status.hidden = true

  button.addEventListener('click', () => {
    void handleImportClick(button, status)
  })

  wrapper.append(button, status)
  document.documentElement.appendChild(wrapper)
}

async function handleImportClick(button: HTMLButtonElement, status: HTMLDivElement): Promise<void> {
  button.disabled = true
  setStatus(status, 'ハイライトを抽出しています...')
  try {
    const response = await runImport(
      document,
      (count, totalFound, truncated) => {
        const truncatedNote = truncated ? `（ページ上の${totalFound}件から先頭${count}件のみ）` : ''
        setStatus(status, `${count}件を送信しています...${truncatedNote}`)
      },
      sendImportMessage,
    )
    if (!response.ok) {
      setStatus(status, response.error)
      return
    }
    const result = response.result
    setStatus(
      status,
      `取り込み完了: 保存 ${result?.savedCount ?? 0} / 重複 ${result?.duplicateCount ?? 0} / スキップ ${result?.skippedCount ?? 0}`,
    )
  } catch (error) {
    setStatus(status, error instanceof Error ? error.message : '取り込みに失敗しました')
  } finally {
    button.disabled = false
  }
}

function setStatus(element: HTMLElement, message: string): void {
  element.hidden = false
  element.textContent = message
}

function sendImportMessage(highlights: unknown): Promise<RuntimeResponse> {
  return new Promise((resolve) => {
    chrome.runtime.sendMessage({ type: 'IMPORT_HIGHLIGHTS', highlights }, (response: RuntimeResponse | undefined) => {
      const lastError = chrome.runtime.lastError
      if (lastError) {
        resolve({ ok: false, error: '拡張機能のservice workerに接続できませんでした' })
        return
      }
      resolve(response ?? { ok: false, error: '取り込み結果を取得できませんでした' })
    })
  })
}
