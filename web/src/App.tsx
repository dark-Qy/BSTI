import html2canvas from 'html2canvas'
import { startTransition, useEffect, useRef, useState } from 'react'
import type { RefObject } from 'react'
import {
  analyzeSession,
  createSession,
  getReportData,
  getSessionStatus,
  startLogin,
} from './api'
import type {
  ReportData,
  SessionEvent,
  SessionStatus,
  SessionStatusResponse,
} from './types'

function App() {
  const [sessionId, setSessionId] = useState('')
  const [statusData, setStatusData] = useState<SessionStatusResponse | null>(null)
  const [reportData, setReportData] = useState<ReportData | null>(null)
  const [booting, setBooting] = useState(true)
  const [busyAction, setBusyAction] = useState<'login' | 'analyze' | 'poster' | 'session' | null>(null)
  const [uiError, setUiError] = useState('')
  const [displayedProgress, setDisplayedProgress] = useState(5)
  const [posterNotice, setPosterNotice] = useState('')
  const reportRef = useRef<HTMLDivElement>(null)

  async function bootstrapSession() {
    setBooting(true)
    setBusyAction('session')
    setUiError('')
    setPosterNotice('')
    setReportData(null)

    try {
      const created = await createSession()
      setSessionId(created.session_id)
      const status = await getSessionStatus(created.session_id)
      setStatusData(status)
    } catch (error) {
      setUiError(toMessage(error))
    } finally {
      setBusyAction(null)
      setBooting(false)
    }
  }

  useEffect(() => {
    void bootstrapSession()
  }, [])

  useEffect(() => {
    if (!sessionId || !statusData) {
      return
    }
    if (statusData.status === 'done' || statusData.status === 'failed') {
      return
    }

    const timer = window.setInterval(async () => {
      try {
        const next = await getSessionStatus(sessionId)
        setStatusData(next)
      } catch (error) {
        setUiError(toMessage(error))
      }
    }, 1800)

    return () => window.clearInterval(timer)
  }, [sessionId, statusData])

  useEffect(() => {
    if (!statusData) {
      return
    }
    setDisplayedProgress((current) => {
      if (statusData.status === 'failed') {
        return current
      }
      return Math.max(current, Math.min(statusData.progress.percent, 97))
    })
  }, [statusData])

  useEffect(() => {
    if (!statusData || !sessionId || reportData) {
      return
    }
    if (statusData.status !== 'done') {
      return
    }

    void (async () => {
      try {
        const response = await getReportData(sessionId)
        startTransition(() => {
          setReportData(response.report)
          setDisplayedProgress(100)
        })
      } catch (error) {
        setUiError(toMessage(error))
      }
    })()
  }, [sessionId, statusData, reportData])

  const pageMode =
    reportData && statusData?.status === 'done'
      ? 'result'
      : statusData && ['collecting', 'analyzing', 'done'].includes(statusData.status)
        ? 'analysis'
        : 'landing'

  async function handleConnect() {
    if (!sessionId) {
      return
    }
    setBusyAction('login')
    setUiError('')

    try {
      const data = await startLogin(sessionId)
      const next = await getSessionStatus(sessionId)
      setStatusData(next)
      if (data.verification_url) {
        window.open(data.verification_url, '_blank', 'noopener,noreferrer')
      }
    } catch (error) {
      setUiError(toMessage(error))
    } finally {
      setBusyAction(null)
    }
  }

  async function handleAnalyze() {
    if (!sessionId) {
      return
    }
    setBusyAction('analyze')
    setUiError('')
    setPosterNotice('')

    try {
      await analyzeSession(sessionId)
      const next = await getSessionStatus(sessionId)
      setStatusData(next)
      setReportData(null)
      setDisplayedProgress(Math.max(next.progress.percent, 55))
    } catch (error) {
      setUiError(toMessage(error))
    } finally {
      setBusyAction(null)
    }
  }

  async function handlePosterDownload() {
    if (!reportRef.current || !reportData) {
      return
    }
    setBusyAction('poster')
    setPosterNotice('')

    try {
      const canvas = await html2canvas(reportRef.current, {
        backgroundColor: '#000000',
        scale: 2,
        useCORS: true,
      })
      const link = document.createElement('a')
      link.download = `${reportData.primary_persona.shorthand.toLowerCase()}-bspi-report.png`
      link.href = canvas.toDataURL('image/png')
      link.click()
      setPosterNotice('专属海报已导出到本地。')
    } catch (error) {
      setUiError(toMessage(error))
    } finally {
      setBusyAction(null)
    }
  }

  const timelineEvents = [...(statusData?.events || [])].sort((a, b) =>
    a.timestamp.localeCompare(b.timestamp),
  )

  return (
    <div className="app-shell">
      <div className="ambient ambient-violet" />
      <div className="ambient ambient-blue" />
      <main className={`page page-${pageMode}`}>
        {pageMode === 'landing' ? (
          <LandingPage
            booting={booting}
            busyAction={busyAction}
            error={uiError}
            sessionId={sessionId}
            statusData={statusData}
            onAnalyze={handleAnalyze}
            onConnect={handleConnect}
            onReset={() => void bootstrapSession()}
          />
        ) : null}

        {pageMode === 'analysis' && statusData ? (
          <AnalysisPage
            busyAction={busyAction}
            displayedProgress={displayedProgress}
            events={timelineEvents}
            statusData={statusData}
          />
        ) : null}

        {pageMode === 'result' && statusData && reportData ? (
          <ResultPage
            busyAction={busyAction}
            error={uiError}
            posterNotice={posterNotice}
            report={reportData}
            reportRef={reportRef}
            sessionId={sessionId}
            onAnalyze={handleAnalyze}
            onDownloadPoster={handlePosterDownload}
            onReset={() => void bootstrapSession()}
          />
        ) : null}
      </main>
    </div>
  )
}

