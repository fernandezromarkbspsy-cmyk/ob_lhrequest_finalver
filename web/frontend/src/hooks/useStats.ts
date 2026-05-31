import { useQuery } from '@tanstack/react-query'
import { fetchStats } from '@/api/client'

export function useStats() {
  return useQuery({
    queryKey: ['stats'],
    queryFn: fetchStats,
    refetchInterval: 5_000,
    staleTime: 4_000,
  })
}
