import { describe, expect, it } from 'vitest'
import { bandToneClass } from './motivation'

describe('bandToneClass', () => {
  it('maps each band to its tone colour', () => {
    expect(bandToneClass('not-started')).toBe('text-text-muted')
    expect(bandToneClass('in-progress')).toBe('text-blue-ink')
    expect(bandToneClass('almost-there')).toBe('text-teal-ink')
    expect(bandToneClass('achieved')).toBe('text-gold-ink')
  })
})
