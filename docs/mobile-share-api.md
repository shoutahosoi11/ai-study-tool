# Mobile Share API

`/api/highlights/share` is the intake endpoint for iOS / Android share-sheet flows.

## Authentication

- Firebase ID token in `Authorization: Bearer <token>`

## Request

`POST /api/highlights/share`

```json
{
  "content": "Focus is a superpower.",
  "book_title": "Deep Work",
  "book_author": "Cal Newport",
  "source_app": "kindle",
  "source_url": "https://read.amazon.com/notebook",
  "shared_at": "2026-04-24T09:00:00Z"
}
```

### Fields

- `content`: required
- `book_title`: optional
- `book_author`: optional
- `source_app`: optional, for example `kindle`, `safari`, `chrome`
- `source_url`: optional
- `shared_at`: optional RFC3339 timestamp

## Response

```json
{
  "saved": true,
  "duplicate": false,
  "highlight": {
    "id": "11111111-2222-3333-4444-555555555555",
    "book_title": "Deep Work",
    "book_author": "Cal Newport",
    "content": "Focus is a superpower.",
    "highlighted_at": "2026-04-24T09:00:00Z",
    "source": "mobile_share",
    "source_app": "kindle",
    "source_url": "https://read.amazon.com/notebook",
    "created_at": "2026-04-24T09:00:01Z"
  }
}
```

If the highlight is already stored for the same user, the API returns:

```json
{
  "saved": false,
  "duplicate": true
}
```

## Notes

- The existing `/api/highlights/import` endpoint remains Kindle-extension-specific.
- `mobile_share` is stored as a distinct highlight `source`.
- Deduplication for mobile share imports uses `source_app`, `source_url`, `book_title`, `book_author`, and normalized `content`.
