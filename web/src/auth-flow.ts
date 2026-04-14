import type { SessionStatusResponse } from './types'

export type AuthFlowPhase =
  | 'created'
  | 'config_pending'
  | 'login_pending'
  | 'login_pending_reused'
  | 'checking_authorization_return'
  | 'authenticated'
  | 'failed_step_1'
  | 'failed_step_2'

export type AuthPrimaryAction = 'connect' | 'open_url' | 'analyze' | 'wait'

type AuthStepState = 'pending' | 'active' | 'done' | 'failed'

export interface AuthStepViewModel {
  number: number
  title: string
  description: string
  state: AuthStepState
  badge: string
}

export interface AuthFlowViewModel {
  phase: AuthFlowPhase
  eyebrow: string
  title: string
  description: string
  weakHint?: string
  icon: string
  primaryLabel: string
  primaryAction: AuthPrimaryAction
  primaryURL?: string
  primaryDisabled: boolean
  showReset: boolean
  steps: AuthStepViewModel[]
}

export interface AuthFlowViewModelOptions {
  busyAction: string | null
  isCheckingAuthorizationReturn: boolean
  sessionId: string
  statusData: SessionStatusResponse | null
}

const CREATED_REUSE_HINT = '若已存在可复用的飞书应用配置，将自动跳过第一步配置。'
const RETURN_CHECK_COPY = '如果你刚刚已在浏览器完成授权，系统会在几秒内自动更新，无需再次点击。'
const RETURN_CHECK_STEP_COPY = '已提交浏览器授权，等待系统确认。'

export function buildAuthFlowViewModel(options: AuthFlowViewModelOptions): AuthFlowViewModel {
  const { busyAction, isCheckingAuthorizationReturn, sessionId, statusData } = options
  const status = statusData?.status || 'created'
  const verificationURL = statusData?.verification_url
  const reusedAppConfig = reusedAuthorizationConfig(statusData)
  const reachedStepTwo = reachedAuthorizationStepTwo(statusData)

  if (status === 'authenticated') {
    return {
      phase: 'authenticated',
      eyebrow: 'Ready to Start',
      title: '已就绪，可开始分析',
      description: '授权已完成，接下来将开始生成你的结构化 BSTI 画像。',
      icon: '●',
      primaryLabel: busyAction === 'analyze' ? '正在提交分析...' : '开启 AI 深度解析',
      primaryAction: 'analyze',
      primaryDisabled: busyAction === 'analyze',
      showReset: true,
      steps: buildSteps({
        firstStepState: 'done',
        firstStepReused: reusedAppConfig,
        secondStepState: 'done',
      }),
    }
  }

  if (status === 'config_pending') {
    return {
      phase: 'config_pending',
      eyebrow: '当前处于第 1 步',
      title: '第一步：完成飞书应用配置',
      description: '本地服务已经就绪，请先完成飞书应用配置。配置完成后会自动进入第二步授权。',
      icon: '◌',
      primaryLabel: busyAction === 'login' ? '正在准备第一步配置...' : '继续完成第一步配置',
      primaryAction: verificationURL ? 'open_url' : 'connect',
      primaryURL: verificationURL,
      primaryDisabled: busyAction === 'login',
      showReset: true,
      steps: buildSteps({
        firstStepState: 'active',
        secondStepState: 'pending',
      }),
    }
  }

  if (status === 'login_pending') {
    if (isCheckingAuthorizationReturn) {
      return {
        phase: 'checking_authorization_return',
        eyebrow: '正在回查浏览器授权结果',
        title: '正在检查飞书授权结果',
        description: RETURN_CHECK_COPY,
        icon: '◌',
        primaryLabel: '正在确认授权结果...',
        primaryAction: 'wait',
        primaryDisabled: true,
        showReset: true,
        steps: buildSteps({
          firstStepState: 'done',
          firstStepReused: reusedAppConfig,
          secondStepState: 'active',
          secondStepDescription: RETURN_CHECK_STEP_COPY,
          secondStepBadge: '确认中',
        }),
      }
    }

    return {
      phase: reusedAppConfig ? 'login_pending_reused' : 'login_pending',
      eyebrow: '当前处于第 2 步',
      title: verificationURL ? '第二步：完成飞书授权' : '第二步：等待授权完成',
      description: reusedAppConfig
        ? '已复用现有飞书应用配置，本次只需完成飞书授权。'
        : verificationURL
          ? '飞书应用配置已完成，当前只差浏览器中的授权确认。授权成功后系统会自动刷新当前状态。'
          : '飞书应用配置已完成，授权链接已创建，请在新窗口中完成飞书确认。',
      icon: '↗',
      primaryLabel: busyAction === 'login' ? '正在准备第二步授权...' : '去完成第二步授权',
      primaryAction: verificationURL ? 'open_url' : 'connect',
      primaryURL: verificationURL,
      primaryDisabled: busyAction === 'login',
      showReset: true,
      steps: buildSteps({
        firstStepState: 'done',
        firstStepReused: reusedAppConfig,
        secondStepState: 'active',
      }),
    }
  }

  if (status === 'failed') {
    const failedStepTwo = reachedStepTwo

    return {
      phase: failedStepTwo ? 'failed_step_2' : 'failed_step_1',
      eyebrow: failedStepTwo ? '第 2 步失败' : '第 1 步失败',
      title: failedStepTwo ? '第二步授权失败' : '连接失败',
      description: failedStepTwo
        ? '第一步配置已完成，但第二步飞书授权未成功。请重新完成授权。'
        : '无法完成飞书配置流程，请重试或新建会话。',
      icon: '×',
      primaryLabel:
        busyAction === 'login'
          ? '正在重试...'
          : failedStepTwo
            ? '重新完成第二步授权'
            : '重新尝试',
      primaryAction: verificationURL ? 'open_url' : 'connect',
      primaryURL: verificationURL,
      primaryDisabled: busyAction === 'login',
      showReset: true,
      steps: buildSteps({
        firstStepState: failedStepTwo ? 'done' : 'failed',
        firstStepReused: reusedAppConfig && failedStepTwo,
        secondStepState: failedStepTwo ? 'failed' : 'pending',
      }),
    }
  }

  return {
    phase: 'created',
    eyebrow: 'Privacy First',
    title: '连接飞书',
    description: sessionId
      ? `本地会话 ${sessionId.slice(0, 8)} 已创建。连接后即可拉取只读范围内的协作数据。`
      : '将基于本地飞书数据生成工作人格画像，所有信息只在当前设备中处理。',
    weakHint: CREATED_REUSE_HINT,
    icon: '◎',
    primaryLabel: busyAction === 'login' ? '正在准备授权...' : '连接我的飞书',
    primaryAction: 'connect',
    primaryDisabled: busyAction === 'login',
    showReset: false,
    steps: buildSteps({
      firstStepState: 'pending',
      secondStepState: 'pending',
    }),
  }
}

