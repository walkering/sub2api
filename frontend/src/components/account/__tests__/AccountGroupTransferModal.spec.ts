import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AccountGroupTransferModal from '../AccountGroupTransferModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      transferAccountsByGroup: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.count ? `${key}:${params.count}` : key
    })
  }
})

function mountModal() {
  return mount(AccountGroupTransferModal, {
    props: {
      show: true,
      groups: [
        { id: 10, name: 'Source' },
        { id: 20, name: 'Target' }
      ]
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: {
          name: 'SelectStub',
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: `
            <select
              v-bind="$attrs"
              :value="modelValue"
              @change="$emit('update:modelValue', $event.target.value)"
            >
              <option value="">empty</option>
              <option v-for="option in options" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          `
        }
      }
    }
  })
}

describe('AccountGroupTransferModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.accounts.transferAccountsByGroup).mockReset()
    vi.mocked(adminAPI.accounts.transferAccountsByGroup).mockResolvedValue({
      source_group_id: 10,
      target_group_id: 20,
      requested_count: 2,
      moved_count: 2,
      account_ids: [1, 2]
    } as any)
  })

  it('未选择源分组和目标分组时应阻止提交', async () => {
    const wrapper = mountModal()

    await wrapper.get('#account-group-transfer-form').trigger('submit.prevent')

    expect(showError).toHaveBeenCalledWith('admin.accounts.groupTransfer.selectGroups')
    expect(adminAPI.accounts.transferAccountsByGroup).not.toHaveBeenCalled()
  })

  it('源分组和目标分组相同时应阻止提交', async () => {
    const wrapper = mountModal()
    const selects = wrapper.findAllComponents({ name: 'SelectStub' })

    selects[0].vm.$emit('update:modelValue', '10')
    selects[1].vm.$emit('update:modelValue', '10')
    await wrapper.get('#account-group-transfer-form').trigger('submit.prevent')

    expect(showError).toHaveBeenCalledWith('admin.accounts.groupTransfer.groupsMustDiffer')
    expect(adminAPI.accounts.transferAccountsByGroup).not.toHaveBeenCalled()
  })

  it('迁移数量不合法时应阻止提交', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="group-transfer-source"]').setValue('10')
    await wrapper.get('[data-testid="group-transfer-target"]').setValue('20')
    await wrapper.get('#group-transfer-count').setValue('0')
    await wrapper.get('#account-group-transfer-form').trigger('submit.prevent')

    expect(showError).toHaveBeenCalledWith('admin.accounts.groupTransfer.invalidCount')
    expect(adminAPI.accounts.transferAccountsByGroup).not.toHaveBeenCalled()
  })

  it('校验通过时应调用分组迁移接口并抛出成功事件', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="group-transfer-source"]').setValue('10')
    await wrapper.get('[data-testid="group-transfer-target"]').setValue('20')
    await wrapper.get('#group-transfer-count').setValue('2')
    await wrapper.get('#account-group-transfer-form').trigger('submit.prevent')
    await flushPromises()

    expect(adminAPI.accounts.transferAccountsByGroup).toHaveBeenCalledWith({
      source_group_id: 10,
      target_group_id: 20,
      count: 2
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.groupTransfer.success:2')
    expect(wrapper.emitted('transferred')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
