import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { actionRequest, createRequest, editRequest, fetchRequests } from '@/api/client'
import { RequestPayload, RequestsFilter } from '@/types'

export function useRequests(filter: RequestsFilter = {}) {
  return useQuery({
    queryKey: ['requests', filter],
    queryFn: () => fetchRequests(filter),
    staleTime: 10_000,
  })
}

export function useCreateRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: RequestPayload) => createRequest(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['requests'] })
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useEditRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: RequestPayload }) =>
      editRequest(id, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['requests'] })
    },
  })
}

export function useRequestAction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, action, payload }: { id: number; action: string; payload?: RequestPayload }) =>
      actionRequest(id, action, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['requests'] })
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}
