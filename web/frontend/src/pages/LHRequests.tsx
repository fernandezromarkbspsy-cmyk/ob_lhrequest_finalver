import { useState } from 'react'
import toast from 'react-hot-toast'
import { useRequests, useCreateRequest, useEditRequest, useRequestAction } from '@/hooks/useRequests'
import { useClusters } from '@/hooks/useClusters'
import { useAuthStore } from '@/store/authStore'
import RequestTable from '@/components/RequestTable/RequestTable'
import ActionModal from '@/components/ActionModal/ActionModal'
import RequestModal from '@/components/RequestModal/RequestModal'
import { Request, RequestAction, RequestPayload, RequestsFilter } from '@/types'

const STATUS_OPTIONS = [
  { value: 'PENDING_OPS', label: 'Pending OPS' },
  { value: 'PENDING_MM',  label: 'Pending MM' },
  { value: 'ASSIGNED',    label: 'Assigned' },
  { value: 'FOR_DOCKING', label: 'For Docking' },
  { value: 'DOCKED',      label: 'Docked' },
  { value: 'CONFIRMED',   label: 'Confirmed' },
  { value: 'REJECTED',    label: 'Rejected' },
  { value: 'CANCELED',    label: 'Canceled' },
]

export default function LHRequests() {
  const user = useAuthStore((s) => s.user)
  const [filter, setFilter] = useState<RequestsFilter>({ queue: 'outbound' })
  const { data, isLoading, refetch } = useRequests(filter)
  const { data: clusters } = useClusters()

  const createMutation = useCreateRequest()
  const editMutation = useEditRequest()
  const actionMutation = useRequestAction()

  const [actionState, setActionState] = useState<{ req: Request; action: RequestAction } | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [editReq, setEditReq] = useState<Request | undefined>()

  const canCreate = user && ['fte_ops', 'ops_pic', 'admin'].includes(user.role)

  const handleAction = (req: Request, action: RequestAction) => {
    if (action === 'edit') { setEditReq(req); return }
    setActionState({ req, action })
  }

  const handleActionSubmit = async (id: number, action: string, payload: RequestPayload) => {
    await actionMutation.mutateAsync({ id, action, payload })
    toast.success('Request updated successfully')
  }

  const handleCreate = async (payload: RequestPayload) => {
    await createMutation.mutateAsync(payload)
    toast.success('Request submitted!')
  }

  const handleEdit = async (payload: RequestPayload) => {
    if (!editReq) return
    await editMutation.mutateAsync({ id: editReq.id, payload })
    toast.success('Request updated')
    setEditReq(undefined)
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">LH Requests</h1>
          <p className="page-subtitle">Outbound linehaul truck requests</p>
        </div>
      </div>

      <RequestTable
        requests={data?.requests ?? []}
        loading={isLoading}
        filter={filter}
        onFilterChange={(f) => setFilter((prev) => ({ ...prev, ...f }))}
        onAction={handleAction}
        onRefresh={refetch}
        showCreate={canCreate}
        onCreateClick={() => setShowCreate(true)}
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

      {showCreate && (
        <RequestModal
          clusters={clusters ?? []}
          onClose={() => setShowCreate(false)}
          onSubmit={handleCreate}
        />
      )}

      {editReq && (
        <RequestModal
          clusters={clusters ?? []}
          editRequest={editReq}
          onClose={() => setEditReq(undefined)}
          onSubmit={handleEdit}
        />
      )}
    </div>
  )
}