function buildSteps(options: {
  firstStepState: AuthStepState
  firstStepReused?: boolean
  secondStepBadge?: string
  secondStepDescription?: string
  secondStepState: AuthStepState
}): AuthStepViewModel[] {
  const {
    firstStepState,
    firstStepReused = false,
    secondStepBadge,
    secondStepDescription,
    secondStepState,
  } = options

  return [
    {
      number: 1,
      title: '飞书应用配置',
      description: firstStepReused
        ? '已复用现有飞书应用配置，本次无需重新完成这一步。'
        : '完成应用配置后，才能继续进行飞书授权。',
      state: firstStepState,
      badge: stepBadge(firstStepState, firstStepReused),
    },
    {
      number: 2,
      title: '飞书授权',
      description: secondStepDescription || '配置完成后，前往浏览器确认授权并返回当前页面。',
      state: secondStepState,
      badge: secondStepBadge || stepBadge(secondStepState),
    },
  ]
}

function reachedStepTwoEvent(stage: string) {
  return ['login_pending', 'authenticated', 'collecting', 'analyzing', 'done'].includes(stage)
}

function reachedAuthorizationStepTwo(statusData: SessionStatusResponse | null) {
  return (statusData?.events || []).some((event) => reachedStepTwoEvent(event.stage))
}

function reusedAuthorizationConfig(statusData: SessionStatusResponse | null) {
  const status = statusData?.status || 'created'
  if (!reachedStepTwoEvent(status)) {
    return false
  }
  return !(statusData?.events || []).some((event) => event.stage === 'config_pending')
}

function stepBadge(state: AuthStepState, reused = false) {
  switch (state) {
    case 'done':
      return reused ? '已复用' : '已完成'
    case 'active':
      return '进行中'
    case 'failed':
      return '失败'
    default:
      return '待开始'
  }
}
