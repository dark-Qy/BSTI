import { render, screen, waitFor } from '@testing-library/react'
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
    vi.restoreAllMocks()
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

    expect(await screen.findByText('洞悉你的工作。')).toBeInTheDocument()
    expect(screen.getByText('重塑你的人格。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '连接我的飞书' })).toBeInTheDocument()
    expect(screen.queryByText('主界面阶段')).not.toBeInTheDocument()
    expect(screen.queryByText('Connection Flow')).not.toBeInTheDocument()
  })

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

    render(<App />)

    expect(await screen.findByText('正在采集飞书协作数据')).toBeInTheDocument()
    expect(screen.getByText('数据处理中...')).toBeInTheDocument()
    expect(screen.getByText('状态流转')).toBeInTheDocument()
    expect(screen.queryByText('Timeline')).not.toBeInTheDocument()
  })

  it('renders the result page as an editorial report with confidence in the footer copy', async () => {
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

    render(<App />)

    expect(await screen.findByText('刨坟者')).toBeInTheDocument()
    expect(screen.getByText('行为解析')).toBeInTheDocument()
    expect(screen.getByText('while(true) { 为什么？ }')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText(/置信度 0.86/)).toBeInTheDocument()
    })
    expect(screen.queryByText('Primary Persona')).not.toBeInTheDocument()
    expect(screen.queryByText('Behavior Vectors')).not.toBeInTheDocument()
  })
})
