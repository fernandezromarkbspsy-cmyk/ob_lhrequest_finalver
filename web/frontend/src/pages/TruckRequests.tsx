import { useState } from 'react'
import toast from 'react-hot-toast'
import { useRequests, useRequestAction } from '@/hooks/useRequests'
import RequestTable from '@/components/RequestTable/RequestTable'
import ActionModal from '@/components/ActionModal/ActionModal'
import { Request, RequestAction, RequestPayload, RequestsFilter } from '@/types'

const STATUS_OPTIONS = [
  { value: 'PENDING_MM',  label: 'Pending MM' },
  { value: 'ASSIGNED',    label: 'Assigned' },
  { value: 'FOR_DOCKING', label: 'For Docking' },
  { value: 'DOCKED',      label: 'Docked' },
  { value: 'CONFIRMED',   label: 'Confirmed' },
]

export default function TruckRequests() {
  const [filter, setFilter] = useState<RequestsFilter>({ queue: 'mm' })
  const { data, isLoading, refetch } = useRequests(filter)
  const actionMutation = useRequestAction()

  const [actionState, setActionState] = useState<{ req: Request; action: RequestAction } | null>(null)

  const handleActionSubmit = async (id: number, action: string, payload: RequestPayload) => {
    await actionMutation.mutateAsync({ id, action, payload })
    toast.success('Request updated')
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Truck Requests</h1>
          <p className="page-subtitle">Midmile truck assignment queue</p>
        </div>
      </div>

      <RequestTable
        requests={data?.requests ?? []}
        loading={isLoading}
        filter={filter}
        onFilterChange={(f) => setFilter((prev) => ({ ...prev, ...f }))}
        onAction={(req, action) => setActionState({ req, action })}
        onRefresh={refetch}
        statusOptions={STATUS_OPTIONS}
      />

      {actionState && (
        <ActionModal
          req={actionState.req}
          action={actionState.action}
          onClose={() => setActionState(null)}
          onSubmit={handleActionSubmit}
        />
      )}
    </div>
  )
}
