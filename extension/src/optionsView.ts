export function formatTokenExpiry(expiresAt: string | undefined): string {
  if (!expiresAt) {
    return '不明'
  }
  const date = new Date(expiresAt)
  if (Number.isNaN(date.getTime())) {
    return '不明'
  }
  return date.toLocaleString()
}
