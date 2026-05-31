import { Component, ErrorInfo, ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

export default class ErrorBoundary extends Component<Props, State> {
  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  state: State = { hasError: false }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, info)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: '16px' }}>
          <span>Something went wrong</span>
          <button onClick={() => window.location.reload()}>Refresh page</button>
        </div>
      )
    }
    return this.props.children
  }
}
