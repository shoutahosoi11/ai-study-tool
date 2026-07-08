import type { ShareIntent } from 'expo-share-intent'
import {
  buildInitialShareDraft,
  buildShareIntentSignature,
  createFallbackUsername,
  draftFromShareIntent,
  emptyShareDraft,
  isShareDraftEmpty,
  mergeShareDraft,
} from '../src/share'

function shareIntent(overrides: Partial<ShareIntent>): ShareIntent {
  return {
    text: null,
    webUrl: null,
    files: null,
    type: null,
    meta: undefined,
    ...overrides,
  } as ShareIntent
}

describe('draftFromShareIntent', () => {
  it('parses a Kindle share with inline citation into title/author/content', () => {
    const draft = draftFromShareIntent(
      shareIntent({
        text: 'ハイライトの本文です。\n\n— 『サンプル書籍』山田太郎 著',
        webUrl: 'https://www.amazon.co.jp/dp/B00EXAMPLE',
      }),
    )

    expect(draft.sourceApp).toBe('kindle')
    expect(draft.bookTitle).toContain('サンプル書籍')
    expect(draft.bookAuthor).toContain('山田太郎')
    expect(draft.content).toBe('ハイライトの本文です。')
    expect(draft.sourceURL).toBe('https://www.amazon.co.jp/dp/B00EXAMPLE')
  })

  it('treats non-Kindle shares as generic content with meta title', () => {
    const draft = draftFromShareIntent(
      shareIntent({
        text: '記事の引用テキスト',
        webUrl: 'https://blog.example.com/post',
        meta: { title: '記事タイトル' },
      }),
    )

    expect(draft.sourceApp).not.toBe('kindle')
    expect(draft.content).toBe('記事の引用テキスト')
    expect(draft.bookTitle).toBe('記事タイトル')
  })
})

describe('mergeShareDraft', () => {
  it('prefers incoming values but keeps current for blanks', () => {
    const current = {
      ...emptyShareDraft(),
      bookTitle: '既存タイトル',
      content: '既存本文',
    }
    const incoming = { ...emptyShareDraft(), content: '新しい本文' }

    const merged = mergeShareDraft(current, incoming)

    expect(merged.content).toBe('新しい本文')
    expect(merged.bookTitle).toBe('既存タイトル')
  })
})

describe('buildInitialShareDraft / isShareDraftEmpty', () => {
  it('returns empty draft when incoming is empty', () => {
    expect(buildInitialShareDraft(emptyShareDraft())).toEqual(emptyShareDraft())
    expect(isShareDraftEmpty(emptyShareDraft())).toBe(true)
  })

  it('copies non-empty incoming draft', () => {
    const incoming = { ...emptyShareDraft(), content: 'x' }
    expect(buildInitialShareDraft(incoming)).toEqual(incoming)
    expect(isShareDraftEmpty(incoming)).toBe(false)
  })
})

describe('buildShareIntentSignature', () => {
  it('changes when text changes and is stable otherwise', () => {
    const a = shareIntent({ text: 'a' })
    expect(buildShareIntentSignature(a)).toBe(buildShareIntentSignature(shareIntent({ text: 'a' })))
    expect(buildShareIntentSignature(a)).not.toBe(buildShareIntentSignature(shareIntent({ text: 'b' })))
  })
})

describe('createFallbackUsername', () => {
  it('derives a normalized username from email local part', () => {
    expect(createFallbackUsername('Taro.Yamada+tag@example.com')).toBe('taro_yamada_tag')
  })

  it('falls back to reader for empty or symbol-only locals', () => {
    expect(createFallbackUsername(null)).toBe('reader')
    expect(createFallbackUsername('***@example.com')).toBe('reader')
  })

  it('pads very short locals to 3 chars', () => {
    expect(createFallbackUsername('ab@example.com')).toBe('abx')
  })
})
