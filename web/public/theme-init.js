// Resolve and apply the theme before the app renders, so there is no
// wrong-theme flash. Mirrors the logic in src/theme/theme.ts.
//
// This lives in a same-origin file (not an inline <script>) so the production
// Content-Security-Policy can keep a strict `script-src 'self'` with no
// 'unsafe-inline'. It is a classic (blocking) script in <head>, so it still runs
// before first paint.
;(function () {
  try {
    var stored = localStorage.getItem('tiyy-theme')
    var theme =
      stored === 'light' || stored === 'dark'
        ? stored
        : window.matchMedia('(prefers-color-scheme: light)').matches
          ? 'light'
          : 'dark'
    document.documentElement.setAttribute('data-theme', theme)
  } catch (e) {
    document.documentElement.setAttribute('data-theme', 'dark')
  }
})()
