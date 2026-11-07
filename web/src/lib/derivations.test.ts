import { describe, expect, it } from 'vitest'
import { timeOfDayGreeting } from './greeting'

describe('timeOfDayGreeting', () => {
  it('picks a greeting by hour', () => {
    expect(timeOfDayGreeting(8)).toBe('Good morning')
    expect(timeOfDayGreeting(13)).toBe('Good afternoon')
    expect(timeOfDayGreeting(20)).toBe('Good evening')
  })
})
