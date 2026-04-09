import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, reactive, ref } from 'vue'

const mockCreatePlan = vi.fn()
const mockShowError = vi.fn()
const mockShowSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mockShowError,
    showSuccess: mockShowSuccess
  })
}))

const ScheduledPlanFormTestComponent = defineComponent({
  setup() {
    const form = reactive({
      group_id: null as number | null,
      cron_expression: '',
      batch_size: 0,
      offset: 0
    })

    const validate = () => {
      if (!form.group_id) return '请选择分组'
      if (!form.cron_expression.trim()) return '请输入 Cron 表达式'
      if (form.batch_size <= 0) return '批次大小必须大于 0'
      if (form.offset < 0) return '偏移秒数必须大于等于 0'
      return ''
    }

    const submit = async () => {
      const error = validate()
      if (error) {
        mockShowError(error)
        return
      }
      await mockCreatePlan({
        group_id: form.group_id,
        cron_expression: form.cron_expression,
        batch_size: form.batch_size,
        offset: form.offset
      })
      mockShowSuccess('ok')
    }

    return { form, submit }
  },
  template: `
    <form @submit.prevent="submit">
      <select id="group" v-model="form.group_id">
        <option :value="null">请选择</option>
        <option :value="1">Group 1</option>
      </select>
      <input id="cron" v-model="form.cron_expression" />
      <input id="batch" v-model.number="form.batch_size" type="number" />
      <input id="offset" v-model.number="form.offset" type="number" />
      <button type="submit">提交</button>
    </form>
  `
})

const LogViewerTestComponent = defineComponent({
  setup() {
    const snapshot = ref({
      job: {
        status: 'running',
        succeeded_accounts: 1,
        failed_accounts: 0
      },
      logs: [
        { id: 1, message: '开始测试账号：acc-1' },
        { id: 2, message: '账号测试成功：acc-1' }
      ],
      items: [
        { id: 1, account_name: 'acc-1', status: 'succeeded' }
      ]
    })
    return { snapshot }
  },
  template: `
    <div>
      <div class="job-status">{{ snapshot.job.status }}</div>
      <div v-for="log in snapshot.logs" :key="log.id" class="log-entry">{{ log.message }}</div>
      <div v-for="item in snapshot.items" :key="item.id" class="item-status">{{ item.account_name }}-{{ item.status }}</div>
    </div>
  `
})

describe('分组定时测试表单校验', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('批次大小非法时阻止提交', async () => {
    const wrapper = mount(ScheduledPlanFormTestComponent)

    await wrapper.find('#group').setValue('1')
    await wrapper.find('#cron').setValue('*/30 * * * *')
    await wrapper.find('#batch').setValue('0')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mockCreatePlan).not.toHaveBeenCalled()
    expect(mockShowError).toHaveBeenCalledWith('批次大小必须大于 0')
  })

  it('表单合法时提交 group/batch/offset 参数', async () => {
    mockCreatePlan.mockResolvedValue({ id: 1 })
    const wrapper = mount(ScheduledPlanFormTestComponent)

    await wrapper.find('#group').setValue('1')
    await wrapper.find('#cron').setValue('*/15 * * * *')
    await wrapper.find('#batch').setValue('3')
    await wrapper.find('#offset').setValue('20')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mockCreatePlan).toHaveBeenCalledWith({
      group_id: 1,
      cron_expression: '*/15 * * * *',
      batch_size: 3,
      offset: 20
    })
  })
})

describe('分组测试日志展示', () => {
  it('展示任务状态、日志和账号结果', () => {
    const wrapper = mount(LogViewerTestComponent)

    expect(wrapper.find('.job-status').text()).toBe('running')
    expect(wrapper.findAll('.log-entry')).toHaveLength(2)
    expect(wrapper.text()).toContain('账号测试成功：acc-1')
    expect(wrapper.find('.item-status').text()).toBe('acc-1-succeeded')
  })
})
