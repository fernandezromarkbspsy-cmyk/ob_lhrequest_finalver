import { useQuery } from '@tanstack/react-query'
import { fetchTrend } from '@/api/client'

export function useTrend() {
  return useQuery({
    queryKey: ['trend'],
    queryFn: fetchTrend,
    refetchInterval: 60_000,
    staleTime: 55_000,
  })
}
