import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll, expect, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import * as axeMatchers from 'vitest-axe/matchers'
import { server } from './msw'
import { clearToken } from '../api/tokenStore'

// jest-axe matcher for the accessibility suite (see src/test/a11y.test.tsx).
// Namespace import: vitest-axe's .d.ts declares the matcher type-only, so a named
// import trips verbatimModuleSyntax — the runtime value is present on the module.
expect.extend(axeMatchers)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterAll(() => server.close())

// jsdom has no matchMedia; default to "not light" (tests override as needed).
if (!window.matchMedia) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

afterEach(() => {
  cleanup()
  server.resetHandlers()
  clearToken()
  document.documentElement.removeAttribute('data-theme')
  localStorage.clear()
})
