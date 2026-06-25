import { apiClient } from './client'
import type {
  CarpoolCard,
  CarpoolJoinRequest,
  CarpoolJoinResponse,
  CarpoolNoticeVersion,
  CarpoolParticipant,
  CarpoolUserDetail,
} from '@/types/carpool'

export const carpoolAPI = {
  async listCards(): Promise<CarpoolCard[]> {
    const { data } = await apiClient.get('/carpool/cards')
    return data
  },

  async currentNotice(): Promise<CarpoolNoticeVersion> {
    const { data } = await apiClient.get('/carpool/notice/current')
    return data
  },

  async join(input: CarpoolJoinRequest): Promise<CarpoolJoinResponse> {
    const { data } = await apiClient.post('/carpool/join', input)
    return data
  },

  async my(): Promise<CarpoolParticipant[]> {
    const { data } = await apiClient.get('/carpool/my')
    return data
  },

  async myDetail(id: number): Promise<CarpoolUserDetail> {
    const { data } = await apiClient.get(`/carpool/my/${id}/detail`)
    return data
  },

  async requestRefund(id: number, input: { refund_method: string }): Promise<CarpoolParticipant> {
    const { data } = await apiClient.post(`/carpool/my/${id}/refund`, input)
    return data
  },
}

export default carpoolAPI
