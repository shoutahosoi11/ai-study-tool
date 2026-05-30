import { build } from 'esbuild'

const common = {
  bundle: true,
  minify: false,
  sourcemap: true,
  target: 'chrome116',
  platform: 'browser',
  logLevel: 'info',
}

await Promise.all([
  build({
    ...common,
    entryPoints: ['src/background.ts'],
    outfile: 'dist/background.js',
    format: 'iife',
  }),
  build({
    ...common,
    entryPoints: ['src/contentScript.ts'],
    outfile: 'dist/contentScript.js',
    format: 'iife',
  }),
  build({
    ...common,
    entryPoints: ['src/options.tsx'],
    outfile: 'dist/options.js',
    format: 'iife',
  }),
])
