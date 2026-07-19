import { QueryClient } from '@tanstack/react-query'

// Data hooks arrive in B3; the client is configured here so wiring stays additive.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
  },
})
