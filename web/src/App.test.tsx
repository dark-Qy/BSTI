import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'

type MockResponse = Record<string, unknown>

const originalOpen = window.open

function jsonResponse(payload: MockResponse) {
  return Promise.resolve(
    new Response(JSON.stringify(payload), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
}

function errorResponse(status: number, payload: MockResponse) {
  return Promise.resolve(
    new Response(JSON.stringify(payload), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
}

describe('App', () => {
  beforeEach(() => {
    window.open = vi.fn()
    window.localStorage.clear()
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    vi.useRealTimers()
    window.open = originalOpen
  })

  it('renders the new Apple-style landing headline and focused auth action', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-1', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-1/status')) {
        return jsonResponse({
          session_id: 'session-1',
          status: 'created',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
          events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-11T14:00:00Z' }],
          next_action: 'connect_feishu',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('BSTI 2026')).toBeInTheDocument()
    expect(await screen.findByText('洞悉你的工作。')).toBeInTheDocument()
    expect(screen.getByText('重塑你的人格。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '连接我的飞书' })).toBeInTheDocument()
    expect(screen.queryByText('若已存在可复用的飞书应用配置，将自动跳过第一步配置。')).not.toBeInTheDocument()
    expect(screen.queryByText('主界面阶段')).not.toBeInTheDocument()
    expect(screen.queryByText('Connection Flow')).not.toBeInTheDocument()
  })

  it('restores a persisted session id from localStorage on startup', async () => {
    window.localStorage.setItem('bsti.activeSessionId', 'session-restored')

    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions/session-restored/status')) {
        return jsonResponse({
          session_id: 'session-restored',
          status: 'authenticated',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'authenticated', label: '授权完成，准备开始分析', percent: 45 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-20T10:00:00Z' },
            { stage: 'authenticated', label: '飞书授权完成', percent: 45, timestamp: '2026-04-20T10:01:00Z' },
          ],
          next_action: 'start_analysis',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('已就绪，可开始分析')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '开启 AI 深度解析' })).toBeInTheDocument()
    expect(fetchSpy).not.toHaveBeenCalledWith('/api/sessions', { method: 'POST' })
    expect(window.localStorage.getItem('bsti.activeSessionId')).toBe('session-restored')
  })

  it('creates and persists a new session id when no local session exists', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-created', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-created/status')) {
        return jsonResponse({
          session_id: 'session-created',
          status: 'created',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
          events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-20T10:02:00Z' }],
          next_action: 'connect_feishu',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByRole('button', { name: '连接我的飞书' })).toBeInTheDocument()
    expect(window.localStorage.getItem('bsti.activeSessionId')).toBe('session-created')
  })

  it('replaces a missing persisted session by clearing storage and creating a new one', async () => {
    window.localStorage.setItem('bsti.activeSessionId', 'session-missing')

    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions/session-missing/status')) {
        return errorResponse(404, { error: 'open /missing/session.json: no such file or directory' })
      }
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-fresh', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-fresh/status')) {
        return jsonResponse({
          session_id: 'session-fresh',
          status: 'created',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
          events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-20T10:03:00Z' }],
          next_action: 'connect_feishu',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByRole('button', { name: '连接我的飞书' })).toBeInTheDocument()
    expect(window.localStorage.getItem('bsti.activeSessionId')).toBe('session-fresh')
  })

  it('clears the persisted session and creates a new one when switching account', async () => {
    window.localStorage.setItem('bsti.activeSessionId', 'session-old')

    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions/session-old/status')) {
        return jsonResponse({
          session_id: 'session-old',
          status: 'created',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
          events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-20T10:04:00Z' }],
          next_action: 'connect_feishu',
        })
      }
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-new', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-new/status')) {
        return jsonResponse({
          session_id: 'session-new',
          status: 'created',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
          events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-20T10:05:00Z' }],
          next_action: 'connect_feishu',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByRole('button', { name: '连接我的飞书' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '切换账号 / 新建会话' }))

    await waitFor(() => {
      expect(window.localStorage.getItem('bsti.activeSessionId')).toBe('session-new')
    })
  })

  it('shows the preconfigured-app hint when server credentials skip the first step', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-preconfigured', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-preconfigured/status')) {
        return jsonResponse({
          session_id: 'session-preconfigured',
          status: 'created',
          app_config_required: false,
          report_ready: false,
          progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
          events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-11T14:00:00Z' }],
          next_action: 'connect_feishu',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('连接飞书')).toBeInTheDocument()
    expect(screen.getByText('服务端已预置飞书应用配置，本次将直接进入飞书授权。')).toBeInTheDocument()
    expect(screen.getByText('已预置')).toBeInTheDocument()
  })

  it('makes the second authorization step visually distinct after config is complete', async () => {
    let loginCalls = 0

    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-login', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-login/status')) {
        return jsonResponse({
          session_id: 'session-login',
          status: 'login_pending',
          app_config_required: true,
          verification_url: 'https://verify.example',
          report_ready: false,
          progress: { stage: 'login_pending', label: '等待完成飞书授权', percent: 32 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-14T09:00:00Z' },
            { stage: 'config_pending', label: '等待完成飞书应用配置', percent: 20, timestamp: '2026-04-14T09:01:00Z' },
            { stage: 'login_pending', label: '等待完成飞书授权', percent: 32, timestamp: '2026-04-14T09:02:00Z' },
          ],
          next_action: 'complete_authorization',
        })
      }
      if (url.endsWith('/api/sessions/session-login/login')) {
        loginCalls += 1
        return jsonResponse({
          status: 'login_pending',
          verification_url: 'https://wrong-step.example',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('第二步：完成飞书授权')).toBeInTheDocument()
    expect(screen.getByText('当前处于第 2 步')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '去完成第二步授权' })).toBeInTheDocument()
    expect(screen.getByText('飞书应用配置')).toBeInTheDocument()
    expect(screen.getByText('飞书授权')).toBeInTheDocument()
    expect(screen.getByText('已完成')).toBeInTheDocument()
    expect(screen.getByText('进行中')).toBeInTheDocument()
    expect(screen.queryByText('在新窗口中继续飞书授权')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '去完成第二步授权' }))

    await waitFor(() => {
      expect(window.open).toHaveBeenCalledWith(
        'https://verify.example',
        '_blank',
        'noopener,noreferrer',
      )
    })
    expect(loginCalls).toBe(0)
  })

  it('uses the current config link as the only action during the first authorization step', async () => {
    let loginCalls = 0

    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-config', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-config/status')) {
        return jsonResponse({
          session_id: 'session-config',
          status: 'config_pending',
          app_config_required: true,
          verification_url: 'https://config.example',
          report_ready: false,
          progress: { stage: 'config_pending', label: '等待完成飞书应用配置', percent: 20 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-14T09:00:00Z' },
            { stage: 'config_pending', label: '等待完成飞书应用配置', percent: 20, timestamp: '2026-04-14T09:01:00Z' },
          ],
          next_action: 'complete_authorization',
        })
      }
      if (url.endsWith('/api/sessions/session-config/login')) {
        loginCalls += 1
        return jsonResponse({
          status: 'config_pending',
          verification_url: 'https://wrong-config.example',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('第一步：完成飞书应用配置')).toBeInTheDocument()
    expect(screen.queryByText('在新窗口中继续飞书授权')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '继续完成第一步配置' }))

    await waitFor(() => {
      expect(window.open).toHaveBeenCalledWith(
        'https://config.example',
        '_blank',
        'noopener,noreferrer',
      )
    })
    expect(loginCalls).toBe(0)
  })

  it('shows the preconfigured first step when server credentials skip configuration', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-preconfigured-login', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-preconfigured-login/status')) {
        return jsonResponse({
          session_id: 'session-preconfigured-login',
          status: 'login_pending',
          app_config_required: false,
          verification_url: 'https://verify.example/preconfigured',
          report_ready: false,
          progress: { stage: 'login_pending', label: '等待完成飞书授权', percent: 32 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-14T10:00:00Z' },
            { stage: 'login_pending', label: '等待完成飞书授权', percent: 32, timestamp: '2026-04-14T10:02:00Z' },
          ],
          next_action: 'complete_authorization',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('第二步：完成飞书授权')).toBeInTheDocument()
    expect(screen.getByText('服务端已预置飞书应用配置，本次只需完成飞书授权。')).toBeInTheDocument()
    expect(screen.getByText('已预置')).toBeInTheDocument()
  })

  it('switches to a return-checking state after browser auth resumes and then auto-upgrades to ready', async () => {
    let statusCalls = 0

    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-return', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-return/login')) {
        return jsonResponse({
          status: 'login_pending',
          verification_url: 'https://verify.example/return',
        })
      }
      if (url.endsWith('/api/sessions/session-return/status')) {
        statusCalls += 1
        if (statusCalls === 1) {
          return jsonResponse({
          session_id: 'session-return',
          status: 'created',
          app_config_required: true,
          report_ready: false,
            progress: { stage: 'created', label: '等待连接飞书', percent: 5 },
            events: [{ stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-14T11:00:00Z' }],
            next_action: 'connect_feishu',
          })
        }
        if (statusCalls <= 3) {
          return jsonResponse({
            session_id: 'session-return',
            status: 'login_pending',
            app_config_required: true,
            verification_url: 'https://verify.example/return',
            report_ready: false,
            progress: { stage: 'login_pending', label: '等待完成飞书授权', percent: 32 },
            events: [
              { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-14T11:00:00Z' },
              { stage: 'login_pending', label: '等待完成飞书授权', percent: 32, timestamp: '2026-04-14T11:01:00Z' },
            ],
            next_action: 'complete_authorization',
          })
        }
        return jsonResponse({
          session_id: 'session-return',
          status: 'authenticated',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'authenticated', label: '授权完成，准备开始分析', percent: 45 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-14T11:00:00Z' },
            { stage: 'login_pending', label: '等待完成飞书授权', percent: 32, timestamp: '2026-04-14T11:01:00Z' },
            { stage: 'authenticated', label: '飞书授权完成', percent: 45, timestamp: '2026-04-14T11:02:00Z' },
          ],
          next_action: 'start_analysis',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    await screen.findByRole('button', { name: '连接我的飞书' })

    fireEvent.click(screen.getByRole('button', { name: '连接我的飞书' }))

    await waitFor(() => {
      expect(window.open).toHaveBeenCalledWith(
        'https://verify.example/return',
        '_blank',
        'noopener,noreferrer',
      )
    })

    await screen.findByText('第二步：完成飞书授权')

    act(() => {
      window.dispatchEvent(new Event('focus'))
    })

    expect(await screen.findByText('正在检查飞书授权结果')).toBeInTheDocument()
    expect(
      screen.getByText('如果你刚刚已在浏览器完成授权，系统会在几秒内自动更新，无需再次点击。'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '正在确认授权结果...' })).toBeDisabled()
    expect(screen.getByText('确认中')).toBeInTheDocument()

    await waitFor(
      () => {
        expect(screen.getByText('已就绪，可开始分析')).toBeInTheDocument()
      },
      { timeout: 2500 },
    )
    expect(screen.getByRole('button', { name: '开启 AI 深度解析' })).toBeInTheDocument()
  }, 8000)

  it('renders the loading page with a focused progress phrase and timeline labels', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-2', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-2/status')) {
        return jsonResponse({
          session_id: 'session-2',
          status: 'collecting',
          app_config_required: true,
          report_ready: false,
          progress: { stage: 'collecting', label: '正在采集授权范围内的飞书数据', percent: 66 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-11T14:00:00Z' },
            { stage: 'collecting', label: '正在采集授权范围内的飞书数据', percent: 66, timestamp: '2026-04-11T14:01:00Z' },
          ],
          next_action: 'wait',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    const { container } = render(<App />)

    expect(await screen.findByText('正在采集飞书协作数据')).toBeInTheDocument()
    expect(screen.getByText('数据处理中...')).toBeInTheDocument()
    expect(screen.getByText('状态流转')).toBeInTheDocument()
    expect(screen.queryByText('Timeline')).not.toBeInTheDocument()
    const currentItem = container.querySelector('.timeline-item.is-current')
    const completedItem = container.querySelector('.timeline-item.is-completed')
    expect(currentItem).not.toBeNull()
    expect(completedItem).not.toBeNull()
    expect(currentItem?.textContent).toContain('采集数据')
  })

  it('renders the result page with an interaction insights section and footer coverage kept separate', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-3', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-3/status')) {
        return jsonResponse({
          session_id: 'session-3',
          status: 'done',
          app_config_required: true,
          report_ready: true,
          progress: { stage: 'done', label: '报告已生成', percent: 100 },
          events: [{ stage: 'done', label: '报告已生成，可以查看结果', percent: 100, timestamp: '2026-04-11T14:02:00Z' }],
          next_action: 'view_report',
          primary_persona: {
            shorthand: 'CORE',
            chinese_label: '刨坟者',
            image_url: '/assets/photos/CORE.png',
            byte_style_dimension: '求真务实',
            analysis_dimension: '本质洞察力',
            one_liner: 'while(true) { 为什么？ }',
            canonical_description: 'CORE 会持续追问根因。',
          },
        })
      }
      if (url.endsWith('/api/sessions/session-3/report-data')) {
        return jsonResponse({
          status: 'done',
          report: {
            primary_persona: {
              shorthand: 'CORE',
              chinese_label: '刨坟者',
              image_url: '/assets/photos/CORE.png',
              byte_style_dimension: '求真务实',
              analysis_dimension: '本质洞察力',
              one_liner: 'while(true) { 为什么？ }',
              canonical_description: 'CORE 会持续追问根因。',
            },
            analysis: {
              summary: '会持续追问底层逻辑。',
              evidence: [
                {
                  domains: ['chat', 'docs'],
                  behavior: '跨域反复追问 why',
                  strength: '高频',
                  is_cross_domain: true,
                  is_distinctive: true,
                },
              ],
              communication_style: '偏向追问本质',
              work_preferences: '偏好深挖问题',
              blind_spots: '可能推进偏慢',
              confidence: 0.86,
              disclaimer: '仅基于授权数据的行为风格观察。',
            },
            work_profile: {
              responsibility_scope: '负责复杂问题拆解和方案推进。',
              typical_workflow: '先追根因，再收敛方案，再推进执行。',
              doc_writing_style: '层级清晰，偏问题驱动。',
              decision_making_pattern: '先看证据再形成结论。',
              tech_stack_or_domain: ['架构设计', '问题排查'],
            },
            expression_fingerprint: {
              catchphrases: ['先看根因', '这里再深挖一下'],
              jargon: ['根因', '链路'],
              sentence_pattern: '短句追问，结论后置。',
              emoji_habit: '几乎不用 emoji',
              formality_spectrum: '正式场景更克制',
              reply_speed_pattern: '关键问题响应快',
              conflict_expression: '通过连续提问表达质疑',
            },
            output_style: {
              doc_structure_preference: '分级标题 + 问题拆解',
              detail_level: '详尽',
              email_reply_pattern: '结论后补充细节',
              chat_reply_pattern: '群聊低频但关键时刻输出',
              meeting_behavior: '后半段集中输出结论',
            },
            knowledge_signals: {
              explicit_opinions: ['没有根因分析就不要急着推进。'],
              learned_lessons: ['问题定义错了，后面全是返工。'],
              repeated_concerns: ['链路复杂度'],
              reference_sources: ['线上数据', '问题单'],
            },
            highlight_tags: ['追问根因', '深度分析'],
            behavior_vectors: [
              { label: '协作方式', left_pole: '独立成局', right_pole: '高频协同', score: 35, summary: '更偏独立深挖。' },
              { label: '表达风格', left_pole: '克制压缩', right_pole: '高频输出', score: 46, summary: '表达偏克制。' },
              { label: '决策路径', left_pole: '证据校准', right_pole: '直觉快判', score: 22, summary: '先看证据。' },
              { label: '推进节奏', left_pole: '稳态推进', right_pole: '高压突进', score: 39, summary: '节奏稳。' },
              { label: '信息处理', left_pole: '深度聚焦', right_pole: '广度扫描', score: 28, summary: '更偏单线程深挖。' },
              { label: '风险态度', left_pole: '防御优先', right_pole: '进攻优先', score: 31, summary: '先考虑出错成本。' },
            ],
            interaction_insights: {
              relationship_summary: '和核心协作者保持高密度 direct message 互动，同时在跨团队群承担信息收敛。',
              core_collaborators: [
                { display_name: '小李', identifier: 'ou_core_1', summary: '方案推进搭档', evidence: '近 30 天 direct message 高频往返。' },
              ],
              frequent_people: [
                { display_name: '小李', identifier: 'ou_core_1', summary: '高频技术讨论对象', evidence: '问题拆解和排期沟通都集中出现。' },
              ],
              frequent_chats: [
                { display_name: '跨团队项目群', identifier: 'oc_chat_1', summary: '高频同步群', evidence: '多次出现关键结论同步。' },
              ],
            },
            contrast_signals: ['群聊里发言不多，但文档里会完整展开推理链路。'],
            share_card: {
              title: '刨坟者 / CORE',
              subtitle: 'while(true) { 为什么？ }',
              image_url: '/assets/photos/CORE.png',
              disclaimer_short: '仅基于授权数据的行为风格观察。',
            },
            coverage: {
              successful_domains: ['chat', 'docs'],
              failed_domains: ['mail'],
              summary: '已覆盖 2 个数据域。',
            },
          },
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    const { container } = render(<App />)

    expect(await screen.findByText('刨坟者')).toBeInTheDocument()
    expect(screen.getByText('行为解析')).toBeInTheDocument()
    expect(screen.getByText('深度洞察')).toBeInTheDocument()
    expect(screen.getByText('工作画像')).toBeInTheDocument()
    expect(screen.getByText('表达指纹')).toBeInTheDocument()
    expect(screen.getByText('知识信号')).toBeInTheDocument()
    expect(screen.getByText('互动关系')).toBeInTheDocument()
    expect(screen.getByText('证据与覆盖')).toBeInTheDocument()
    expect(screen.getByText('while(true) { 为什么？ }')).toBeInTheDocument()
    expect(screen.getByText('跨团队项目群')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText(/置信度 0.86/)).toBeInTheDocument()
    })
    expect(screen.getAllByText(/置信度 0.86/)).toHaveLength(1)
    expect(screen.getAllByText('仅基于授权数据的行为风格观察。')).toHaveLength(1)
    expect(container.querySelectorAll('.report-stack')).toHaveLength(0)
    expect(screen.queryByText('ou_core_1')).not.toBeInTheDocument()
    expect(screen.queryByText('oc_chat_1')).not.toBeInTheDocument()

    const footer = container.querySelector('.report-footer')
    expect(footer).not.toBeNull()
    expect(footer?.textContent).not.toContain('已覆盖 2 个数据域。')
    expect(footer?.textContent).not.toContain('已纳入：chat / docs')
    expect(footer?.textContent).not.toContain('未纳入：mail')

    expect(screen.queryByText('Primary Persona')).not.toBeInTheDocument()
    expect(screen.queryByText('Behavior Vectors')).not.toBeInTheDocument()
  })

  it('keeps the hero image directly under the title block instead of pushing it to the bottom', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-4', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-4/status')) {
        return jsonResponse({
          session_id: 'session-4',
          status: 'done',
          app_config_required: true,
          report_ready: true,
          progress: { stage: 'done', label: '报告已生成', percent: 100 },
          events: [{ stage: 'done', label: '报告已生成，可以查看结果', percent: 100, timestamp: '2026-04-12T09:02:00Z' }],
          next_action: 'view_report',
          primary_persona: {
            shorthand: 'CORE',
            chinese_label: '非常非常长的人格标题用于验证首屏图片完整可见',
            image_url: '/assets/photos/CORE.png',
            byte_style_dimension: '求真务实',
            analysis_dimension: '本质洞察力',
            one_liner: 'while(true) { 为什么？ }',
            canonical_description: 'CORE 会持续追问根因。',
          },
        })
      }
      if (url.endsWith('/api/sessions/session-4/report-data')) {
        return jsonResponse({
          status: 'done',
          report: {
            primary_persona: {
              shorthand: 'CORE',
              chinese_label: '非常非常长的人格标题用于验证首屏图片完整可见',
              image_url: '/assets/photos/CORE.png',
              byte_style_dimension: '求真务实',
              analysis_dimension: '本质洞察力',
              one_liner: 'while(true) { 为什么？ }',
              canonical_description: 'CORE 会持续追问根因。',
            },
            analysis: {
              summary: '会持续追问底层逻辑。',
              evidence: [
                {
                  domains: ['chat', 'docs'],
                  behavior: '跨域反复追问 why',
                  strength: '高频',
                  is_cross_domain: true,
                  is_distinctive: true,
                },
              ],
              communication_style: '偏向追问本质',
              work_preferences: '偏好深挖问题',
              blind_spots: '可能推进偏慢',
              confidence: 0.86,
              disclaimer: '仅基于授权数据的行为风格观察。',
            },
            work_profile: {
              responsibility_scope: '负责复杂问题拆解和方案推进。',
              typical_workflow: '先追根因，再收敛方案，再推进执行。',
              doc_writing_style: '层级清晰，偏问题驱动。',
              decision_making_pattern: '先看证据再形成结论。',
              tech_stack_or_domain: ['架构设计', '问题排查'],
            },
            expression_fingerprint: {
              catchphrases: ['先看根因', '这里再深挖一下'],
              jargon: ['根因', '链路'],
              sentence_pattern: '短句追问，结论后置。',
              emoji_habit: '几乎不用 emoji',
              formality_spectrum: '正式场景更克制',
              reply_speed_pattern: '关键问题响应快',
              conflict_expression: '通过连续提问表达质疑',
            },
            output_style: {
              doc_structure_preference: '分级标题 + 问题拆解',
              detail_level: '详尽',
              email_reply_pattern: '结论后补充细节',
              chat_reply_pattern: '群聊低频但关键时刻输出',
              meeting_behavior: '后半段集中输出结论',
            },
            knowledge_signals: {
              explicit_opinions: ['没有根因分析就不要急着推进。'],
              learned_lessons: ['问题定义错了，后面全是返工。'],
              repeated_concerns: ['链路复杂度'],
              reference_sources: ['线上数据', '问题单'],
            },
            highlight_tags: ['追问根因', '深度分析', '结构控'],
            behavior_vectors: [
              { label: '协作方式', left_pole: '独立成局', right_pole: '高频协同', score: 35, summary: '更偏独立深挖。' },
              { label: '表达风格', left_pole: '克制压缩', right_pole: '高频输出', score: 46, summary: '表达偏克制。' },
              { label: '决策路径', left_pole: '证据校准', right_pole: '直觉快判', score: 22, summary: '先看证据。' },
              { label: '推进节奏', left_pole: '稳态推进', right_pole: '高压突进', score: 39, summary: '节奏稳。' },
              { label: '信息处理', left_pole: '深度聚焦', right_pole: '广度扫描', score: 28, summary: '更偏单线程深挖。' },
              { label: '风险态度', left_pole: '防御优先', right_pole: '进攻优先', score: 31, summary: '先考虑出错成本。' },
            ],
            contrast_signals: ['群聊里发言不多，但文档里会完整展开推理链路。'],
            share_card: {
              title: '刨坟者 / CORE',
              subtitle: 'while(true) { 为什么？ }',
              image_url: '/assets/photos/CORE.png',
              disclaimer_short: '仅基于授权数据的行为风格观察。',
            },
            coverage: {
              successful_domains: ['chat', 'docs'],
              failed_domains: ['mail'],
              summary: '已覆盖 2 个数据域。',
            },
          },
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    const { container } = render(<App />)

    expect(await screen.findByText('非常非常长的人格标题用于验证首屏图片完整可见')).toBeInTheDocument()

    const heroRegion = container.querySelector('.report-region-hero')
    const heroCopy = container.querySelector('.hero-copy')
    const heroVisual = container.querySelector('.hero-visual')
    const insightRegion = container.querySelector('.report-region-insight')

    expect(heroRegion).not.toBeNull()
    expect(heroCopy).not.toBeNull()
    expect(heroVisual).not.toBeNull()
    expect(heroVisual?.querySelector('img.persona-art')).not.toBeNull()
    expect(heroRegion?.firstElementChild).toBe(heroCopy)
    expect(heroCopy?.nextElementSibling).toBe(heroVisual)

    expect(heroRegion?.textContent).not.toContain('while(true) { 为什么？ }')
    expect(heroRegion?.textContent).not.toContain('追问根因')
    expect(insightRegion?.textContent).toContain('while(true) { 为什么？ }')
    expect(insightRegion?.textContent).toContain('追问根因')
  })

  it('renders an auth failure as authorization retry instead of analysis retry', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-auth-failed', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-auth-failed/status')) {
        return jsonResponse({
          session_id: 'session-auth-failed',
          status: 'auth_failed',
          app_config_required: true,
          verification_url: 'https://verify.example/auth-failed',
          error: '飞书授权失败',
          report_ready: false,
          progress: { stage: 'auth_failed', label: '授权流程失败，请重新连接飞书', percent: 100 },
          events: [
            { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-21T10:00:00Z' },
            { stage: 'login_pending', label: '等待完成飞书授权', percent: 32, timestamp: '2026-04-21T10:01:00Z' },
            { stage: 'auth_failed', label: '飞书授权失败', percent: 100, timestamp: '2026-04-21T10:02:00Z' },
          ],
          next_action: 'complete_authorization',
        })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('授权失败')).toBeInTheDocument()
    expect(screen.getByText('第一步配置已完成，但第二步飞书授权未成功。请重新完成授权。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '重新完成第二步授权' })).toBeInTheDocument()
  })

  it('renders an analysis failure as analysis retry instead of authorization failure', async () => {
    let analyzeCalls = 0

    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-analysis-failed', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-analysis-failed/status')) {
        return jsonResponse({
          session_id: 'session-analysis-failed',
          status: analyzeCalls === 0 ? 'analysis_failed' : 'collecting',
          app_config_required: true,
          error: analyzeCalls === 0 ? 'modelhub response validation failed' : '',
          report_ready: false,
          progress: analyzeCalls === 0
            ? { stage: 'analysis_failed', label: '分析流程失败，可直接重试分析', percent: 100 }
            : { stage: 'collecting', label: '正在采集授权范围内的飞书数据', percent: 66 },
          events: analyzeCalls === 0
            ? [
                { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-21T11:00:00Z' },
                { stage: 'authenticated', label: '飞书授权完成', percent: 45, timestamp: '2026-04-21T11:01:00Z' },
                { stage: 'collecting', label: '正在采集授权范围内的飞书数据', percent: 66, timestamp: '2026-04-21T11:02:00Z' },
                { stage: 'analysis_failed', label: '分析流程执行失败', percent: 100, timestamp: '2026-04-21T11:03:00Z' },
              ]
            : [
                { stage: 'created', label: 'Session created', percent: 5, timestamp: '2026-04-21T11:00:00Z' },
                { stage: 'authenticated', label: '飞书授权完成', percent: 45, timestamp: '2026-04-21T11:01:00Z' },
                { stage: 'collecting', label: '正在采集授权范围内的飞书数据', percent: 66, timestamp: '2026-04-21T11:04:00Z' },
              ],
          next_action: analyzeCalls === 0 ? 'start_analysis' : 'wait',
        })
      }
      if (url.endsWith('/api/sessions/session-analysis-failed/analyze')) {
        analyzeCalls += 1
        return jsonResponse({ status: 'collecting' })
      }
      throw new Error(`Unhandled fetch: ${url}`)
    })

    render(<App />)

    expect(await screen.findByText('分析失败')).toBeInTheDocument()
    expect(screen.getByText('授权已完成，但分析流程执行失败。你可以直接重试分析，无需重新授权。')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '重新开始分析' }))

    await waitFor(() => {
      expect(analyzeCalls).toBe(1)
    })
  })
})
