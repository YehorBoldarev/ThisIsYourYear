import { describe, expect, it } from 'vitest'
import { motivationBand } from './motivation'

describe('motivationBand thresholds', () => {
  it('maps the ratio boundaries to the four bands', () => {
    expect(motivationBand(0, 0).band).toBe('not-started')
    expect(motivationBand(0, 5).band).toBe('not-started')
    expect(motivationBand(0.01, 100).band).toBe('in-progress')
    expect(motivationBand(0.79, 100).band).toBe('in-progress')
    expect(motivationBand(0.8, 10).band).toBe('almost-there')
    expect(motivationBand(0.99, 100).band).toBe('almost-there')
    expect(motivationBand(1, 8).band).toBe('achieved')
  })
})

describe('motivationBand copy', () => {
  it('carries the design copy, tone, and title per band', () => {
    expect(motivationBand(0, 0)).toMatchObject({
      shortLabel: 'Just started',
      title: 'The blank page',
      toneClass: 'text-text-muted',
    })
    expect(motivationBand(0.4, 10)).toMatchObject({
      shortLabel: 'In progress',
      title: 'Building momentum',
      toneClass: 'text-blue',
    })
    expect(motivationBand(0.9, 10)).toMatchObject({
      shortLabel: 'Almost there',
      title: 'In sight',
      toneClass: 'text-teal',
    })
    expect(motivationBand(1, 4)).toMatchObject({
      shortLabel: 'Achieved',
      title: 'A promise kept',
      toneClass: 'text-gold',
    })
    expect(motivationBand(1, 4).message).toContain('kept a promise')
  })
})
