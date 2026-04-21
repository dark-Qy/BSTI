import html2canvas from 'html2canvas'
import { startTransition, useEffect, useRef, useState } from 'react'
import type { RefObject } from 'react'
import { LandingPage } from './landing-auth'
import {
  RequestError,
  analyzeSession,
  createSession,
  getReportData,
  getSessionStatus,
  startLogin,
} from './api'
import {
  clearPersistedSessionId,
  loadPersistedSessionId,
  persistSessionId,
} from './session-persistence'
import type {
  EvidenceItem,
  InteractionTarget,
  ReportData,
  SessionEvent,
  SessionStatus,
  SessionStatusResponse,
} from './types'

function isStoppedStatus(status: SessionStatus) {
  return ['authenticated', 'collecting', 'analyzing', 'done', 'auth_failed', 'analysis_failed', 'failed'].includes(status)
}

function isTerminalStatus(status: SessionStatus) {
  return ['done', 'auth_failed', 'analysis_failed', 'failed'].includes(status)
}

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

      if (isStoppedStatus(next.status)) {
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

  async function restoreSession(candidateSessionId: string) {
    setSessionId(candidateSessionId)

    try {
      const status = await getSessionStatus(candidateSessionId)
      setStatusData(status)
      setUiError('')
      clearAuthorizationReturnCheck()
      return true
    } catch (error) {
      if (error instanceof RequestError && error.status === 404) {
        clearPersistedSessionId()
        setSessionId('')
        setStatusData(null)
        return false
      }

      setUiError(toMessage(error))
      setStatusData(null)
      return true
    }
  }

  async function createFreshSession() {
    const created = await createSession()
    persistSessionId(created.session_id)
    setSessionId(created.session_id)
    const status = await getSessionStatus(created.session_id)
    setStatusData(status)
    setUiError('')
    clearAuthorizationReturnCheck()
  }

  async function bootstrapSession(options?: { forceNew?: boolean }) {
    const forceNew = options?.forceNew ?? false

    setBooting(true)
    setBusyAction('session')
    setUiError('')
    setPosterNotice('')
    setReportData(null)
    setStatusData(null)

    try {
      if (forceNew) {
        clearPersistedSessionId()
        setSessionId('')
      } else {
        const persistedSessionId = loadPersistedSessionId()
        if (persistedSessionId) {
          const restored = await restoreSession(persistedSessionId)
          if (restored) {
            return
          }
        }
      }

      await createFreshSession()
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
    if (isTerminalStatus(statusData.status)) {
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
      if (['auth_failed', 'analysis_failed', 'failed'].includes(statusData.status)) {
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
            onReset={() => void bootstrapSession({ forceNew: true })}
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
            onReset={() => void bootstrapSession({ forceNew: true })}
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
  const interactionInsights = report.interaction_insights ?? {
    relationship_summary: '',
    core_collaborators: [],
    frequent_people: [],
    frequent_chats: [],
  }
  const { frontDisclaimer, footerDisclaimer } = splitDisclaimers(
    report.share_card.disclaimer_short,
    report.analysis.disclaimer,
  )
  const contrastSignals = report.contrast_signals ?? []

  return (
    <section className="result-page">
      <header className="result-header">
        <div className="result-brand">BSTI 2026</div>
        <div className="result-header-actions">
          <button className="secondary-button" onClick={onReset}>
            切换账号 / 新建会话
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

          <section className="editorial-section report-region">
            <div className="section-kicker">Work Profile</div>
            <h2>工作画像</h2>
            <dl className="insight-list">
              <div>
                <dt>负责范围</dt>
                <dd>{report.work_profile.responsibility_scope}</dd>
              </div>
              <div>
                <dt>典型路径</dt>
                <dd>{report.work_profile.typical_workflow}</dd>
              </div>
              <div>
                <dt>文档风格</dt>
                <dd>{report.work_profile.doc_writing_style}</dd>
              </div>
              <div>
                <dt>决策模式</dt>
                <dd>{report.work_profile.decision_making_pattern}</dd>
              </div>
              <div>
                <dt>领域关键词</dt>
                <dd>{report.work_profile.tech_stack_or_domain.join(' / ')}</dd>
              </div>
              <div>
                <dt>输出结构</dt>
                <dd>{report.output_style.doc_structure_preference}</dd>
              </div>
              <div>
                <dt>细节密度</dt>
                <dd>{report.output_style.detail_level}</dd>
              </div>
              <div>
                <dt>邮件回复</dt>
                <dd>{report.output_style.email_reply_pattern}</dd>
              </div>
              <div>
                <dt>群聊风格</dt>
                <dd>{report.output_style.chat_reply_pattern}</dd>
              </div>
              <div>
                <dt>会议角色</dt>
                <dd>{report.output_style.meeting_behavior}</dd>
              </div>
            </dl>
          </section>

          <section className="editorial-section report-region">
            <div className="section-kicker">Expression</div>
            <h2>表达指纹</h2>
            <dl className="insight-list">
              <div>
                <dt>高频短语</dt>
                <dd>{report.expression_fingerprint.catchphrases.join(' / ')}</dd>
              </div>
              <div>
                <dt>术语习惯</dt>
                <dd>{report.expression_fingerprint.jargon.join(' / ')}</dd>
              </div>
              <div>
                <dt>句式特征</dt>
                <dd>{report.expression_fingerprint.sentence_pattern}</dd>
              </div>
              <div>
                <dt>Emoji 习惯</dt>
                <dd>{report.expression_fingerprint.emoji_habit}</dd>
              </div>
              <div>
                <dt>正式程度</dt>
                <dd>{report.expression_fingerprint.formality_spectrum}</dd>
              </div>
              <div>
                <dt>回复节奏</dt>
                <dd>{report.expression_fingerprint.reply_speed_pattern}</dd>
              </div>
              <div>
                <dt>分歧表达</dt>
                <dd>{report.expression_fingerprint.conflict_expression}</dd>
              </div>
            </dl>
          </section>

          <section className="editorial-section report-region">
            <div className="section-kicker">Knowledge</div>
            <h2>知识信号</h2>
            <div className="interaction-grid">
              <SignalList title="明确观点" items={report.knowledge_signals.explicit_opinions} />
              <SignalList title="踩坑经验" items={report.knowledge_signals.learned_lessons} />
              <SignalList title="反复强调" items={report.knowledge_signals.repeated_concerns} />
              <SignalList title="常引用来源" items={report.knowledge_signals.reference_sources} />
            </div>
          </section>

          <section className="editorial-section interaction-section report-region report-region-interaction">
            <div className="section-kicker">Connections</div>
            <h2>互动关系</h2>
            <p className="interaction-summary">
              {interactionInsights.relationship_summary || '当前授权数据里还没有形成稳定的互动关系信号。'}
            </p>
            <div className="interaction-grid">
              <InteractionList title="核心协作对象" items={interactionInsights.core_collaborators} />
              <InteractionList title="高频互动人" items={interactionInsights.frequent_people} />
              <InteractionList title="高频群聊" items={interactionInsights.frequent_chats} />
            </div>
          </section>

          <section className="editorial-section evidence-section report-region report-region-evidence">
            <div className="section-kicker">Coverage</div>
            <h2>证据与覆盖</h2>
            <ul className="evidence-list">
              {report.analysis.evidence.map((item) => (
                <EvidenceRow item={item} key={`${item.behavior}-${item.strength}`} />
              ))}
            </ul>
            {contrastSignals.length > 0 ? (
              <>
                <h3>跨域反差点</h3>
                <ul className="evidence-list">
                  {contrastSignals.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </>
            ) : null}
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

function InteractionList(props: { title: string; items: InteractionTarget[] }) {
  const { title, items } = props
  return (
    <article className="interaction-card">
      <h3>{title}</h3>
      {items.length === 0 ? (
        <p className="interaction-empty">当前授权数据里还没有形成稳定信号。</p>
      ) : (
        <ul className="interaction-list">
          {items.map((item) => (
            <li key={`${title}-${item.display_name}-${item.summary}`}>
              <div className="interaction-item-head">
                <strong>{item.display_name}</strong>
              </div>
              <p>{item.summary}</p>
              <small>{item.evidence}</small>
            </li>
          ))}
        </ul>
      )}
    </article>
  )
}

function SignalList(props: { title: string; items: string[] }) {
  const { title, items } = props
  return (
    <article className="interaction-card">
      <h3>{title}</h3>
      {items.length === 0 ? (
        <p className="interaction-empty">当前授权数据里还没有形成稳定信号。</p>
      ) : (
        <ul className="interaction-list">
          {items.map((item) => (
            <li key={`${title}-${item}`}>
              <p>{item}</p>
            </li>
          ))}
        </ul>
      )}
    </article>
  )
}

function EvidenceRow(props: { item: EvidenceItem }) {
  const { item } = props
  return (
    <li>
      <strong>{item.domains.join(' / ')}</strong>
      {`: ${item.behavior} | ${item.strength}`}
      {item.is_cross_domain ? ' | 跨域一致' : ''}
      {item.is_distinctive ? ' | 区分性信号' : ''}
    </li>
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
    case 'auth_failed':
      return '授权失败'
    case 'analysis_failed':
      return '分析失败'
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
