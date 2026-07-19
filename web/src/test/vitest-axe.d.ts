// Types for the jest-axe matcher registered in setup.ts. vitest-axe ships an
// `extend-expect` that augments the legacy `Vi` global namespace, which Vitest 4
// no longer uses — so augment the `vitest` module's assertion interfaces directly.
import type { AxeMatchers } from 'vitest-axe/matchers'

/* eslint-disable @typescript-eslint/no-empty-object-type, @typescript-eslint/no-unused-vars */
declare module 'vitest' {
  interface Assertion<T = unknown> extends AxeMatchers {}
  interface AsymmetricMatchersContaining extends AxeMatchers {}
}
