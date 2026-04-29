import type { Account } from '@/types'

export type AccountUsageThresholdState = 'sticky_only' | 'not_schedulable'
export type AccountQuotaUsageDimension = 'total' | 'daily' | 'weekly'

export interface AccountMaxQuotaUsageRatio {
  ratio: number
  dimension: AccountQuotaUsageDimension
}

export interface AccountUsageThresholdStatus {
  state: AccountUsageThresholdState
  groupId: number
  groupName: string
  thresholdPercent: number
  usageRatio: number
  dimension: AccountQuotaUsageDimension
}

const getResetAwareUsed = (used: number | null | undefined, resetAt: string | null | undefined, nowMs: number): number => {
  if (resetAt) {
    const resetMs = Date.parse(resetAt)
    if (!Number.isNaN(resetMs) && resetMs <= nowMs) {
      return 0
    }
  }

  return used ?? 0
}

const getQuotaRatio = (used: number | null | undefined, limit: number | null | undefined): number | null => {
  if (limit === null || limit === undefined || limit <= 0) {
    return null
  }

  return (used ?? 0) / limit
}

export const getAccountMaxQuotaUsageRatio = (
  account: Pick<
    Account,
    | 'quota_used'
    | 'quota_limit'
    | 'quota_daily_used'
    | 'quota_daily_limit'
    | 'quota_daily_reset_at'
    | 'quota_weekly_used'
    | 'quota_weekly_limit'
    | 'quota_weekly_reset_at'
  >,
  now: Date = new Date(),
): AccountMaxQuotaUsageRatio | null => {
  const nowMs = now.getTime()
  const candidates: AccountMaxQuotaUsageRatio[] = []

  const totalRatio = getQuotaRatio(account.quota_used, account.quota_limit)
  if (totalRatio !== null) {
    candidates.push({ ratio: totalRatio, dimension: 'total' })
  }

  const dailyRatio = getQuotaRatio(
    getResetAwareUsed(account.quota_daily_used, account.quota_daily_reset_at, nowMs),
    account.quota_daily_limit,
  )
  if (dailyRatio !== null) {
    candidates.push({ ratio: dailyRatio, dimension: 'daily' })
  }

  const weeklyRatio = getQuotaRatio(
    getResetAwareUsed(account.quota_weekly_used, account.quota_weekly_reset_at, nowMs),
    account.quota_weekly_limit,
  )
  if (weeklyRatio !== null) {
    candidates.push({ ratio: weeklyRatio, dimension: 'weekly' })
  }

  if (candidates.length === 0) {
    return null
  }

  return candidates.reduce((max, current) => (current.ratio > max.ratio ? current : max))
}

export const getAccountUsageThresholdStatus = (
  account: Pick<
    Account,
    | 'groups'
    | 'active_sessions'
    | 'quota_used'
    | 'quota_limit'
    | 'quota_daily_used'
    | 'quota_daily_limit'
    | 'quota_daily_reset_at'
    | 'quota_weekly_used'
    | 'quota_weekly_limit'
    | 'quota_weekly_reset_at'
  >,
  now: Date = new Date(),
): AccountUsageThresholdStatus | null => {
  const usage = getAccountMaxQuotaUsageRatio(account, now)
  if (!usage) {
    return null
  }

  const activeSessions = account.active_sessions ?? 0
  const state: AccountUsageThresholdState = activeSessions > 0 ? 'sticky_only' : 'not_schedulable'

  let matched: AccountUsageThresholdStatus | null = null
  for (const group of account.groups ?? []) {
    const thresholdPercent = group.account_usage_threshold_percent ?? null
    if (thresholdPercent === null || thresholdPercent <= 0) {
      continue
    }
    if (usage.ratio < thresholdPercent / 100) {
      continue
    }

    const candidate: AccountUsageThresholdStatus = {
      state,
      groupId: group.id,
      groupName: group.name,
      thresholdPercent,
      usageRatio: usage.ratio,
      dimension: usage.dimension,
    }

    if (!matched || thresholdPercent < matched.thresholdPercent || (thresholdPercent === matched.thresholdPercent && group.id < matched.groupId)) {
      matched = candidate
    }
  }

  return matched
}
