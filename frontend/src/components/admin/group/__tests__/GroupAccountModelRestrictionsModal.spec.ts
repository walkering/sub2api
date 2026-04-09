import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import GroupAccountModelRestrictionsModal from '../GroupAccountModelRestrictionsModal.vue'
import { adminAPI } from '@/api/admin'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      updateAccountModelRestrictions: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (params?.count !== undefined) {
          return `${key}:${params.count}`
        }
        return key
      }
    })
  }
})

function mountModal(extraProps: Record<string, unknown> = {}) {
  return mount(GroupAccountModelRestrictionsModal, {
    props: {
      show: true,
      group: {
        id: 7,
        name: 'OpenAI Group',
        platform: 'openai',
        account_count: 3
      },
      ...extraProps
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /></div>', props: ['show', 'title', 'width'] },
        Icon: true,
        ModelWhitelistSelector: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: `
            <div>
              <button
                type="button"
                data-testid="set-whitelist-models"
                @click="$emit('update:modelValue', ['gpt-5.4'])"
              >
                set-models
              </button>
            </div>
          `
        }
      }
    }
  })
}

describe('GroupAccountModelRestrictionsModal', () => {
  beforeEach(() => {
    vi.mocked(adminAPI.groups.updateAccountModelRestrictions).mockReset()
    vi.mocked(adminAPI.groups.updateAccountModelRestrictions).mockResolvedValue({
      group_id: 7,
      target_count: 3,
      success: 3,
      failed: 0,
      success_ids: [1, 2, 3],
      failed_ids: [],
      results: []
    } as any)
  })

  it('白名单留空时应提交空 model_mapping', async () => {
    const wrapper = mountModal({
      group: {
        id: 8,
        name: 'Anthropic Group',
        platform: 'anthropic',
        account_count: 2
      }
    })

    await wrapper.get('#group-account-model-restrictions-form').trigger('submit.prevent')
    await flushPromises()

    expect(adminAPI.groups.updateAccountModelRestrictions).toHaveBeenCalledTimes(1)
    expect(adminAPI.groups.updateAccountModelRestrictions).toHaveBeenCalledWith(8, {
      credentials: {
        model_mapping: {}
      }
    })
  })

  it('映射模式添加预设后应提交对应 model_mapping', async () => {
    const wrapper = mountModal()

    await wrapper.get('#group-account-model-restriction-mode-mapping').trigger('click')
    const presetButton = wrapper.findAll('button').find((button) => button.text().includes('GPT-5.4'))
    expect(presetButton).toBeTruthy()
    await presetButton!.trigger('click')
    await wrapper.get('#group-account-model-restrictions-form').trigger('submit.prevent')
    await flushPromises()

    expect(adminAPI.groups.updateAccountModelRestrictions).toHaveBeenCalledWith(7, {
      credentials: {
        model_mapping: {
          'gpt-5.4': 'gpt-5.4'
        }
      }
    })
  })

  it('显示账号本体配置影响其他分组的提示文案', () => {
    const wrapper = mountModal()

    expect(wrapper.text()).toContain('admin.groups.accountModelRestrictions.impactScope')
    expect(wrapper.text()).toContain('admin.groups.accountModelRestrictions.openaiPassthroughHint')
  })
})
