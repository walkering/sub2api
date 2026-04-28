import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BatchAccountTestModal from '../BatchAccountTestModal.vue'

const { streamAccountTest, getAvailableModels } = vi.hoisted(() => ({
  streamAccountTest: vi.fn(),
  getAvailableModels: vi.fn()
}))

vi.mock('@/utils/accountTestStream', () => ({
  streamAccountTest
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels
    }
  }
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
    vi.mocked(getAvailableModels).mockReset()
  })

  it('runs selected accounts sequentially with the selected model and emits completed', async () => {
    vi.mocked(getAvailableModels)
      .mockResolvedValueOnce([
        { id: 'gpt-4.1', display_name: 'GPT-4.1' },
        { id: 'gpt-5', display_name: 'GPT-5' }
      ] as any)
      .mockResolvedValueOnce([
        { id: 'claude-haiku-4-5', display_name: 'Claude Haiku 4.5' },
        { id: 'claude-sonnet-4', display_name: 'Claude Sonnet 4' }
      ] as any)

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
          { id: 1, name: 'Account A', platform: 'openai' },
          { id: 2, name: 'Account B', platform: 'anthropic' }
        ]
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: true,
          Select: {
            props: ['modelValue', 'options', 'valueKey', 'labelKey', 'disabled', 'placeholder'],
            emits: ['update:modelValue'],
            template: `
              <select
                data-test="model-select"
                :value="modelValue"
                :disabled="disabled"
                @change="$emit('update:modelValue', $event.target.value)"
              >
                <option value="">{{ placeholder }}</option>
                <option
                  v-for="option in options"
                  :key="option[valueKey || 'value']"
                  :value="option[valueKey || 'value']"
                >
                  {{ option[labelKey || 'label'] }}
                </option>
              </select>
            `
          }
        }
      }
    })

    await flushPromises()
    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.bulkActions.test'))
    expect(startButton).toBeTruthy()

    const selects = wrapper.findAll('[data-test="model-select"]')
    expect(selects).toHaveLength(2)
    await selects[0]!.setValue('gpt-5')

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(getAvailableModels).toHaveBeenCalledTimes(2)
    expect(streamAccountTest).toHaveBeenCalledTimes(2)
    expect(vi.mocked(streamAccountTest).mock.calls[0]?.[0]).toMatchObject({
      accountId: 1,
      authToken: 'test-token',
      modelId: 'gpt-5'
    })
    expect(vi.mocked(streamAccountTest).mock.calls[1]?.[0]).toMatchObject({
      accountId: 2,
      authToken: 'test-token',
      modelId: 'claude-sonnet-4'
    })
    expect(wrapper.text()).toContain('first account ok')
    expect(wrapper.text()).toContain('second account failed')
    expect(wrapper.emitted('completed')).toHaveLength(1)
  })
})
