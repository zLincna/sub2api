import { describe, expect, it } from 'vitest'
import {
  getChatIntegrationBaseUrl,
  normalizeChatIntegrationApiKey,
  resolveChatIntegrationUrl
} from '@/utils/chatIntegrations'

function decodeConfigFromUrl(url: string, tokenName: string): Record<string, string> {
  const encoded = new URL(url).searchParams.get(tokenName) || ''
  return JSON.parse(atob(decodeURIComponent(encoded)))
}

describe('chatIntegrations utils', () => {
  it('normalizes API keys with an sk prefix', () => {
    expect(normalizeChatIntegrationApiKey('abc123')).toBe('sk-abc123')
    expect(normalizeChatIntegrationApiKey('sk-abc123')).toBe('sk-abc123')
  })

  it('normalizes configured API base URLs to the site root', () => {
    expect(getChatIntegrationBaseUrl('https://api.example.com/v1')).toBe('https://api.example.com')
    expect(getChatIntegrationBaseUrl('https://api.example.com///')).toBe('https://api.example.com')
  })

  it('builds Cherry Studio config payloads', () => {
    const url = resolveChatIntegrationUrl(
      'cherrystudio://providers/api-keys?v=1&data={cherryConfig}',
      'abc123',
      'https://api.example.com',
      'zemra-ai'
    )
    const payload = decodeConfigFromUrl(url, 'data')

    expect(payload).toEqual({
      id: 'zemra-ai',
      baseUrl: 'https://api.example.com',
      apiKey: 'sk-abc123'
    })
  })

  it('replaces address and key placeholders for web presets', () => {
    const url = resolveChatIntegrationUrl(
      'https://example.com/?base={address}/v1&key={key}',
      'abc123',
      'https://api.example.com',
      'sub2api'
    )

    expect(url).toBe('https://example.com/?base=https%3A%2F%2Fapi.example.com/v1&key=sk-abc123')
  })
})