interface LandingPageProps {
  booting: boolean
  busyAction: string | null
  error: string
  sessionId: string
  statusData: SessionStatusResponse | null
  onAnalyze: () => void
  onConnect: () => void
  onReset: () => void
}

function LandingPage(props: LandingPageProps) {
  const { booting, busyAction, error, sessionId, statusData, onAnalyze, onConnect, onReset } = props
  const status = statusData?.status || 'created'
  const panelState = resolveLandingPanel(status, busyAction, statusData?.verification_url)
  const panelError = error || statusData?.error || ''

  return (
    <section className="landing-page">
      <div className="landing-brand">BSPI 2026</div>

      <div className="landing-hero">
        <p className="landing-kicker">基于本地飞书数据，生成你的 BSPI 分析报告。</p>
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
          <p>{panelState.description(sessionId)}</p>
        </div>

        <div className="focus-actions">
          <button
            className="primary-button"
            disabled={panelState.primaryDisabled(booting)}
            onClick={panelState.primaryAction === 'analyze' ? onAnalyze : onConnect}
          >
            {panelState.primaryLabel}
          </button>

          {panelState.showReset ? (
            <button className="secondary-button" disabled={busyAction === 'session'} onClick={onReset}>
              新建会话
            </button>
          ) : null}

          {statusData?.verification_url ? (
            <a
              className="text-link"
              href={statusData.verification_url}
              rel="noreferrer"
              target="_blank"
            >
              在新窗口中继续飞书授权
            </a>
          ) : null}
        </div>

        {panelError ? <div className="error-banner">{panelError}</div> : null}
      </div>
    </section>
  )
}

interface AnalysisPageProps {
  busyAction: string | null
  displayedProgress: number
  events: SessionEvent[]
  statusData: SessionStatusResponse
}

function AnalysisPage(props: AnalysisPageProps) {
  const { busyAction, displayedProgress, events, statusData } = props
  const progress = Math.max(displayedProgress, statusData.progress.percent)

  return (
    <section className="analysis-page">
      <div className="analysis-hero">
        <div
          className="progress-ring"
          style={{ ['--progress' as string]: `${progress}%` }}
        >
          <div className="progress-ring-inner">
            <strong>{statusData.progress.percent}%</strong>
            <span>数据处理中...</span>
          </div>
        </div>

        <div className="analysis-copy">
          <h2>正在采集飞书协作数据</h2>
          <p>
            将聊天、文档、日程等数据交由大模型处理，生成结构化 BSPI
            画像。请勿关闭页面。
          </p>
          <div className="analysis-note">
            {busyAction === 'analyze'
              ? '分析任务已提交，系统正在进入稳定处理阶段。'
              : statusData.progress.label}
          </div>
        </div>
      </div>

      <div className="timeline-panel">
        <div className="timeline-header">
          <div className="timeline-status-dot" />
          <div>
            <div className="timeline-kicker">Session Flow</div>
            <h3>状态流转</h3>
          </div>
        </div>

        <div className="timeline-stream">
          {events.map((event, index) => (
            <TimelineEvent
              event={event}
              index={index}
              key={`${event.timestamp}-${event.stage}-${event.label}`}
              total={events.length}
            />
          ))}
        </div>
      </div>
    </section>
  )
}

