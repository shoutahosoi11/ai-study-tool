import { theme } from '../../theme'
import type { ImportHighlightsResponse } from '../../types/kindle'
import type { SyncState } from '../../hooks/useKindleSync'

type Book = {
  id: string
  asin: string
  book_title: string
  book_author: string
  highlight_count: number
  notebook_url?: string
}

type Props = {
  book: Book
  syncState: SyncState
  syncStatusText?: string
  generateStatusText?: string
  syncResult?: ImportHighlightsResponse
  syncError?: string
  syncAvailable?: boolean
  generateEnabled?: boolean
  isGenerating?: boolean
  onViewHighlights: () => void
  onGenerate: () => void
}

function isAllCopyProtected(err?: string) {
  if (!err) return false
  return err === 'ALL_COPY_PROTECTED' || err.indexOf('コピー制限') !== -1
}

function getSyncErrorMessage(err?: string) {
  if (!err) return ''
  if (err === 'NOT_AUTHENTICATED') return 'アプリにログインしてください'
  if (err === 'BOOK_IDENTIFIER_MISSING') return 'この本は Amazon ページ上で ASIN を取得できなかったため同期できません'
  if (err === 'BOOK_TARGET_UNAVAILABLE') return 'この本を開く Kindle ノートURLを作れませんでした。もう一度一覧を取り直してください'
  if (err === 'NOT_LOGGED_IN') return 'Amazon にログインしてください'
  if (err === 'BOOK_NOT_FOUND') return '対象の本が Kindle ノートページで見つかりませんでした'
  if (err === 'BOOK_OPEN_FAILED') return '対象の本を開けませんでした'
  if (err === 'NOTEBOOK_NOT_REACHED') return 'Amazon のノートページに移動できませんでした'
  if (err === 'UNEXPECTED_PAGE') return 'Amazon のノートページに移動できませんでした'
  if (err === 'NOTEBOOK_TIMEOUT') return 'Kindle ノートページの応答が止まりました。Amazon タブを開き直して再試行してください'
  if (err === 'TAB_CREATE_FAILED') return 'Kindle 取得用タブを開けませんでした'
  if (err.startsWith('TAB_CREATE_FAILED:')) return `Kindle 取得用タブを開けませんでした (${err.slice('TAB_CREATE_FAILED:'.length).trim()})`
  if (err === 'SYNC_TIMEOUT') return '同期がタイムアウトしました。拡張を再読み込みして再試行してください'
  if (err === 'NO_HIGHLIGHTS') return 'この本のハイライトが見つかりませんでした'
  return err
}

function getAvailableHighlightCount(book: Book, syncResult?: ImportHighlightsResponse) {
  if (!syncResult) return book.highlight_count
  return Math.max(book.highlight_count, syncResult.saved_count + syncResult.duplicate_count)
}

function getDisplayBookTitle(book: Book, syncResult?: ImportHighlightsResponse) {
  if (book.book_title) return book.book_title

  const syncedTitle = syncResult?.highlights?.find(function (highlight) {
    return Boolean(highlight.book_title)
  })?.book_title
  if (syncedTitle) return syncedTitle

  return book.asin || 'Kindle 本'
}

export function KindleBookCard({
  book,
  syncState,
  syncStatusText,
  generateStatusText,
  syncResult,
  syncError,
  syncAvailable = true,
  generateEnabled = true,
  isGenerating = false,
  onViewHighlights,
  onGenerate,
}: Props) {
  const isSyncing = syncState === 'syncing'
  const isBusy = isSyncing || isGenerating
  const allCopyProtected = isAllCopyProtected(syncError)
  const availableHighlightCount = getAvailableHighlightCount(book, syncResult)
  const canViewHighlights = syncAvailable && !isBusy
  const canGenerate = syncAvailable && generateEnabled && !isBusy && (!allCopyProtected || availableHighlightCount > 0)

  return (
    <div
      style={{
        border: `1px solid ${theme.colors.border}`,
        borderRadius: theme.radius.md,
        background: theme.colors.background,
        padding: theme.spacing.md,
        display: 'flex',
        flexDirection: 'column',
        gap: theme.spacing.sm,
      }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.xs }}>
        <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>
          {getDisplayBookTitle(book, syncResult)}
        </p>
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {availableHighlightCount > 0 ? `ハイライト ${availableHighlightCount} 件保存済み` : '未同期'}
        </p>
      </div>

      {syncResult && (
        <p style={{ margin: 0, color: theme.colors.success, fontSize: theme.fontSize.sm }}>
          保存 {syncResult.saved_count} 件 / 重複 {syncResult.duplicate_count} 件 / コピー制限 {syncResult.copy_protected_count} 件
        </p>
      )}
      {syncResult?.warning && (
        <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
          {syncResult.warning}
        </p>
      )}
      {allCopyProtected && (
        <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
          コピー制限によりハイライトを取得できませんでした
        </p>
      )}
      {syncError && !allCopyProtected && (
        <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
          同期に失敗しました: {getSyncErrorMessage(syncError)}
        </p>
      )}
      {!syncAvailable && (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          この本は Amazon ページ上で ASIN を取得できなかったため、まだ同期できません
        </p>
      )}
      {isSyncing && syncStatusText && (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {syncStatusText}
        </p>
      )}
      {isGenerating && generateStatusText && (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {generateStatusText}
        </p>
      )}

      <div style={{ display: 'flex', gap: theme.spacing.sm }}>
        <button
          type="button"
          onClick={onViewHighlights}
          disabled={!canViewHighlights}
          style={{
            flex: 1,
            border: 'none',
            borderRadius: theme.radius.sm,
            padding: `${theme.spacing.sm} ${theme.spacing.md}`,
            background: canViewHighlights ? theme.colors.primary : theme.colors.border,
            color: canViewHighlights ? theme.colors.background : theme.colors.secondary,
            cursor: canViewHighlights ? 'pointer' : 'not-allowed',
            fontSize: theme.fontSize.sm,
            fontWeight: 700,
          }}
        >
          {isSyncing ? '同期中...' : isGenerating ? '作成中...' : '一覧を見る'}
        </button>
        <button
          type="button"
          onClick={onGenerate}
          disabled={!canGenerate}
          style={{
            flex: 1,
            border: 'none',
            borderRadius: theme.radius.sm,
            padding: `${theme.spacing.sm} ${theme.spacing.md}`,
            background: canGenerate ? theme.colors.success : theme.colors.secondary,
            color: theme.colors.background,
            cursor: canGenerate ? 'pointer' : 'not-allowed',
            fontSize: theme.fontSize.sm,
            fontWeight: 700,
          }}
        >
          {isGenerating ? '問題を作成中...' : '問題を作る'}
        </button>
      </div>
    </div>
  )
}
