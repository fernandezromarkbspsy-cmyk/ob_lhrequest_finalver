import { useQuery } from '@tanstack/react-query'
import { fetchClusters } from '@/api/client'

export function useClusters() {
  return useQuery({
    queryKey: ['clusters'],
    queryFn: fetchClusters,
    staleTime: 5 * 60_000,
  })
}
