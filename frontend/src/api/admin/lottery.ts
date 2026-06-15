import { apiClient } from '../client'
import type {
  LotteryConfig,
  LotteryDrawRecord,
  LotteryPrize,
  PaginatedLotteryRecords
} from '../lottery'

export interface LotteryPrizeInput {
  name: string
  amount: number
  probability: number
  daily_stock: number
  total_stock: number
  enabled: boolean
  color: string
  sort_order: number
}

export const adminLotteryAPI = {
  async getConfig(): Promise<LotteryConfig> {
    const { data } = await apiClient.get('/admin/lottery/config')
    return data
  },

  async updateConfig(config: LotteryConfig): Promise<LotteryConfig> {
    const { data } = await apiClient.put('/admin/lottery/config', config)
    return data
  },

  async listPrizes(): Promise<LotteryPrize[]> {
    const { data } = await apiClient.get('/admin/lottery/prizes')
    return data
  },

  async createPrize(input: LotteryPrizeInput): Promise<LotteryPrize> {
    const { data } = await apiClient.post('/admin/lottery/prizes', input)
    return data
  },

  async updatePrize(id: number, input: LotteryPrizeInput): Promise<LotteryPrize> {
    const { data } = await apiClient.put(`/admin/lottery/prizes/${id}`, input)
    return data
  },

  async deletePrize(id: number): Promise<void> {
    await apiClient.delete(`/admin/lottery/prizes/${id}`)
  },

  async listRecords(params: { page?: number; page_size?: number; user_id?: number } = {}): Promise<PaginatedLotteryRecords & { items: LotteryDrawRecord[] }> {
    const { data } = await apiClient.get('/admin/lottery/records', { params })
    return data
  }
}

export default adminLotteryAPI
