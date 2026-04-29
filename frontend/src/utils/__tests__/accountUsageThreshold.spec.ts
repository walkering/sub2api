import { describe, expect, it } from 'vitest'
import type { Account } from '@/types'
import {
  getAccountMaxQuotaUsageRatio,
  getAccountUsageThresholdStatus,
} from '../accountUsageThreshold'

type RatioAccount = Parameters<typeof getAccountMaxQuotaUsageRatio>[0]
type UsageAccount = Parameters<typeof getAccountUsageThresholdStatus>[0]

const now = new Date('2026-04-10T00:00:00Z')

const makeGroup = (
  id: number,
  thresholdPercent: number | null,
  name = `group-${id}`,
): Account['groups'][number] =>
  ({
    id,
    name,
    account_usage_threshold_percent: thresholdPercent,
  }) as Account['groups'][number]

const makeRatioAccount = (overrides: Partial<RatioAccount> = {}): RatioAccount => ({
  quota_used: null,
  quota_limit: null,
  quota_daily_used: null,
  quota_daily_limit: null,
  quota_daily_reset_at: null,
  quota_weekly_used: null,
  quota_weekly_limit: null,
  quota_weekly_reset_at: null,
  ...overrides,
})

const makeUsageAccount = (overrides: Partial<UsageAccount> = {}): UsageAccount =>
  ({
    ...makeRatioAccount(),
    groups: [],
    active_sessions: 0,
    ...overrides,
  }) as UsageAccount

describe('accountUsageThreshold', () => {
  it('日/周配额已到重置时间时按 0 参与最大比例计算', () => {
    const ratio = getAccountMaxQuotaUsageRatio(
      makeRatioAccount({
        quota_used: 40,
        quota_limit: 100,
        quota_daily_used: 10,
        quota_daily_limit: 10,
        quota_daily_reset_at: '2026-04-09T23:59:59Z',
        quota_weekly_used: 20,
        quota_weekly_limit: 20,
        quota_weekly_reset_at: '2026-04-09T12:00:00Z',
      }),
      now,
    )

    expect(ratio).toEqual({
      ratio: 0.4,
      dimension: 'total',
    })
  })

  it('没有任何可计算配额上限时返回 null', () => {
    expect(getAccountMaxQuotaUsageRatio(makeRatioAccount(), now)).toBeNull()
    expect(getAccountUsageThresholdStatus(makeUsageAccount(), now)).toBeNull()
  })

  it('达到阈值且仍有活跃会话时返回 sticky_only', () => {
    const status = getAccountUsageThresholdStatus(
      makeUsageAccount({
        quota_used: 95,
        quota_limit: 100,
        active_sessions: 1,
        groups: [makeGroup(1, 95, 'alpha')],
      }),
      now,
    )

    expect(status).toEqual({
      state: 'sticky_only',
      groupId: 1,
      groupName: 'alpha',
      thresholdPercent: 95,
      usageRatio: 0.95,
      dimension: 'total',
    })
  })

  it('达到阈值且无活跃会话时返回 not_schedulable', () => {
    const status = getAccountUsageThresholdStatus(
      makeUsageAccount({
        quota_daily_used: 19,
        quota_daily_limit: 20,
        active_sessions: 0,
        groups: [makeGroup(2, 95, 'beta')],
      }),
      now,
    )

    expect(status).toEqual({
      state: 'not_schedulable',
      groupId: 2,
      groupName: 'beta',
      thresholdPercent: 95,
      usageRatio: 0.95,
      dimension: 'daily',
    })
  })

  it('多分组命中时选择最严格阈值', () => {
    const status = getAccountUsageThresholdStatus(
      makeUsageAccount({
        quota_used: 92,
        quota_limit: 100,
        groups: [makeGroup(10, 95, 'loose'), makeGroup(3, 90, 'strict')],
      }),
      now,
    )

    expect(status?.groupId).toBe(3)
    expect(status?.groupName).toBe('strict')
    expect(status?.thresholdPercent).toBe(90)
  })

  it('阈值相同且都命中时选择较小的分组 ID', () => {
    const status = getAccountUsageThresholdStatus(
      makeUsageAccount({
        quota_weekly_used: 48,
        quota_weekly_limit: 50,
        groups: [makeGroup(8, 95, 'later'), makeGroup(4, 95, 'earlier')],
      }),
      now,
    )

    expect(status?.groupId).toBe(4)
    expect(status?.groupName).toBe('earlier')
  })
})
