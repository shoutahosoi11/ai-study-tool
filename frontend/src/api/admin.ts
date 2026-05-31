import { apiClient } from './client'

export type AdminLLMBudget = {
  budget_date: string
  max_requests: number
  used_requests: number
  max_estimated_cost_yen: number
  used_estimated_cost_yen: number
  updated_at?: string
}

export type AdminLLMUsageTotals = {
  request_count: number
  input_tokens: number
  output_tokens: number
  estimated_cost_yen: number
}

export type AdminJobStatusCounts = {
  queued: number
  processing: number
  failed: number
  completed: number
  enqueue_failed: number
}

export type AdminAuditLog = {
  id: string
  admin_user_id: string
  action: string
  target_type: string
  target_id?: string
  metadata: Record<string, unknown>
  created_at: string
}

export type AdminOverview = {
  budget: AdminLLMBudget
  llm_usage_today: AdminLLMUsageTotals
  generation_jobs: AdminJobStatusCounts
  cloud_tasks_queue_estimate: number
  stripe_webhook_error_count: number
  admob_ssv_error_count: number
  extension_import_count: number
  rate_limit_429_count: number
  recent_audit_logs: AdminAuditLog[]
}

export type AdminQuestionBudget = {
  free_used_today: number
  token_used_today: number
  ad_views_today: number
  available_tokens: number
}

export type AdminUser = {
  id: string
  firebase_uid: string
  email?: string
  username: string
  plan: string
  subscription_status?: string
  created_at: string
  last_active_at?: string
  question_budget: AdminQuestionBudget
  extension_token_count: number
  recent_jobs_count: number
}

export type AdminExtensionToken = {
  id: string
  name?: string
  scopes: string[]
  created_at: string
  last_used_at?: string
  expires_at?: string
  revoked_at?: string
}

export type AdminLLMProviderUsage = {
  provider: string
  model: string
  request_count: number
  input_tokens: number
  output_tokens: number
  estimated_cost_yen: number
}

export type AdminFailedJobReason = {
  reason: string
  count: number
}

export type AdminLLMOverview = {
  budget: AdminLLMBudget
  usage_today: AdminLLMUsageTotals
  provider_models: AdminLLMProviderUsage[]
  failed_job_reasons: AdminFailedJobReason[]
}

export type AdminGenerationJob = {
  id: string
  user_id: string
  book_id: string
  status: string
  reason: string
  retry_count: number
  created_at: string
  updated_at: string
  failed_at?: string
  completed_at?: string
}

export type AdminBilling = {
  events: Array<{
    event_id: string
    event_type: string
    processed_at: string
  }>
  failure_count: number
}

export type AdminAdMob = {
  events: Array<{
    transaction_id: string
    user_id: string
    reward_amount: number
    verified_at: string
  }>
  duplicate_count: number
  stale_fallback_count: number
}

export async function fetchAdminOverview() {
  const response = await apiClient.get<AdminOverview>('/admin/overview')
  return response.data
}

export async function searchAdminUsers(query: string) {
  const response = await apiClient.get<{ users: AdminUser[] }>('/admin/users', {
    params: { q: query.trim() },
  })
  return response.data.users
}

export async function fetchAdminUser(userId: string) {
  const response = await apiClient.get<AdminUser>(`/admin/users/${userId}`)
  return response.data
}

export async function fetchAdminExtensionTokens(userId: string) {
  const response = await apiClient.get<{ tokens: AdminExtensionToken[] }>(`/admin/users/${userId}/extension-tokens`)
  return response.data.tokens
}

export async function revokeAdminExtensionToken(userId: string, tokenId: string) {
  await apiClient.post(`/admin/users/${userId}/extension-tokens/${tokenId}/revoke`)
}

export async function revokeAllAdminExtensionTokens(userId: string) {
  const response = await apiClient.post<{ revoked_count: number }>(`/admin/users/${userId}/extension-tokens/revoke-all`)
  return response.data.revoked_count
}

export async function adminLogoutAll(userId: string) {
  await apiClient.post(`/admin/users/${userId}/logout-all`)
}

export async function fetchAdminLLM() {
  const response = await apiClient.get<AdminLLMOverview>('/admin/llm')
  return response.data
}

export async function updateAdminLLMBudget(maxRequests: number, maxEstimatedCostYen: number) {
  const response = await apiClient.put<AdminLLMBudget>('/admin/llm/budget', {
    max_requests: maxRequests,
    max_estimated_cost_yen: maxEstimatedCostYen,
  })
  return response.data
}

export async function fetchAdminJobs(status: string) {
  const response = await apiClient.get<{ jobs: AdminGenerationJob[] }>('/admin/jobs', {
    params: { status },
  })
  return response.data.jobs
}

export async function retryAdminJob(jobId: string) {
  await apiClient.post(`/admin/jobs/${jobId}/retry`)
}

export async function cancelAdminJob(jobId: string) {
  await apiClient.post(`/admin/jobs/${jobId}/cancel`)
}

export async function fetchAdminBilling() {
  const response = await apiClient.get<AdminBilling>('/admin/billing')
  return response.data
}

export async function fetchAdminAdMob() {
  const response = await apiClient.get<AdminAdMob>('/admin/admob')
  return response.data
}
