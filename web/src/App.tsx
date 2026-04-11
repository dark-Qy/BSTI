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
import type { ReportData, SessionStatus, SessionStatusResponse } from './types'

const statusOrder: SessionStatus[] = [
  'created',
  'config_pending',
  'login_pending',
  'authenticated',
  'collecting',
  'analyzing',
  'done',
  'failed',
]

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
    if (!statusData) {
      return
    }
    if (statusData.status === 'done' && sessionId && !reportData) {
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
    }
  }, [sessionId, statusData, reportData])

  const pageMode =
    reportData && statusData?.status === 'done'
      ? 'result'
      : statusData && ['collecting', 'analyzing'].includes(statusData.status)
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
        backgroundColor: '#07111f',
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
      <div className="aurora aurora-one" />
      <div className="aurora aurora-two" />
      <div className="grid-overlay" />
      <main className={`page page-${pageMode}`}>
        {pageMode === 'landing' ? (
          <LandingPage
            booting={booting}
            busyAction={busyAction}
            error={uiError}
            sessionId={sessionId}
            statusData={statusData}
            onConnect={handleConnect}
            onAnalyze={handleAnalyze}
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
  onConnect: () => void
  onAnalyze: () => void
  onReset: () => void
}

function LandingPage(props: LandingPageProps) {
  const { booting, busyAction, error, sessionId, statusData, onAnalyze, onConnect, onReset } = props
  const ready = statusData?.status === 'authenticated'
  const verificationURL = statusData?.verification_url

  return (
    <section className="landing-layout">
      <div className="hero-panel glass-card">
        <div className="eyebrow">BSPI 2026</div>
        <h1>解码你的飞书工作人格</h1>
        <p className="hero-copy">
          用本地授权的飞书协作数据，生成一份更像“工作现场行为画像”的 BSPI
          报告。数据只在本机流转，不做云端托管。
        </p>
        <div className="hero-badges">
          <span>Local First</span>
          <span>Read-only Feishu</span>
          <span>Structured AI Report</span>
        </div>
        <div className="hero-grid">
          <StatCard value="3" label="主界面阶段" />
          <StatCard value="20" label="人格候选" />
          <StatCard value="30d" label="默认分析窗口" />
        </div>
      </div>

      <div className="auth-panel">
        <div className="glass-card auth-card">
          <div className="card-header">
            <div>
              <div className="eyebrow">Connection Flow</div>
              <h2>连接与授权</h2>
            </div>
            <span className={`status-pill status-${statusData?.status || 'created'}`}>
              {friendlyStatus(statusData?.status || 'created')}
            </span>
          </div>

          <div className="step-stack">
            <StepRow
              active
              title="本地运行"
              description={sessionId ? `Session ${sessionId.slice(0, 8)} 已创建。` : '正在初始化本地会话。'}
            />
            <StepRow
              active={Boolean(statusData && statusOrder.indexOf(statusData.status) >= 1)}
              title="飞书配置 / 登录"
              description={statusData?.progress.label || '等待连接飞书。'}
            />
            <StepRow
              active={Boolean(statusData && statusOrder.indexOf(statusData.status) >= 3)}
              title="AI 深度分析"
              description={ready ? '当前会话已就绪，可以开始生成报告。' : '完成授权后即可开始。'}
            />
          </div>

          <div className="action-cluster">
            <button
              className="primary-button"
              disabled={booting || busyAction === 'login'}
              onClick={ready ? onAnalyze : onConnect}
            >
              {ready
                ? busyAction === 'analyze'
                  ? '正在提交分析...'
                  : '开启 AI 深度解析'
                : busyAction === 'login'
                  ? '正在准备授权...'
                  : '连接我的飞书'}
            </button>
            <button className="secondary-button" disabled={busyAction === 'session'} onClick={onReset}>
              新建会话
            </button>
            {statusData?.status === 'failed' ? (
              <button className="ghost-link" onClick={ready ? onAnalyze : onConnect}>
                立即重试
              </button>
            ) : null}
          </div>

          {verificationURL ? (
            <div className="link-card">
              <div className="eyebrow">Verification Link</div>
              <a href={verificationURL} rel="noreferrer" target="_blank">
                在新窗口中继续飞书授权
              </a>
            </div>
          ) : null}

          {error ? <div className="error-banner">{error}</div> : null}
          <div className="micro-note">
            首次使用可能会看到应用配置引导；后续会优先复用本地可刷新登录态。
          </div>
        </div>
      </div>
    </section>
  )
}

interface AnalysisPageProps {
  busyAction: string | null
  displayedProgress: number
  events: SessionStatusResponse['events']
  statusData: SessionStatusResponse
}

function AnalysisPage(props: AnalysisPageProps) {
  const { busyAction, displayedProgress, events, statusData } = props
  return (
    <section className="analysis-layout">
      <div className="analysis-core glass-card">
        <div className="orb-shell">
          <div className="orb-pulse" />
          <div className="orb-ring" />
          <div className="orb-text">
            <span>AI</span>
            <strong>{statusData.progress.percent}%</strong>
          </div>
        </div>
        <div className="analysis-copy">
          <div className="eyebrow">Analysis Runtime</div>
          <h2>{statusData.progress.label}</h2>
          <p>
            正在把聊天、文档、日程、任务、邮件和会议等授权数据整合成一份结构化
            BSPI 画像。保持当前页面开启，结果准备好后会自动切换。
          </p>
        </div>
        <div className="progress-track">
          <div className="progress-fill" style={{ width: `${displayedProgress}%` }} />
        </div>
        <div className="analysis-footnote">
          {busyAction === 'analyze' ? '分析任务已发出，正在等待后端进入稳定处理。' : '报告生成中'}
        </div>
      </div>

      <div className="timeline-card glass-card">
        <div className="card-header">
          <div>
            <div className="eyebrow">Timeline</div>
            <h3>状态流转播报</h3>
          </div>
          <span className="terminal-dot" />
        </div>
        <div className="timeline-list">
          {events.map((event) => (
            <div className="timeline-item" key={`${event.timestamp}-${event.stage}-${event.label}`}>
              <div className="timeline-meta">
                <span>{friendlyStatus(event.stage as SessionStatus)}</span>
                <span>{new Date(event.timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
              </div>
              <div className="timeline-label">{event.label}</div>
            </div>
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
  const { busyAction, error, posterNotice, report, reportRef, sessionId, onAnalyze, onDownloadPoster, onReset } = props
  const persona = report.primary_persona
  return (
    <section className="result-layout">
      <header className="result-header">
        <div>
          <div className="eyebrow">BSPI 2026</div>
          <h2>你的工作人格画像已生成</h2>
        </div>
        <div className="header-actions">
          <button className="secondary-button" onClick={onReset}>
            新建会话
          </button>
          <a className="ghost-link" href={`/api/sessions/${sessionId}/report`} rel="noreferrer" target="_blank">
            打开兼容版 HTML 报告
          </a>
        </div>
      </header>

      <div className="report-grid" ref={reportRef}>
        <article className="glass-card hero-report-card">
          <div className="hero-report-copy">
            <div className="eyebrow">Primary Persona</div>
            <h1>
              {persona.chinese_label}
              <span>{persona.shorthand}</span>
            </h1>
            <p className="one-liner">{persona.one_liner}</p>
            <div className="tag-row">
              {report.highlight_tags.map((tag) => (
                <span className="tag-pill" key={tag}>
                  {tag}
                </span>
              ))}
            </div>
          </div>
          <img alt={persona.shorthand} className="persona-art" src={persona.image_url} />
        </article>

        <article className="glass-card bento-card">
          <div className="card-header">
            <div>
              <div className="eyebrow">Behavior Vectors</div>
              <h3>行为信号</h3>
            </div>
            <span className="metric-chip">{report.analysis.confidence}</span>
          </div>
          <div className="vector-stack">
            {report.behavior_vectors.map((vector) => (
              <div className="vector-card" key={vector.label}>
                <div className="vector-title">
                  <span>{vector.label}</span>
                  <strong>{vector.score}</strong>
                </div>
                <div className="vector-poles">
                  <span>{vector.left_pole}</span>
                  <span>{vector.right_pole}</span>
                </div>
                <div className="vector-bar-shell">
                  <div className="vector-bar-fill" style={{ width: `${vector.score}%` }} />
                </div>
                <p>{vector.summary}</p>
              </div>
            ))}
          </div>
        </article>

        <article className="glass-card bento-card insight-card">
          <div className="eyebrow">AI Insight</div>
          <h3>深度洞察</h3>
          <p className="summary-text">{report.analysis.summary}</p>
          <dl className="insight-grid">
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
              <dt>官方定义</dt>
              <dd>{persona.canonical_description}</dd>
            </div>
          </dl>
        </article>

        <article className="glass-card bento-card evidence-card">
          <div className="eyebrow">Evidence & Coverage</div>
          <h3>证据与覆盖</h3>
          <ul className="evidence-list">
            {report.analysis.evidence.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
          <div className="coverage-box">
            <strong>数据覆盖</strong>
            <p>{report.coverage.summary}</p>
            <div className="coverage-meta">
              <span>已纳入：{report.coverage.successful_domains.join(' / ') || '无'}</span>
              <span>未纳入：{report.coverage.failed_domains.join(' / ') || '无'}</span>
            </div>
          </div>
        </article>
      </div>

      <footer className="result-toolbar glass-card">
        <div>
          <div className="eyebrow">Local Export</div>
          <p>{report.share_card.disclaimer_short}</p>
        </div>
        <div className="toolbar-actions">
          <button className="secondary-button" disabled={busyAction === 'analyze'} onClick={onAnalyze}>
            重新生成
          </button>
          <button className="primary-button" disabled={busyAction === 'poster'} onClick={onDownloadPoster}>
            {busyAction === 'poster' ? '正在导出...' : '保存专属报告'}
          </button>
        </div>
      </footer>

      {posterNotice ? <div className="notice-banner">{posterNotice}</div> : null}
      {error ? <div className="error-banner result-error">{error}</div> : null}
    </section>
  )
}

function StatCard(props: { value: string; label: string }) {
  return (
    <div className="mini-stat">
      <strong>{props.value}</strong>
      <span>{props.label}</span>
    </div>
  )
}

function StepRow(props: { active: boolean; title: string; description: string }) {
  return (
    <div className={`step-row ${props.active ? 'active' : ''}`}>
      <span className="step-dot" />
      <div>
        <strong>{props.title}</strong>
        <p>{props.description}</p>
      </div>
    </div>
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
      return '采集中'
    case 'analyzing':
      return '分析中'
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
