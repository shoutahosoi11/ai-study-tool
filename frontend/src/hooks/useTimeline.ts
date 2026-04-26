import { useState, useEffect } from 'react'
import { fetchTimeline } from '../api/posts'
import type { TimelinePost } from '../types/post'

export function useTimeline() {
  const [posts, setPosts] = useState<TimelinePost[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)

  async function loadPosts(reset = false) {
    setLoading(true)
    setError(null)
    try {
      const currentOffset = reset ? 0 : offset
      const data = await fetchTimeline(20, currentOffset)
      if (reset) {
        setPosts(data.posts)
      } else {
        setPosts(prev => [...prev, ...data.posts])
      }
      setOffset(currentOffset + data.posts.length)
      setHasMore(data.posts.length === 20)
    } catch (err) {
      setError('タイムラインの取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  useEffect(function () {
    loadPosts(true)
  }, [])

  function refresh() {
    setOffset(0)
    loadPosts(true)
  }

  function loadMore() {
    if (!loading && hasMore) {
      loadPosts(false)
    }
  }

  return { posts, loading, error, refresh, loadMore, hasMore }
}
