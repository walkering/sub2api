/**
 * Admin Scheduled Tests API endpoints
 * Handles scheduled test plan management for account connectivity monitoring
 */

import { apiClient } from '../client'
import type {
  ScheduledTestPlan,
  ScheduledTestResult,
  AccountTestJob,
  AccountTestJobSnapshot,
  CreateScheduledTestPlanRequest,
  UpdateScheduledTestPlanRequest,
  CreateAccountTestJobRequest
} from '@/types'

/**
 * List all scheduled test plans for an account
 * @param accountId - Account ID
 * @returns List of scheduled test plans
 */
export async function listByAccount(accountId: number): Promise<ScheduledTestPlan[]> {
  const { data } = await apiClient.get<ScheduledTestPlan[]>(
    `/admin/accounts/${accountId}/scheduled-test-plans`
  )
  return data ?? []
}

export async function listByGroup(groupId: number): Promise<ScheduledTestPlan[]> {
  const { data } = await apiClient.get<ScheduledTestPlan[]>(
    `/admin/groups/${groupId}/scheduled-test-plans`
  )
  return data ?? []
}

/**
 * Create a new scheduled test plan
 * @param req - Plan creation request
 * @returns Created plan
 */
export async function create(req: CreateScheduledTestPlanRequest): Promise<ScheduledTestPlan> {
  const { data } = await apiClient.post<ScheduledTestPlan>(
    '/admin/scheduled-test-plans',
    req
  )
  return data
}

/**
 * Update an existing scheduled test plan
 * @param id - Plan ID
 * @param req - Fields to update
 * @returns Updated plan
 */
export async function update(id: number, req: UpdateScheduledTestPlanRequest): Promise<ScheduledTestPlan> {
  const { data } = await apiClient.put<ScheduledTestPlan>(
    `/admin/scheduled-test-plans/${id}`,
    req
  )
  return data
}

/**
 * Delete a scheduled test plan
 * @param id - Plan ID
 */
export async function deletePlan(id: number): Promise<void> {
  await apiClient.delete(`/admin/scheduled-test-plans/${id}`)
}

/**
 * List test results for a plan
 * @param planId - Plan ID
 * @param limit - Optional max number of results to return
 * @returns List of test results
 */
export async function listResults(planId: number, limit?: number): Promise<ScheduledTestResult[]> {
  const { data } = await apiClient.get<ScheduledTestResult[]>(
    `/admin/scheduled-test-plans/${planId}/results`,
    {
      params: limit ? { limit } : undefined
    }
  )
  return data ?? []
}

export async function createGroupJob(
  groupId: number,
  req: CreateAccountTestJobRequest = {}
): Promise<AccountTestJob> {
  const { data } = await apiClient.post<AccountTestJob>(`/admin/groups/${groupId}/test-jobs`, req)
  return data
}

export async function listGroupJobs(groupId: number, limit: number = 20): Promise<AccountTestJob[]> {
  const { data } = await apiClient.get<AccountTestJob[]>(`/admin/groups/${groupId}/test-jobs`, {
    params: { limit }
  })
  return data ?? []
}

export async function getJob(jobId: number): Promise<AccountTestJob> {
  const { data } = await apiClient.get<AccountTestJob>(`/admin/test-jobs/${jobId}`)
  return data
}

export async function getJobSnapshot(jobId: number, logLimit: number = 200): Promise<AccountTestJobSnapshot> {
  const { data } = await apiClient.get<AccountTestJobSnapshot>(`/admin/test-jobs/${jobId}/snapshot`, {
    params: { log_limit: logLimit }
  })
  return data
}

export const scheduledTestsAPI = {
  listByAccount,
  listByGroup,
  create,
  update,
  delete: deletePlan,
  listResults,
  createGroupJob,
  listGroupJobs,
  getJob,
  getJobSnapshot
}

export default scheduledTestsAPI
