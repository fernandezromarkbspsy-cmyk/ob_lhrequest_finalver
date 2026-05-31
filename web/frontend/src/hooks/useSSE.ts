import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/authStore'

export function useSSE() {
  const user = useAuthStore((s) => s.user)
  const qc = useQueryClient()

  useEffect(() => {
    if (!user) return

    let es: EventSource | null = null
    let delay = 1000
    let timer: ReturnType<typeof setTimeout> | null = null
    let cancelled = false

    const invalidate = () => {
      qc.invalidateQueries({ queryKey: ['requests'] })
      qc.invalidateQueries({ queryKey: ['stats'] })
    }

    function connect() {
      if (cancelled) return

      es = new EventSource('/api/events')

      es.addEventListener('system.connected', () => {
        delay = 1000
      })

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

      es.onerror = () => {
        es?.close()
        es = null
        if (cancelled) return
        const jitter = delay * 0.2 * Math.random()
        const waitFor = delay + jitter
        const nextDelay = Math.min(delay * 2, 30000)
        timer = setTimeout(() => {
          delay = nextDelay
          connect()
        }, waitFor)
      }
    }

    connect()

    return () => {
      cancelled = true
      if (timer !== null) clearTimeout(timer)
      es?.close()
    }
  }, [user?.id, qc])
}
