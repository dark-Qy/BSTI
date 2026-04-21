import { buildAuthFlowViewModel, type AuthStepViewModel } from './auth-flow'
import type { SessionStatusResponse } from './types'

interface LandingPageProps {
  booting: boolean
  busyAction: string | null
  error: string
  isCheckingAuthorizationReturn: boolean
  sessionId: string
  statusData: SessionStatusResponse | null
  onAnalyze: () => void
  onConnect: () => void
  onOpenVerificationURL: (url: string) => void
  onReset: () => void
}

export function LandingPage(props: LandingPageProps) {
  const {
    booting,
    busyAction,
    error,
    isCheckingAuthorizationReturn,
    sessionId,
    statusData,
    onAnalyze,
    onConnect,
    onOpenVerificationURL,
    onReset,
  } = props
  const panelState = buildAuthFlowViewModel({
    busyAction,
    isCheckingAuthorizationReturn,
    sessionId,
    statusData,
  })
  const panelError = error || statusData?.error || ''

  const handlePrimaryAction = () => {
    if (panelState.primaryAction === 'analyze') {
      onAnalyze()
      return
    }
    if (panelState.primaryAction === 'open_url' && panelState.primaryURL) {
      onOpenVerificationURL(panelState.primaryURL)
      return
    }
    if (panelState.primaryAction === 'connect') {
      onConnect()
    }
  }

  return (
    <section className="landing-page">
      <div className="landing-brand">BSTI 2026</div>

      <div className="landing-hero">
        <p className="landing-kicker">基于本地飞书数据，生成你的 BSTI 分析报告。</p>
        <h1>
          <span>洞悉你的工作。</span>
          <span className="gradient-line">重塑你的人格。</span>
        </h1>
        <p className="landing-subtitle">安全，且完全私密。所有授权数据仅在当前设备内处理。</p>
        <div className="landing-pills">
          <span>Local First</span>
          <span>Read-only Scope</span>
          <span>Private by Design</span>
        </div>
      </div>

      <div className="focus-panel">
        <div className="focus-icon" aria-hidden="true">
          {panelState.icon}
        </div>
        <div className="focus-copy">
          <div className="focus-status">{panelState.eyebrow}</div>
          <h2>{panelState.title}</h2>
          <p>{panelState.description}</p>
          {panelState.weakHint ? <div className="focus-note">{panelState.weakHint}</div> : null}
        </div>

        <AuthorizationSteps steps={panelState.steps} />

        <div className="focus-actions">
          <button
            className="primary-button"
            disabled={booting || panelState.primaryDisabled}
            onClick={handlePrimaryAction}
          >
            {panelState.primaryLabel}
          </button>

          {panelState.showReset ? (
            <button className="secondary-button" disabled={busyAction === 'session'} onClick={onReset}>
              切换账号 / 新建会话
            </button>
          ) : null}
        </div>

        {panelError ? <div className="error-banner">{panelError}</div> : null}
      </div>
    </section>
  )
}

function AuthorizationSteps(props: { steps: AuthStepViewModel[] }) {
  return (
    <div className="auth-steps" aria-label="authorization-steps">
      {props.steps.map((step) => (
        <div className={`auth-step auth-step-${step.state}`} key={step.number}>
          <div className="auth-step-marker">
            <span>{step.number}</span>
          </div>
          <div className="auth-step-copy">
            <strong>{step.title}</strong>
            <p>{step.description}</p>
          </div>
          <span className="auth-step-state">{step.badge}</span>
        </div>
      ))}
    </div>
  )
}
