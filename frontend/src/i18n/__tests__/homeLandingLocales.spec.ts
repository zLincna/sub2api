import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const homeKeys = [
  'carpoolSpotlight.badge',
  'carpoolSpotlight.title',
  'carpoolSpotlight.description',
  'carpoolSpotlight.cta',
  'carpoolSpotlight.guarantee',
  'carpoolSpotlight.items.shared.title',
  'carpoolSpotlight.items.shared.desc',
  'carpoolSpotlight.items.queue.title',
  'carpoolSpotlight.items.queue.desc',
  'carpoolSpotlight.items.visible.title',
  'carpoolSpotlight.items.visible.desc',
  'carpoolSpotlight.items.revenue.title',
  'carpoolSpotlight.items.revenue.desc',
  'features.gptPool',
  'features.gptPoolDesc',
  'features.claudeCode',
  'features.claudeCodeDesc',
  'features.opsTeam',
  'features.opsTeamDesc',
  'features.noWater',
  'features.noWaterDesc',
  'features.transparentProfit',
  'features.transparentProfitDesc',
  'features.benefits',
  'features.benefitsDesc'
] as const

function readHomeMessage(locale: typeof zh, key: string): unknown {
  return key.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[segment]
  }, locale.home)
}

describe('home landing locale keys', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ] as const)('contains localized copy for every custom home key in %s', (_name, locale) => {
    for (const key of homeKeys) {
      const value = readHomeMessage(locale, key)
      expect(value, `home.${key}`).toEqual(expect.any(String))
      expect((value as string).trim(), `home.${key}`).not.toBe('')
      expect(value, `home.${key}`).not.toBe(`home.${key}`)
    }
  })
})
