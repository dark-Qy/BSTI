import html2canvas from 'html2canvas'
import { startTransition, useEffect, useRef, useState } from 'react'
import type { RefObject } from 'react'
import { LandingPage } from './landing-auth'
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
  const [authorizationReturnPhase, setAuthorizationReturnPhase] = useState<'idle' | 'armed' | 'checking'>('idle')
  const reportRef = useRef<HTMLDivElement>(null)
  const authorizationReturnPhaseRef = useRef<'idle' | 'armed' | 'checking'>('idle')
  const authorizationReturnDeadlineRef = useRef<number | null>(null)

  function clearAuthorizationReturnCheck() {
    authorizationReturnDeadlineRef.current = null
    authorizationReturnPhaseRef.current = 'idle'
    setAuthorizationReturnPhase('idle')
  }

  function armAuthorizationReturnCheck() {
    authorizationReturnDeadlineRef.current = null
    authorizationReturnPhaseRef.current = 'armed'
    setAuthorizationReturnPhase('armed')
  }

  async function refreshSessionStatus(reason: 'poll' | 'return' | 'connect' = 'poll') {
    if (!sessionId) {
      return null
    }

    try {
      const next = await getSessionStatus(sessionId)
      setStatusData(next)

      if (['authenticated', 'collecting', 'analyzing', 'done', 'failed'].includes(next.status)) {
        clearAuthorizationReturnCheck()
        return next
      }

      if (reason === 'return') {
        authorizationReturnDeadlineRef.current = Date.now() + 6000
        authorizationReturnPhaseRef.current = 'checking'
        setAuthorizationReturnPhase('checking')
      }

      return next
    } catch (error) {
      setUiError(toMessage(error))
      return null
    }
  }

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
      clearAuthorizationReturnCheck()
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
    authorizationReturnPhaseRef.current = authorizationReturnPhase
  }, [authorizationReturnPhase])

  useEffect(() => {
    if (!sessionId || !statusData) {
      return
    }
    if (statusData.status === 'done' || statusData.status === 'failed') {
      return
    }

    const timer = window.setInterval(async () => {
      void refreshSessionStatus('poll')
    }, authorizationReturnPhase === 'checking' ? 400 : 1800)

    return () => window.clearInterval(timer)
  }, [authorizationReturnPhase, sessionId, statusData])

  useEffect(() => {
    if (authorizationReturnPhase !== 'checking') {
      return
    }
    const deadline = authorizationReturnDeadlineRef.current
    if (!deadline) {
      return
    }
    const remaining = deadline - Date.now()
    if (remaining <= 0) {
      clearAuthorizationReturnCheck()
      return
    }
    const timer = window.setTimeout(() => {
      clearAuthorizationReturnCheck()
    }, remaining)
    return () => window.clearTimeout(timer)
  }, [authorizationReturnPhase])

  useEffect(() => {
    function beginAuthorizationReturnCheck() {
      if (authorizationReturnPhaseRef.current !== 'armed') {
        return
      }
      if (document.visibilityState === 'hidden') {
        return
      }
      authorizationReturnDeadlineRef.current = Date.now() + 6000
      authorizationReturnPhaseRef.current = 'checking'
      setAuthorizationReturnPhase('checking')
      void refreshSessionStatus('return')
    }

    window.addEventListener('focus', beginAuthorizationReturnCheck)
    document.addEventListener('visibilitychange', beginAuthorizationReturnCheck)

    return () => {
      window.removeEventListener('focus', beginAuthorizationReturnCheck)
      document.removeEventListener('visibilitychange', beginAuthorizationReturnCheck)
    }
  }, [sessionId])

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
      const next = await refreshSessionStatus('connect')
      if (data.verification_url) {
        if (next?.status !== 'authenticated') {
          armAuthorizationReturnCheck()
        }
        window.open(data.verification_url, '_blank', 'noopener,noreferrer')
      }
    } catch (error) {
      setUiError(toMessage(error))
    } finally {
      setBusyAction(null)
    }
  }

  function handleOpenVerificationURL(url: string) {
    armAuthorizationReturnCheck()
    window.open(url, '_blank', 'noopener,noreferrer')
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
      link.download = `${reportData.primary_persona.shorthand.toLowerCase()}-bsti-report.png`
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
            isCheckingAuthorizationReturn={authorizationReturnPhase === 'checking'}
            sessionId={sessionId}
            statusData={statusData}
            onAnalyze={handleAnalyze}
            onConnect={handleConnect}
            onOpenVerificationURL={handleOpenVerificationURL}
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
            将聊天、文档、日程等数据交由大模型处理，生成结构化 BSTI
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
  const { frontDisclaimer, footerDisclaimer } = splitDisclaimers(
    report.share_card.disclaimer_short,
    report.analysis.disclaimer,
  )

  return (
    <section className="result-page">
      <header className="result-header">
        <div className="result-brand">BSTI 2026</div>
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
          <section className="hero-column report-region report-region-hero">
            <div className="hero-copy">
              <div className="persona-overline">{persona.shorthand}</div>
              <h1>{persona.chinese_label}</h1>
              <p className="hero-subtitle">
                {persona.byte_style_dimension} / {persona.analysis_dimension}
              </p>
            </div>

            <div className="hero-visual">
              <img alt={persona.shorthand} className="persona-art" src={persona.image_url} />
            </div>
          </section>

          <section className="editorial-section vector-section report-region report-region-vectors">
            <div className="section-header">
              <div>
                <div className="section-kicker">Data Portrait</div>
                <h2>行为解析</h2>
              </div>
              <span className="confidence-badge">置信度 {report.analysis.confidence}</span>
            </div>
            {frontDisclaimer ? <p className="section-meta">{frontDisclaimer}</p> : null}
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
          </section>

          <section className="editorial-section insight-section report-region report-region-insight">
            <div className="section-kicker">Deep Reading</div>
            <h2>深度洞察</h2>
            <div className="insight-copy">
              <p className="hero-one-liner">{persona.one_liner}</p>
              <div className="outline-tags insight-tags">
                {report.highlight_tags.map((tag) => (
                  <span key={tag}>{tag}</span>
                ))}
              </div>
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
          </section>

          <section className="editorial-section evidence-section report-region report-region-evidence">
            <div className="section-kicker">Coverage</div>
            <h2>证据与覆盖</h2>
            <ul className="evidence-list">
              {report.analysis.evidence.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
            <p className="coverage-summary">{report.coverage.summary}</p>
            <div className="coverage-meta">
              <div className="coverage-item">
                <strong>已纳入：</strong>
                <span>{report.coverage.successful_domains.join(' / ') || '无'}</span>
              </div>
              <div className="coverage-item">
                <strong>未纳入：</strong>
                <span>{report.coverage.failed_domains.join(' / ') || '无'}</span>
              </div>
            </div>
          </section>

          <footer className="report-footer report-region report-region-footer">
            <div className="footer-stack">
              {footerDisclaimer ? <p className="footer-disclaimer">{footerDisclaimer}</p> : null}
            </div>
          </footer>
        </div>
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
  const stateClass = active ? 'is-current' : completed ? 'is-completed' : 'is-pending'

  return (
    <article className={`timeline-item ${stateClass}`}>
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

function splitDisclaimers(shortText: string, longText: string) {
  const shortDisclaimer = normalizeText(shortText)
  const longDisclaimer = normalizeText(longText)

  if (!shortDisclaimer && !longDisclaimer) {
    return { frontDisclaimer: '', footerDisclaimer: '' }
  }

  if (!shortDisclaimer) {
    return { frontDisclaimer: '', footerDisclaimer: longDisclaimer }
  }

  if (!longDisclaimer) {
    return { frontDisclaimer: shortDisclaimer, footerDisclaimer: '' }
  }

  if (
    shortDisclaimer === longDisclaimer ||
    shortDisclaimer.includes(longDisclaimer) ||
    longDisclaimer.includes(shortDisclaimer)
  ) {
    return {
      frontDisclaimer: shortDisclaimer.length <= longDisclaimer.length ? shortDisclaimer : longDisclaimer,
      footerDisclaimer: '',
    }
  }

  return {
    frontDisclaimer: shortDisclaimer,
    footerDisclaimer: longDisclaimer,
  }
}

function normalizeText(value: string) {
  return value.replace(/\s+/g, ' ').trim()
}

function toMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}

export default App
