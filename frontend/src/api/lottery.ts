import { apiClient } from './client'

export interface LotteryThresholdRule {
  amount: number
  chances: number
}

export interface LotteryLoginGrantConfig {
  enabled: boolean
  daily_chances: number
  expiry_mode: 'end_of_day' | 'hours'
  expiry_hours: number
}

export interface LotteryThresholdGrantConfig {
  enabled: boolean
  thresholds: LotteryThresholdRule[]
  expiry_mode: 'end_of_day' | 'hours'
  expiry_hours: number
}

export interface LotteryConfig {
  enabled: boolean
  button_enabled: boolean
  timezone: string
  rule_text: string
  login_grant: LotteryLoginGrantConfig
  spend_grant: LotteryThresholdGrantConfig
  recharge_grant: LotteryThresholdGrantConfig
}

export interface LotteryPrize {
  id: number
  name: string
  amount: number
  probability: number
  daily_stock: number
  daily_used: number
  total_stock: number
  total_used: number
  enabled: boolean
  color: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface LotteryChanceSummary {
  remaining: number
  granted: number
  used: number
  expired: number
  by_source: Record<string, number>
}

export interface LotteryDrawRecord {
  id: number
  user_id: number
  user_email?: string
  prize_id: number
  prize_name: string
  amount: number
  balance_before: number
  balance_after: number
  source_type: string
  created_at: string
}

export interface LotteryStatus {
  config: LotteryConfig
  prizes: LotteryPrize[]
  chances: LotteryChanceSummary
  recent_draws: LotteryDrawRecord[]
  server_time: string
}

export interface LotteryDrawResult {
  prize: LotteryPrize
  record: LotteryDrawRecord
  remaining_chances: number
}

export interface PaginatedLotteryRecords {
  items: LotteryDrawRecord[]
  total: number
  page: number
  page_size: number
  pages: number
}

export const lotteryAPI = {
  async status(): Promise<LotteryStatus> {
    const { data } = await apiClient.get('/lottery/status')
    return data
  },

  async draw(): Promise<LotteryDrawResult> {
    const { data } = await apiClient.post('/lottery/draw')
    return data
  },

  async history(params: { page?: number; page_size?: number } = {}): Promise<PaginatedLotteryRecords> {
    const { data } = await apiClient.get('/lottery/history', { params })
    return data
  }
}

export default lotteryAPI