interface ResultPageProps {
  busyAction: string | null
  error: string
  posterNotice: string
  report: ReportData
  reportRef: RefObject<HTMLDivElement | null>
  sessionId: string
  onAnalyze: () => void
  onDownloadPoster: () => void
  onReset: () => void
}

function ResultPage(props: ResultPageProps) {
  const {
    busyAction,
    error,
    posterNotice,
    report,
    reportRef,
    sessionId,
    onAnalyze,
    onDownloadPoster,
    onReset,
  } = props
  const persona = report.primary_persona

  return (
    <section className="result-page">
      <header className="result-header">
        <div className="result-brand">BSPI 2026</div>
        <div className="result-header-actions">
          <button className="secondary-button" onClick={onReset}>
            新建会话
          </button>
          <a
            className="text-link"
            href={`/api/sessions/${sessionId}/report`}
            rel="noreferrer"
            target="_blank"
          >
            打开兼容版 HTML 报告
          </a>
        </div>
      </header>

      <div className="report-canvas" ref={reportRef}>
        <div className="report-grid">
          <section className="hero-column">
            <div className="persona-overline">{persona.shorthand}</div>
            <h1>{persona.chinese_label}</h1>
            <p className="hero-one-liner">{persona.one_liner}</p>
            <div className="outline-tags">
              {report.highlight_tags.map((tag) => (
                <span key={tag}>{tag}</span>
              ))}
            </div>
            <img alt={persona.shorthand} className="persona-art" src={persona.image_url} />
          </section>

          <section className="report-column">
            <div className="editorial-section">
              <div className="section-kicker">Data Portrait</div>
              <h2>行为解析</h2>
              <div className="vector-list">
                {report.behavior_vectors.map((vector) => (
                  <article className="vector-row" key={vector.label}>
                    <div className="vector-head">
                      <div>
                        <h3>{vector.label}</h3>
                        <p>{vector.summary}</p>
                      </div>
                      <strong>{vector.score}</strong>
                    </div>
                    <div className="vector-poles">
                      <span>{vector.left_pole}</span>
                      <span>{vector.right_pole}</span>
                    </div>
                    <div className="vector-track">
                      <div className="vector-fill" style={{ width: `${vector.score}%` }} />
                      <div className="vector-thumb" style={{ left: `${vector.score}%` }} />
                    </div>
                  </article>
                ))}
              </div>
            </div>

            <div className="editorial-section">
              <div className="section-kicker">Deep Reading</div>
              <h2>深度洞察</h2>
              <div className="insight-copy">
                <p>{report.analysis.summary}</p>
                <dl className="insight-list">
                  <div>
                    <dt>沟通风格</dt>
                    <dd>{report.analysis.communication_style}</dd>
                  </div>
                  <div>
                    <dt>工作偏好</dt>
                    <dd>{report.analysis.work_preferences}</dd>
                  </div>
                  <div>
                    <dt>风险与盲区</dt>
                    <dd>{report.analysis.blind_spots}</dd>
                  </div>
                  <div>
                    <dt>人格定义</dt>
                    <dd>{persona.canonical_description}</dd>
                  </div>
                </dl>
              </div>
            </div>
          </section>
        </div>

        <section className="evidence-strip">
          <div className="editorial-section">
            <div className="section-kicker">Coverage</div>
            <h2>证据与覆盖</h2>
            <div className="coverage-layout">
              <ul className="evidence-list">
                {report.analysis.evidence.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
              <div className="coverage-copy">
                <p>{report.coverage.summary}</p>
                <div className="coverage-meta">
                  <span>已纳入：{report.coverage.successful_domains.join(' / ') || '无'}</span>
                  <span>未纳入：{report.coverage.failed_domains.join(' / ') || '无'}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <footer className="report-footer">
          <div className="footer-meta">
            <span>置信度 {report.analysis.confidence}</span>
            <span>{report.share_card.disclaimer_short}</span>
            <span>{report.analysis.disclaimer}</span>
          </div>
        </footer>
      </div>

      <div className="result-toolbar">
        <button className="secondary-button" disabled={busyAction === 'analyze'} onClick={onAnalyze}>
          重新生成
        </button>
        <button className="primary-button" disabled={busyAction === 'poster'} onClick={onDownloadPoster}>
          {busyAction === 'poster' ? '正在导出...' : '保存专属报告'}
        </button>
      </div>

      {posterNotice ? <div className="notice-banner">{posterNotice}</div> : null}
      {error ? <div className="error-banner">{error}</div> : null}
    </section>
  )
}

function TimelineEvent(props: { event: SessionEvent; index: number; total: number }) {
  const { event, index, total } = props
  const completed = index < total - 1
  const active = index === total - 1

  return (
    <article className="timeline-item">
      <div className={`timeline-node ${completed ? 'done' : ''} ${active ? 'active' : ''}`}>
        <span />
      </div>
      <div className="timeline-content">
        <div className="timeline-row">
          <strong>{friendlyStatus(event.stage as SessionStatus)}</strong>
          <span>
            {new Date(event.timestamp).toLocaleTimeString('zh-CN', {
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
            })}
          </span>
        </div>
        <p>{event.label}</p>
      </div>
    </article>
  )
}

function resolveLandingPanel(
  status: SessionStatus,
  busyAction: string | null,
  verificationURL?: string,
) {
  switch (status) {
    case 'authenticated':
      return {
        eyebrow: 'Ready to Start',
        title: '已就绪，可开始分析',
        description: () => '授权已完成，接下来将开始生成你的结构化 BSPI 画像。',
        icon: '●',
        primaryLabel: busyAction === 'analyze' ? '正在提交分析...' : '开启 AI 深度解析',
        primaryAction: 'analyze' as const,
        primaryDisabled: (booting: boolean) => booting || busyAction === 'analyze',
        showReset: true,
      }
    case 'config_pending':
      return {
        eyebrow: 'Configuration',
        title: '继续完成配置',
        description: () => '本地服务已经就绪，请按提示完成飞书应用配置后继续授权。',
        icon: '◌',
        primaryLabel: busyAction === 'login' ? '正在准备授权...' : '继续配置与授权',
        primaryAction: 'connect' as const,
        primaryDisabled: (booting: boolean) => booting || busyAction === 'login',
        showReset: true,
      }
    case 'login_pending':
      return {
        eyebrow: 'Authorization',
        title: verificationURL ? '连接飞书' : '等待授权完成',
        description: () =>
          verificationURL
            ? '请完成浏览器授权，授权成功后系统会自动刷新当前状态。'
            : '授权链接已创建，请在新窗口中完成飞书确认。',
        icon: '↗',
        primaryLabel: busyAction === 'login' ? '正在准备授权...' : '去授权',
        primaryAction: 'connect' as const,
        primaryDisabled: (booting: boolean) => booting || busyAction === 'login',
        showReset: true,
      }
    case 'failed':
      return {
        eyebrow: 'Failed State',
        title: '连接失败',
        description: () => '无法完成飞书配置流程，请重试或新建会话。',
        icon: '×',
        primaryLabel: busyAction === 'login' ? '正在重试...' : '重新尝试',
        primaryAction: 'connect' as const,
        primaryDisabled: (booting: boolean) => booting || busyAction === 'login',
        showReset: true,
      }
    default:
      return {
        eyebrow: 'Privacy First',
        title: '连接飞书',
        description: (sessionId: string) =>
          sessionId
            ? `本地会话 ${sessionId.slice(0, 8)} 已创建。连接后即可拉取只读范围内的协作数据。`
            : '将基于本地飞书数据生成工作人格画像，所有信息只在当前设备中处理。',
        icon: '◎',
        primaryLabel: busyAction === 'login' ? '正在准备授权...' : '连接我的飞书',
        primaryAction: 'connect' as const,
        primaryDisabled: (booting: boolean) => booting || busyAction === 'login',
        showReset: false,
      }
  }
}

function friendlyStatus(status: SessionStatus) {
  switch (status) {
    case 'config_pending':
      return '等待配置'
    case 'login_pending':
      return '等待授权'
    case 'authenticated':
      return '已就绪'
    case 'collecting':
      return '采集数据'
    case 'analyzing':
      return '人格分析'
    case 'done':
      return '已完成'
    case 'failed':
      return '失败'
    default:
      return '待开始'
  }
}

function toMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}

export default App
