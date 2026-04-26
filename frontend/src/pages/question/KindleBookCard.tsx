import { Button } from '../../components/common/Button'
import { theme } from '../../theme'

type Book = {
  id: string
  asin: string
  book_title: string
  book_author: string
  highlight_count: number
}

type Props = {
  book: Book
  stock?: number
  target?: number
  preparing?: number
  isPreparing?: boolean
  isGenerating?: boolean
  isViewingHighlights?: boolean
  onViewHighlights: () => void
  onGenerate: () => void
}

export function KindleBookCard({
  book,
  stock,
  target,
  preparing = 0,
  isPreparing = false,
  isGenerating = false,
  isViewingHighlights = false,
  onViewHighlights,
  onGenerate,
}: Props) {
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
          {book.book_title || book.asin || 'Kindle 本'}
        </p>
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {book.highlight_count > 0 ? `ハイライト ${book.highlight_count} 件保存済み` : 'まだハイライトがありません'}
        </p>
        {typeof stock === 'number' && typeof target === 'number' ? (
          <p style={{ margin: 0, color: isPreparing ? theme.colors.primary : theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            {isPreparing ? `${preparing}問準備中` : `準備済み ${stock} / ${target} 問`}
          </p>
        ) : null}
      </div>

      <div style={{ display: 'flex', gap: theme.spacing.sm }}>
        <div style={{ flex: 1 }}>
          <Button fullWidth onClick={onGenerate} loading={isGenerating} disabled={isGenerating || isViewingHighlights || isPreparing}>
            問題を解く
          </Button>
        </div>
        <div style={{ flex: 1 }}>
          <Button
            variant="outline"
            fullWidth
            onClick={onViewHighlights}
            loading={isViewingHighlights}
            disabled={isViewingHighlights || isGenerating}
          >
            一覧を見る
          </Button>
        </div>
      </div>
    </div>
  )
}
