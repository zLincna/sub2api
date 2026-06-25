import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'
import type {
  CarpoolAdminManagementResponse,
  CarpoolNoticeInput,
  CarpoolNoticeVersion,
  CarpoolSession,
  CarpoolVoucher,
  CarpoolVoucherInput,
  CarpoolVehicleType,
  CarpoolVehicleTypeInput,
} from '@/types/carpool'

export const adminCarpoolAPI = {
  async overview(): Promise<Record<string, unknown>> {
    const { data } = await apiClient.get('/admin/carpool/overview')
    return data
  },

  async listTypes(): Promise<CarpoolVehicleType[]> {
    const { data } = await apiClient.get('/admin/carpool/types')
    return data
  },

  async createType(input: CarpoolVehicleTypeInput): Promise<CarpoolVehicleType> {
    const { data } = await apiClient.post('/admin/carpool/types', input)
    return data
  },

  async updateType(id: number, input: CarpoolVehicleTypeInput): Promise<CarpoolVehicleType> {
    const { data } = await apiClient.put(`/admin/carpool/types/${id}`, input)
    return data
  },

  async deleteType(id: number): Promise<void> {
    await apiClient.delete(`/admin/carpool/types/${id}`)
  },

  async listSessions(params: { page?: number; page_size?: number; status?: string } = {}): Promise<BasePaginationResponse<CarpoolSession>> {
    const { data } = await apiClient.get('/admin/carpool/sessions', { params })
    return data
  },

  async management(params: { page?: number; page_size?: number; status?: string } = {}): Promise<CarpoolAdminManagementResponse> {
    const { data } = await apiClient.get('/admin/carpool/management', { params })
    return data
  },

  async provisionSession(id: number, input: { status: string; account_info: Record<string, unknown>; proxy_info: Record<string, unknown>; communication: Record<string, unknown>; admin_notes: string }): Promise<CarpoolSession> {
    const { data } = await apiClient.put(`/admin/carpool/sessions/${id}/provision`, input)
    return data
  },

  async listVouchers(sessionId: number): Promise<CarpoolVoucher[]> {
    const { data } = await apiClient.get(`/admin/carpool/sessions/${sessionId}/vouchers`)
    return data
  },

  async createVoucher(sessionId: number, input: CarpoolVoucherInput): Promise<CarpoolVoucher> {
    const { data } = await apiClient.post(`/admin/carpool/sessions/${sessionId}/vouchers`, input)
    return data
  },

  async deleteVoucher(id: number): Promise<void> {
    await apiClient.delete(`/admin/carpool/vouchers/${id}`)
  },

  async listNotices(): Promise<CarpoolNoticeVersion[]> {
    const { data } = await apiClient.get('/admin/carpool/notices')
    return data
  },

  async createNotice(input: CarpoolNoticeInput): Promise<CarpoolNoticeVersion> {
    const { data } = await apiClient.post('/admin/carpool/notices', input)
    return data
  },
}

export default adminCarpoolAPI
