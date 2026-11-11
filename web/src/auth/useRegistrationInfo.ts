import { useQuery } from '@tanstack/react-query'
import { client } from '../api/client'
import { ApiError } from '../api/errors'
import type { components } from '../api/schema'

export type RegistrationInfo = components['schemas']['RegistrationInfo']

/** The server's sign-up policy: whether registration needs an invite code. */
export function useRegistrationInfo() {
  return useQuery({
    queryKey: ['auth', 'registration'],
    queryFn: async (): Promise<RegistrationInfo> => {
      const { data, error, response } = await client.GET('/api/v1/auth/registration')
      if (error || !data) throw ApiError.fromBody(response.status, error)
      return data
    },
    staleTime: Infinity, // a server setting; it doesn't change while the page is open
  })
}
