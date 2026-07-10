import { apiClient } from './client'

export type ModelMarketplaceBillingMode = 'token' | 'per_request' | 'image'

export interface ModelMarketplaceConfigItem {
  id: string
  platform: string
  description: string
  channel_name: string
  channel_description: string
  group_name: string
  rate_multiplier: number
  billing_mode: ModelMarketplaceBillingMode
  input_price_per_million: number | null
  output_price_per_million: number | null
  cache_write_price_per_million: number | null
  cache_read_price_per_million: number | null
  image_output_price_per_request: number | null
  per_request_price: number | null
  enabled: boolean
}

export interface ModelMarketplaceConfig {
  models: ModelMarketplaceConfigItem[]
}

export async function getModelMarketplaceConfig(): Promise<ModelMarketplaceConfig> {
  const { data } = await apiClient.get<ModelMarketplaceConfig>('/settings/model-marketplace')
  return data
}

export const modelMarketplaceAPI = { getModelMarketplaceConfig }

export default modelMarketplaceAPI
