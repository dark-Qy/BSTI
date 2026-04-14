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

describe('App', () => {
  beforeEach(() => {
    window.open = vi.fn()
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
    expect(screen.getByText('若已存在可复用的飞书应用配置，将自动跳过第一步配置。')).toBeInTheDocument()
    expect(screen.queryByText('主界面阶段')).not.toBeInTheDocument()
    expect(screen.queryByText('Connection Flow')).not.toBeInTheDocument()
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

  it('explains when the app configuration step was reused before entering authorization', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-reused', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-reused/status')) {
        return jsonResponse({
          session_id: 'session-reused',
          status: 'login_pending',
          verification_url: 'https://verify.example/reused',
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
    expect(screen.getByText('已复用现有飞书应用配置，本次只需完成飞书授权。')).toBeInTheDocument()
    expect(screen.getByText('已复用')).toBeInTheDocument()
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

  it('renders the result page as five explicit grid regions without footer coverage overlap', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/api/sessions')) {
        return jsonResponse({ session_id: 'session-3', status: 'created' })
      }
      if (url.endsWith('/api/sessions/session-3/status')) {
        return jsonResponse({
          session_id: 'session-3',
          status: 'done',
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
              evidence: ['跨域反复追问 why'],
              communication_style: '偏向追问本质',
              work_preferences: '偏好深挖问题',
              blind_spots: '可能推进偏慢',
              confidence: 0.86,
              disclaimer: '仅基于授权数据的行为风格观察。',
            },
            highlight_tags: ['追问根因', '深度分析'],
            behavior_vectors: [
              { label: '协作方式', left_pole: '独立成局', right_pole: '高频协同', score: 35, summary: '更偏独立深挖。' },
              { label: '表达风格', left_pole: '克制压缩', right_pole: '高频输出', score: 46, summary: '表达偏克制。' },
              { label: '决策路径', left_pole: '证据校准', right_pole: '直觉快判', score: 22, summary: '先看证据。' },
              { label: '推进节奏', left_pole: '稳态推进', right_pole: '高压突进', score: 39, summary: '节奏稳。' },
            ],
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
    expect(screen.getByText('证据与覆盖')).toBeInTheDocument()
    expect(screen.getByText('while(true) { 为什么？ }')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText(/置信度 0.86/)).toBeInTheDocument()
    })
    expect(screen.getAllByText(/置信度 0.86/)).toHaveLength(1)
    expect(screen.getAllByText('仅基于授权数据的行为风格观察。')).toHaveLength(1)
    expect(container.querySelectorAll('.report-stack')).toHaveLength(0)

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
              evidence: ['跨域反复追问 why'],
              communication_style: '偏向追问本质',
              work_preferences: '偏好深挖问题',
              blind_spots: '可能推进偏慢',
              confidence: 0.86,
              disclaimer: '仅基于授权数据的行为风格观察。',
            },
            highlight_tags: ['追问根因', '深度分析', '结构控'],
            behavior_vectors: [
              { label: '协作方式', left_pole: '独立成局', right_pole: '高频协同', score: 35, summary: '更偏独立深挖。' },
              { label: '表达风格', left_pole: '克制压缩', right_pole: '高频输出', score: 46, summary: '表达偏克制。' },
              { label: '决策路径', left_pole: '证据校准', right_pole: '直觉快判', score: 22, summary: '先看证据。' },
              { label: '推进节奏', left_pole: '稳态推进', right_pole: '高压突进', score: 39, summary: '节奏稳。' },
            ],
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
})
