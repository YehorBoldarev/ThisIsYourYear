import { describe, expect, it } from 'vitest'
import { daysToNewYear, yearElapsedFraction } from './year'

describe('yearElapsedFraction', () => {
  it('is ~0 at the start of the year and ~1 at the end', () => {
    expect(yearElapsedFraction(new Date(2026, 0, 1, 0, 0, 0))).toBeCloseTo(0, 5)
    expect(yearElapsedFraction(new Date(2026, 11, 31, 23, 59, 59))).toBeGreaterThan(0.99)
  })

  it('is about half-way at mid-year', () => {
    expect(yearElapsedFraction(new Date(2026, 6, 2))).toBeGreaterThan(0.49)
    expect(yearElapsedFraction(new Date(2026, 6, 2))).toBeLessThan(0.51)
  })
})

describe('daysToNewYear', () => {
  it('counts whole days remaining', () => {
    expect(daysToNewYear(new Date(2026, 11, 31, 0, 0, 0))).toBe(1)
    expect(daysToNewYear(new Date(2026, 0, 1, 0, 0, 0))).toBe(365)
  })
})
