import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BatchAccountTestModal from '../BatchAccountTestModal.vue'

const { streamAccountTest } = vi.hoisted(() => ({
  streamAccountTest: vi.fn()
}))

vi.mock('@/utils/accountTestStream', () => ({
  streamAccountTest
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.bulkTest.testingWithModel') {
          return `testing-${params?.model}`
        }
        if (key === 'admin.accounts.bulkTest.successWithModel') {
          return `success-${params?.model}`
        }
        if (key === 'admin.accounts.bulkTest.imageReceived') {
          return `image-${params?.count}`
        }
        return key
      }
    })
  }
})

describe('BatchAccountTestModal', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null))
      },
      configurable: true
    })
    vi.mocked(streamAccountTest).mockReset()
  })

  it('runs selected accounts sequentially and emits completed', async () => {
    vi.mocked(streamAccountTest)
      .mockImplementationOnce(async ({ onEvent }) => {
        onEvent?.({ type: 'test_start', model: 'gpt-5' })
        onEvent?.({ type: 'content', text: 'first account ok' })
        return { success: true }
      })
      .mockImplementationOnce(async ({ onEvent }) => {
        onEvent?.({ type: 'test_start', model: 'claude-sonnet' })
        return { success: false, error: 'second account failed' }
      })

    const wrapper = mount(BatchAccountTestModal, {
      props: {
        show: true,
        accounts: [
          { id: 1, name: 'Account A' },
          { id: 2, name: 'Account B' }
        ]
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: true
        }
      }
    })

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.bulkActions.test'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(streamAccountTest).toHaveBeenCalledTimes(2)
    expect(vi.mocked(streamAccountTest).mock.calls[0]?.[0]).toMatchObject({
      accountId: 1,
      authToken: 'test-token'
    })
    expect(vi.mocked(streamAccountTest).mock.calls[1]?.[0]).toMatchObject({
      accountId: 2,
      authToken: 'test-token'
    })
    expect(wrapper.text()).toContain('first account ok')
    expect(wrapper.text()).toContain('second account failed')
    expect(wrapper.emitted('completed')).toHaveLength(1)
  })
})
