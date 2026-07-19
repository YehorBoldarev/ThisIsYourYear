import { describe, expect, it } from 'vitest'
import { motivationBand } from './motivation'
import { timeOfDayGreeting } from './greeting'

describe('motivationBand', () => {
  it('maps ratio + total to a band', () => {
    expect(motivationBand(0, 0).band).toBe('not-started')
    expect(motivationBand(0, 5).band).toBe('not-started')
    expect(motivationBand(0.4, 10).band).toBe('in-progress')
    expect(motivationBand(0.8, 10).band).toBe('almost-there')
    expect(motivationBand(0.95, 20).band).toBe('almost-there')
    expect(motivationBand(1, 8).band).toBe('achieved')
  })
})

describe('timeOfDayGreeting', () => {
  it('picks a greeting by hour', () => {
    expect(timeOfDayGreeting(8)).toBe('Good morning')
    expect(timeOfDayGreeting(13)).toBe('Good afternoon')
    expect(timeOfDayGreeting(20)).toBe('Good evening')
  })
})
