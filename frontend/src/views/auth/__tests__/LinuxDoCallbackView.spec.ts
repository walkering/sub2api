import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  routeQuery: {} as Record<string, unknown>,
  replace: vi.fn(),
  setToken: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  completeLinuxDoOAuthRegistration: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: mocks.routeQuery }),
  useRouter: () => ({ replace: mocks.replace }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'auth.linuxdo.autoCheckinSuccess') {
          return `签到成功，赠送 ${params?.amount} 余额`
        }
        if (key === 'auth.loginSuccess') {
          return '登录成功'
        }
        return key
      },
    }),
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    setToken: mocks.setToken,
  }),
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

vi.mock('@/api/auth', () => ({
  completeLinuxDoOAuthRegistration: mocks.completeLinuxDoOAuthRegistration,
}))

import LinuxDoCallbackView from '../LinuxDoCallbackView.vue'

function resetLocation(hash = '') {
  window.history.replaceState({}, '', `/auth/oauth/linuxdo/callback${hash}`)
}

const globalMountOptions = {
  stubs: {
    transition: false,
    RouterLink: true,
    AuthLayout: {
      template: '<div><slot /></div>',
    },
    Icon: true,
  },
}

describe('LinuxDoCallbackView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    for (const key of Object.keys(mocks.routeQuery)) {
      delete mocks.routeQuery[key]
    }
    resetLocation()
  })

  it('fragment 返回自动签到奖励时显示签到成功提示', async () => {
    mocks.setToken.mockResolvedValue(undefined)
    resetLocation(
      '#access_token=access-token&auto_checkin_awarded=true&auto_checkin_bonus_amount=4&redirect=%2Fwallet'
    )

    mount(LinuxDoCallbackView, {
      global: globalMountOptions,
    })
    await flushPromises()

    expect(mocks.setToken).toHaveBeenCalledWith('access-token')
    expect(mocks.showSuccess).toHaveBeenCalledWith('签到成功，赠送 4 余额')
    expect(mocks.replace).toHaveBeenCalledWith('/wallet')
  })

  it('邀请码补全注册未触发签到奖励时显示普通登录成功提示', async () => {
    mocks.setToken.mockResolvedValue(undefined)
    mocks.completeLinuxDoOAuthRegistration.mockResolvedValue({
      access_token: 'completed-access-token',
      refresh_token: 'completed-refresh-token',
      expires_in: 3600,
      token_type: 'Bearer',
      auto_checkin_awarded: false,
      auto_checkin_bonus_amount: 0,
    })
    resetLocation('#error=invitation_required&pending_oauth_token=pending-token&redirect=%2Fdashboard')

    const wrapper = mount(LinuxDoCallbackView, {
      global: globalMountOptions,
    })
    await flushPromises()

    await wrapper.find('input').setValue('invite-123')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(mocks.completeLinuxDoOAuthRegistration).toHaveBeenCalledWith(
      'pending-token',
      'invite-123'
    )
    expect(mocks.setToken).toHaveBeenCalledWith('completed-access-token')
    expect(mocks.showSuccess).toHaveBeenCalledWith('登录成功')
    expect(mocks.replace).toHaveBeenCalledWith('/dashboard')
  })
})
