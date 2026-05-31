# Chrome Extension Store Readiness

Complete this before packaging and submitting the Manifest V3 extension.

## Manifest

- [ ] `extension/manifest.json` is the production manifest.
- [ ] `minimum_chrome_version` is `102` or higher.
- [ ] `host_permissions` includes only:
  - `https://read.amazon.co.jp/notebook*`
  - `https://read.amazon.com/notebook*`
  - final production API origin, for example `https://api.ai-study-tool.com/*`
- [ ] `localhost`, `*.run.app`, and staging origins are absent.
- [ ] `activeTab`, `scripting`, `cookies`, `history`, `webRequest`,
  `unlimitedStorage`, and `<all_urls>` are not requested unless a future change
  explicitly justifies them.
- [ ] `storage` permission is justified because the extension stores settings
  and scoped extension token metadata locally.
- [ ] Content scripts match only Kindle Notebook pages.
- [ ] The extension does not auto-crawl or automatically import without user
  action.
- [ ] Import starts only from explicit user interaction.
- [ ] The scoped token is used by the service worker/background flow and is not
  injected into the Kindle page.

## Data Handling

The extension may process and send:

- Kindle highlight text
- book title
- author
- ASIN/location metadata

The extension does not currently send Kindle notes. If note sync is added later,
update permission justifications, privacy policy, backend authorization, and
store copy before release.

## Store Listing

- [ ] Privacy policy URL is final and reachable.
- [ ] Icon assets are present in required Chrome Web Store sizes.
- [ ] Store description explains user-triggered Kindle highlight import.
- [ ] Permission justification explains `storage` and limited host permissions.
- [ ] Data handling disclosure mentions highlights, book metadata, and account
  pairing.
- [ ] Amazon Kindle Notebook terms/platform risk has been reviewed.

## Disconnect / Revoke

- [ ] User can disconnect/revoke the extension token.
- [ ] Backend revocation invalidates future imports for that token.
- [ ] Re-pairing flow is documented for users.

## Build

```bash
cd extension
npm run typecheck
npm test
npm run build
```

Package only the production manifest, `options.html`, built `dist/`, and
required static assets. Exclude `node_modules`, tests, source maps if not needed
for store submission, and `manifest.development.json`.

