import { useEffect, useState } from 'react'
import { Button } from '../../components/common/Button'
import { theme } from '../../theme'
import type { Highlight } from '../../types/highlight'

type Props = {
  bookTitle: string
  highlights: Highlight[]
  loading: boolean
  error: string
  savingHighlightId?: string
  onClose: () => void
  onSaveExplanation: (highlightId: string, explanation: string) => Promise<void>
}

export function KindleHighlightListModal({
  bookTitle,
  highlights,
  loading,
  error,
  savingHighlightId,
  onClose,
  onSaveExplanation,
}: Props) {
  const [drafts, setDrafts] = useState<Record<string, string>>({})

  useEffect(
    function () {
      const nextDrafts: Record<string, string> = {}
      highlights.forEach(function (highlight) {
        nextDrafts[highlight.id] = highlight.explanation ?? ''
      })
      setDrafts(nextDrafts)
    },
    [highlights]
  )

  async function handleSave(highlightId: string) {
    await onSaveExplanation(highlightId, drafts[highlightId] ?? '')
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: theme.spacing.md,
        zIndex: 220,
      }}
      onClick={function (e) {
        if (e.target === e.currentTarget) {
          onClose()
        }
      }}
    >
      <div
        style={{
          background: theme.colors.background,
          borderRadius: theme.radius.md,
          padding: theme.spacing.lg,
          width: '100%',
          maxWidth: '720px',
          maxHeight: '85vh',
          overflowY: 'auto',
          display: 'flex',
          flexDirection: 'column',
          gap: theme.spacing.md,
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: theme.spacing.md }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.xs }}>
            <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>Kindle ハイライト一覧</p>
            <p style={{ margin: 0, fontSize: theme.fontSize.base, fontWeight: 700 }}>{bookTitle || 'Kindle 本'}</p>
          </div>
          <Button variant="ghost" onClick={onClose}>
            閉じる
          </Button>
        </div>

        {loading ? (
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>同期後のハイライト一覧を読み込み中...</p>
        ) : error ? (
          <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{error}</p>
        ) : highlights.length === 0 ? (
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>この本のハイライトはまだありません</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.md }}>
            {highlights.map(function (highlight) {
              const isSaving = savingHighlightId === highlight.id
              return (
                <div
                  key={highlight.id}
                  style={{
                    border: `1px solid ${theme.colors.border}`,
                    borderRadius: theme.radius.md,
                    background: theme.colors.backgroundAlt,
                    padding: theme.spacing.md,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: theme.spacing.sm,
                  }}
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.xs }}>
                    <p style={{ margin: 0, fontSize: theme.fontSize.base, lineHeight: 1.6 }}>{highlight.content}</p>
                    {highlight.location && (
                      <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
                        {highlight.location}
                      </p>
                    )}
                  </div>

                  <textarea
                    value={drafts[highlight.id] ?? ''}
                    onChange={function (e) {
                      const nextValue = e.target.value
                      setDrafts(function (prev) {
                        return {
                          ...prev,
                          [highlight.id]: nextValue,
                        }
                      })
                    }}
                    rows={4}
                    placeholder="このハイライトの解説を書けます"
                    style={{
                      width: '100%',
                      resize: 'vertical',
                      padding: theme.spacing.sm,
                      border: `1px solid ${theme.colors.border}`,
                      borderRadius: theme.radius.sm,
                      fontSize: theme.fontSize.sm,
                      background: theme.colors.background,
                    }}
                  />

                  <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <Button
                      onClick={function () {
                        void handleSave(highlight.id)
                      }}
                      loading={isSaving}
                      disabled={isSaving}
                    >
                      解説を保存
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
