import { readFileSync } from 'node:fs'
import { readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('extension security shape', () => {
  it('does not let the content script read extension tokens', () => {
    const source = readFileSync(new URL('../src/contentScript.ts', import.meta.url), 'utf8')
    expect(source).not.toContain('getToken')
    expect(source).not.toContain('chrome.storage')
    expect(source).not.toContain('Authorization')
  })

  it('does not use HTML injection APIs', () => {
    for (const file of ['../src/contentScript.ts', '../src/options.tsx']) {
      const source = readFileSync(new URL(file, import.meta.url), 'utf8')
      expect(source).not.toContain('innerHTML')
      expect(source).not.toContain('dangerouslySetInnerHTML')
    }
  })

  it('keeps production host permissions narrow', () => {
    const manifest = JSON.parse(readFileSync(new URL('../manifest.json', import.meta.url), 'utf8')) as {
      minimum_chrome_version?: string
      permissions: string[]
      host_permissions: string[]
    }
    expect(manifest.minimum_chrome_version).toBe('102')
    expect(manifest.permissions).toEqual(['storage'])
    expect(manifest.permissions).not.toContain('activeTab')
    expect(manifest.permissions).not.toContain('scripting')
    expect(manifest.permissions).not.toContain('cookies')
    expect(manifest.permissions).not.toContain('history')
    expect(manifest.permissions).not.toContain('webRequest')
    expect(manifest.permissions).not.toContain('unlimitedStorage')
    expect(manifest.host_permissions).not.toContain('<all_urls>')
    expect(manifest.host_permissions.some((permission) => permission.includes('localhost'))).toBe(false)
    expect(manifest.host_permissions.some((permission) => permission.includes('run.app'))).toBe(false)
  })

  it('keeps development-only origins out of the production manifest', () => {
    const manifest = JSON.parse(readFileSync(new URL('../manifest.development.json', import.meta.url), 'utf8')) as {
      permissions: string[]
      host_permissions: string[]
    }
    expect(manifest.permissions).toEqual(['storage'])
    expect(manifest.host_permissions.some((permission) => permission.includes('localhost'))).toBe(true)
    expect(manifest.host_permissions.some((permission) => permission.includes('run.app'))).toBe(true)
  })

  it('does not log token, pairing, or raw API details from extension source', () => {
    for (const file of sourceFiles(new URL('../src', import.meta.url).pathname)) {
      const source = readFileSync(file, 'utf8')
      expect(source).not.toMatch(/console\./)
    }
  })
})

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const fullPath = join(dir, entry)
    if (statSync(fullPath).isDirectory()) {
      return sourceFiles(fullPath)
    }
    return fullPath.endsWith('.ts') || fullPath.endsWith('.tsx') ? [fullPath] : []
  })
}
