import { describe, expect, it } from 'vitest'
import { formatDueShort, toApiDate, toInputDate } from './date'

describe('toApiDate', () => {
  it('converts input date → API date', () => {
    expect(toApiDate('2026-08-09')).toBe('09-08-2026')
  })
  it('returns empty for blank or malformed input', () => {
    expect(toApiDate('')).toBe('')
    expect(toApiDate('09-08-2026')).toBe('')
  })
})

describe('toInputDate', () => {
  it('converts API date → input date', () => {
    expect(toInputDate('09-08-2026')).toBe('2026-08-09')
  })
  it('round-trips with toApiDate', () => {
    expect(toApiDate(toInputDate('31-12-2026'))).toBe('31-12-2026')
  })
  it('returns empty for blank or malformed input', () => {
    expect(toInputDate('')).toBe('')
    expect(toInputDate('2026-08-09')).toBe('')
  })
})

describe('formatDueShort', () => {
  it('formats a short due label without leading zeros', () => {
    expect(formatDueShort('04-10-2026')).toBe('Oct 4')
    expect(formatDueShort('31-01-2027')).toBe('Jan 31')
  })
  it('returns empty for blank or malformed input', () => {
    expect(formatDueShort('')).toBe('')
    expect(formatDueShort('2026-10-04')).toBe('')
  })
})
