export type ChatIntegrationType = 'web' | 'custom-protocol' | 'fluent' | 'ccswitch'

export interface ChatIntegrationTemplate {
  id: string
  name: string
  type: ChatIntegrationType
  template: string
}

export const CHAT_INTEGRATION_TEMPLATES: ChatIntegrationTemplate[] = [
  {
    id: 'cherry-studio',
    name: 'Cherry Studio',
    type: 'custom-protocol',
    template: 'cherrystudio://providers/api-keys?v=1&data={cherryConfig}'
  },
  {
    id: 'aionui',
    name: 'AionUI',
    type: 'custom-protocol',
    template: 'aionui://provider/add?v=1&data={aionuiConfig}'
  },
  {
    id: 'fluentread',
    name: '流畅阅读',
    type: 'fluent',
    template: 'fluentread'
  },
  {
    id: 'ccswitch',
    name: 'CC Switch',
    type: 'ccswitch',
    template: 'ccswitch'
  },
  {
    id: 'deepchat',
    name: 'DeepChat',
    type: 'custom-protocol',
    template: 'deepchat://provider/install?v=1&data={deepchatConfig}'
  },
  {
    id: 'lobe-chat',
    name: 'Lobe Chat 官方示例',
    type: 'web',
    template:
      'https://chat-preview.lobehub.com/?settings={"keyVaults":{"openai":{"apiKey":"{key}","baseURL":"{address}/v1"}}}'
  },
  {
    id: 'ai-as-workspace',
    name: 'AI as Workspace',
    type: 'web',
    template:
      'https://aiaw.app/set-provider?provider={"type":"openai","settings":{"apiKey":"{key}","baseURL":"{address}/v1","compatibility":"strict"}}'
  },
  {
    id: 'ama',
    name: 'AMA 问天',
    type: 'custom-protocol',
    template: 'ama://set-api-key?server={address}&key={key}'
  },
  {
    id: 'opencat',
    name: 'OpenCat',
    type: 'custom-protocol',
    template: 'opencat://team/join?domain={address}&token={key}'
  }
]

export function getChatIntegrationBaseUrl(apiBaseUrl?: string | null): string {
  const source = apiBaseUrl?.trim() || window.location.origin
  return source.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
}

export function normalizeChatIntegrationApiKey(apiKey: string): string {
  const trimmed = apiKey.trim()
  if (!trimmed) return ''
  return trimmed.startsWith('sk-') ? trimmed : `sk-${trimmed}`
}

function encodeBase64Utf8(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary)
}

function replaceAll(source: string, token: string, value: string): string {
  return source.split(token).join(value)
}

export function resolveChatIntegrationUrl(
  template: string,
  apiKey: string,
  baseUrl: string,
  providerId = 'sub2api'
): string {
  let url = template
  const safeApiKey = normalizeChatIntegrationApiKey(apiKey)
  const safeBaseUrl = baseUrl.replace(/\/+$/, '')

  if (url.includes('{cherryConfig}')) {
    const payload = {
      id: providerId,
      baseUrl: safeBaseUrl,
      apiKey: safeApiKey
    }
    return replaceAll(url, '{cherryConfig}', encodeURIComponent(encodeBase64Utf8(JSON.stringify(payload))))
  }

  if (url.includes('{aionuiConfig}')) {
    const payload = {
      platform: providerId,
      baseUrl: safeBaseUrl,
      apiKey: safeApiKey
    }
    return replaceAll(url, '{aionuiConfig}', encodeURIComponent(encodeBase64Utf8(JSON.stringify(payload))))
  }

  if (url.includes('{deepchatConfig}')) {
    const payload = {
      id: providerId,
      baseUrl: safeBaseUrl,
      apiKey: safeApiKey
    }
    return replaceAll(url, '{deepchatConfig}', encodeURIComponent(encodeBase64Utf8(JSON.stringify(payload))))
  }

  url = replaceAll(url, '{address}', encodeURIComponent(safeBaseUrl))
  url = replaceAll(url, '{key}', safeApiKey)
  return url
}

export function sendToFluentRead(apiKey: string, baseUrl: string, providerId = 'sub2api'): boolean {
  const container = document.getElementById('fluent-new-api-container')
  if (!container) {
    return false
  }

  container.dispatchEvent(
    new CustomEvent('fluent:prefill', {
      detail: {
        id: providerId,
        baseUrl,
        apiKey: normalizeChatIntegrationApiKey(apiKey)
      }
    })
  )

  return true
}
