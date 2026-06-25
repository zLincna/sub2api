import type { CreateOrderResult } from './payment'

export interface CarpoolVehicleType {
  id: number
  product: string
  plan_tier: string
  multiplier: string
  name: string
  seat_count: number
  total_price: number
  unit_price: number
  service_days: number
  refund_wait_hours: number
  completed_base_count: number
  enabled: boolean
  support_revenue_pool: boolean
  require_static_ip: boolean
  wait_duration_options: number[]
  refund_methods: string[]
  description: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface CarpoolSession {
  id: number
  vehicle_type_id: number
  session_no: string
  status: string
  seat_count: number
  paid_count: number
  started_at?: string
  filled_at?: string
  provisioned_at?: string
  service_started_at?: string
  service_ended_at?: string
  account_info?: Record<string, unknown>
  proxy_info?: Record<string, unknown>
  communication?: Record<string, unknown>
  admin_notes?: string
  created_at: string
  updated_at: string
  edges?: {
    vehicle_type?: CarpoolVehicleType
    participants?: CarpoolParticipant[]
    vouchers?: CarpoolVoucher[]
  }
}

export interface CarpoolVoucher {
  id: number
  session_id: number
  file_url: string
  file_name: string
  description?: string
  uploaded_by: number
  created_at: string
}

export interface CarpoolParticipant {
  id: number
  session_id?: number
  vehicle_type_id: number
  user_id: number
  payment_order_id?: number
  status: string
  amount: number
  wait_until: string
  refund_method: string
  notice_version_id?: number
  notice_accepted_at?: string
  notice_accept_ip?: string
  joined_at?: string
  paid_at?: string
  refunded_at?: string
  created_at: string
  updated_at: string
  edges?: {
    session?: CarpoolSession
    vehicle_type?: CarpoolVehicleType
    user?: { id: number; email: string; username?: string }
  }
}

export interface CarpoolNoticeVersion {
  id: number
  title: string
  content_md: string
  version: number
  active: boolean
  published_at?: string
  created_at: string
  updated_at: string
}

export interface CarpoolCard {
  vehicle_type: CarpoolVehicleType
  session: CarpoolSession
  paid_count: number
  seat_count: number
  completed_count: number
  real_completed_count: number
  display_completed_count: number
  refund_wait_hours: number
  refund_available_at?: string
  completed_base_count: number
}

export interface CarpoolJoinRequest {
  vehicle_type_id: number
  notice_version_id: number
  notice_accepted: boolean
  payment_type: string
  return_url?: string
  payment_source?: string
  openid?: string
}

export interface CarpoolJoinResponse {
  participant: CarpoolParticipant
  order: CreateOrderResult
}

export interface CarpoolVehicleTypeInput {
  product: string
  plan_tier: string
  multiplier: string
  name: string
  seat_count: number
  total_price: number
  unit_price: number
  service_days: number
  refund_wait_hours: number
  completed_base_count: number
  enabled: boolean
  support_revenue_pool: boolean
  require_static_ip: boolean
  wait_duration_options: number[]
  refund_methods: string[]
  description: string
  sort_order: number
}

export interface CarpoolNoticeInput {
  title: string
  content_md: string
  active: boolean
}

export interface CarpoolVoucherInput {
  file_url: string
  file_name: string
  description: string
}

export interface CarpoolAdminUsageSummary {
  request_count: number
  total_tokens: number
  total_cost: number
  total_actual_cost: number
}

export interface CarpoolUserUsageWindows {
  five_hour: CarpoolAdminUsageSummary
  seven_day: CarpoolAdminUsageSummary
}

export interface CarpoolAccountWindowUsage {
  account_id: number
  account_name: string
  window: string
  utilization: number
  resets_at?: string
  remaining_seconds: number
  usage: CarpoolAdminUsageSummary
}

export interface CarpoolUserMemberUsage {
  participant_id: number
  user_id: number
  display_name: string
  initial: string
  avatar_url?: string
  is_self: boolean
  status: string
  usage: CarpoolAdminUsageSummary
  windows: CarpoolUserUsageWindows
}

export interface CarpoolUserDetail {
  participant: CarpoolParticipant
  session?: CarpoolSession
  members: CarpoolUserMemberUsage[]
  total_usage: CarpoolAdminUsageSummary
  total_windows: CarpoolUserUsageWindows
  account_windows: CarpoolAccountWindowUsage[]
}

export interface CarpoolAdminParticipantRow {
  participant: CarpoolParticipant
  usage: CarpoolAdminUsageSummary
  user?: { id: number; email: string; username?: string }
}

export interface CarpoolAdminSessionRow {
  session: CarpoolSession
  participants: CarpoolAdminParticipantRow[]
  usage: CarpoolAdminUsageSummary
}

export interface CarpoolAdminManagementSummary {
  completed_sessions: number
  paid_members: number
  active_members: number
  total_paid_amount: number
  total_tokens: number
  total_actual_cost: number
  by_status: Record<string, number>
  by_segment: Array<{
    label: string
    sessions: number
    paid_members: number
    amount: number
  }>
}

export interface CarpoolAdminManagementResponse {
  summary: CarpoolAdminManagementSummary
  items: CarpoolAdminSessionRow[]
  total: number
  page: number
  page_size: number
}
