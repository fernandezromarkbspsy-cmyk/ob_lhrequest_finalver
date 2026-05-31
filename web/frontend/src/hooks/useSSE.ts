import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/authStore'

export function useSSE() {
  const user = useAuthStore((s) => s.user)
  const qc = useQueryClient()

  useEffect(() => {
    if (!user) return

    const es = new EventSource('/api/events')

    const invalidate = () => {
      qc.invalidateQueries({ queryKey: ['requests'] })
      qc.invalidateQueries({ queryKey: ['stats'] })
    }

    es.addEventListener('request.created', (e) => {
      invalidate()
      try {
        const data = JSON.parse(e.data)
        toast.success(`New request — ${data.payload?.cluster ?? ''}`)
      } catch {
        toast.success('New request submitted')
      }
    })

    es.addEventListener('request.status_changed', (e) => {
      invalidate()
      try {
        const data = JSON.parse(e.data)
        const from = data.previous_status?.replace(/_/g, ' ')
        const to = data.status?.replace(/_/g, ' ')
        if (from && to && from !== to) {
          toast(`Request #${data.aggregate_id}: ${from} → ${to}`, { icon: '🔄' })
        }
      } catch {
        invalidate()
      }
    })

    es.addEventListener('request.updated', invalidate)

    return () => es.close()
  }, [user?.id, qc])
}
