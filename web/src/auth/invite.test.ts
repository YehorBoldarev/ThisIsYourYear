import { describe, expect, it } from 'vitest'
import { inviteFrom, withInvite } from './invite'

describe('invite helpers', () => {
  it('reads the code from a query string', () => {
    expect(inviteFrom('?invite=sunflower-2027')).toBe('sunflower-2027')
    expect(inviteFrom('?x=1&invite=%20a%20b%20')).toBe('a b')
    expect(inviteFrom('')).toBe('')
  })

  it('attaches the code only when there is one', () => {
    expect(withInvite('/register', 'sun flower')).toBe('/register?invite=sun%20flower')
    expect(withInvite('/register', '  ')).toBe('/register')
    expect(withInvite('/login', undefined)).toBe('/login')
  })
})
