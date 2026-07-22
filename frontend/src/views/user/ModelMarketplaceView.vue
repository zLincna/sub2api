<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 py-6 sm:px-6 lg:px-8">
      <section class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-5 p-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-900/25 dark:text-primary-300">
              <Icon name="sparkles" size="sm" />
              {{ t('modelMarketplace.badge') }}
            </div>
            <h1 class="text-2xl font-bold text-gray-950 dark:text-white sm:text-3xl">
              {{ t('modelMarketplace.title') }}
            </h1>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
              {{ t('modelMarketplace.description') }}
            </p>
          </div>

          <button
            type="button"
            class="btn btn-secondary h-10 flex-shrink-0"
            :disabled="loading"
            @click="loadChannels"
          >
            <Icon name="refresh" size="md" :class="{ 'animate-spin': loading }" />
            <span>{{ t('common.refresh', 'Refresh') }}</span>
          </button>
        </div>

        <div class="grid border-t border-gray-100 bg-gray-50/70 dark:border-dark-700 dark:bg-dark-800/50 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="stat in stats"
            :key="stat.label"
            class="border-b border-gray-100 px-5 py-4 last:border-b-0 dark:border-dark-700 sm:border-r sm:last:border-r-0 lg:border-b-0"
          >
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
            <div class="mt-1 text-2xl font-bold text-gray-950 dark:text-white">{{ stat.value }}</div>
          </div>
        </div>
      </section>

      <section class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="relative w-full lg:max-w-md">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model="searchQuery"
              type="text"
              class="input pl-10"
              :placeholder="t('modelMarketplace.searchPlaceholder')"
            />
          </div>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
            <div class="flex max-w-full gap-2 overflow-x-auto pb-1">
              <button
                v-for="item in platformFilters"
                :key="item.value"
                type="button"
                class="inline-flex h-9 flex-shrink-0 items-center gap-2 rounded-lg border px-3 text-sm font-medium transition-colors"
                :class="selectedPlatform === item.value
                  ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500/70 dark:bg-primary-900/25 dark:text-primary-300'
                  : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700'"
                @click="selectedPlatform = item.value"
              >
                <PlatformIcon
                  v-if="item.value !== ALL_PLATFORMS"
                  :platform="item.value as GroupPlatform"
                  size="sm"
                />
                <Icon v-else name="grid" size="sm" />
                {{ item.label }}
              </button>
            </div>

            <div class="marketplace-billing-select w-full sm:w-40">
              <Select
                v-model="selectedBillingMode"
                :options="billingModeOptions"
                :searchable="false"
              />
            </div>
          </div>
        </div>
      </section>

      <section
        v-if="loading"
        class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      >
        <div
          v-for="idx in 6"
          :key="idx"
          class="h-64 animate-pulse rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"
        >
          <div class="h-1 rounded-t-xl bg-gray-200 dark:bg-dark-700"></div>
          <div class="space-y-4 p-5">
            <div class="h-5 w-2/3 rounded bg-gray-200 dark:bg-dark-700"></div>
            <div class="h-4 w-full rounded bg-gray-100 dark:bg-dark-800"></div>
            <div class="h-4 w-3/4 rounded bg-gray-100 dark:bg-dark-800"></div>
            <div class="grid grid-cols-2 gap-3 pt-4">
              <div class="h-16 rounded bg-gray-100 dark:bg-dark-800"></div>
              <div class="h-16 rounded bg-gray-100 dark:bg-dark-800"></div>
            </div>
          </div>
        </div>
      </section>

      <section
        v-else-if="filteredModels.length === 0"
        class="rounded-xl border border-dashed border-gray-300 bg-white px-6 py-14 text-center dark:border-dark-600 dark:bg-dark-900"
      >
        <Icon name="inbox" size="xl" class="mx-auto mb-3 text-gray-400" />
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('modelMarketplace.emptyTitle') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('modelMarketplace.emptyDescription') }}
        </p>
      </section>

      <div v-else class="space-y-6">
        <section
          v-for="section in groupedModels"
          :key="section.platform"
          class="space-y-3"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <span
                :class="[
                  'inline-flex h-9 w-9 items-center justify-center rounded-lg',
                  platformBadgeLightClass(section.platform),
                ]"
              >
                <PlatformIcon :platform="section.platform as GroupPlatform" size="md" />
              </span>
              <div>
                <h2 class="text-lg font-semibold text-gray-950 dark:text-white">
                  {{ platformLabel(section.platform) }}
                </h2>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('modelMarketplace.platformCount', { count: section.models.length }) }}
                </p>
              </div>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <article
              v-for="model in section.models"
              :key="model.id"
              class="group overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-900"
            >
              <div :class="['h-1', platformAccentBarClass(model.platform)]"></div>
              <div class="flex min-h-[250px] flex-col p-5">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <span
                        :class="[
                          'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                          platformBadgeClass(model.platform),
                        ]"
                      >
                        <PlatformIcon :platform="model.platform as GroupPlatform" size="xs" />
                        {{ platformLabel(model.platform) }}
                      </span>
                      <span
                        class="rounded-md bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-800 dark:text-gray-300"
                      >
                        {{ billingModeLabel(model.pricing) }}
                      </span>
                    </div>
                    <h3 class="mt-3 break-words text-lg font-semibold leading-tight text-gray-950 dark:text-white">
                      {{ model.name }}
                    </h3>
                    <p v-if="model.description" class="mt-2 line-clamp-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
                      {{ model.description }}
                    </p>
                    <p class="mt-1 text-[11px] text-gray-400 dark:text-gray-500">
                      {{ t('modelMarketplace.channelsCount', { count: model.channels.length }) }}
                    </p>
                  </div>

                  <button
                    type="button"
                    class="inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-gray-200 text-gray-500 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500/60 dark:hover:text-primary-300"
                    :title="t('modelMarketplace.copyModel')"
                    @click="copyModelName(model.name)"
                  >
                    <Icon name="copy" size="sm" />
                  </button>
                </div>

                <div class="mt-4 grid grid-cols-2 gap-2">
                  <div
                    v-for="row in pricingRows(model)"
                    :key="row.label"
                    class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800"
                  >
                    <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ row.label }}</div>
                    <div class="mt-1 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ row.value }}</div>
                  </div>
                </div>

                <div class="mt-4 space-y-2">
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                    {{ t('modelMarketplace.availableGroups') }}
                  </div>
                  <div class="flex flex-wrap gap-1.5">
                    <GroupBadge
                      v-for="group in visibleGroups(model)"
                      :key="group.id"
                      :name="group.name"
                      :platform="group.platform as GroupPlatform"
                      :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                      :rate-multiplier="group.rate_multiplier"
                      :user-rate-multiplier="userGroupRates[group.id] ?? null"
                      always-show-rate
                    />
                    <span
                      v-if="model.groups.length > visibleGroupLimit"
                      class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500 dark:bg-dark-800 dark:text-gray-400"
                    >
                      +{{ model.groups.length - visibleGroupLimit }}
                    </span>
                    <span v-if="model.groups.length === 0" class="text-xs text-gray-400">
                      {{ t('modelMarketplace.noGroups') }}
                    </span>
                  </div>
                </div>

                <div class="mt-auto pt-4">
                  <div class="flex flex-wrap gap-2">
                    <span
                      v-for="channel in model.channels.slice(0, 3)"
                      :key="channel.name"
                      class="inline-flex max-w-full items-center gap-1 rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
                      :title="channel.description || channel.name"
                    >
                      <Icon name="server" size="xs" />
                      <span class="truncate">{{ channel.name }}</span>
                    </span>
                    <span
                      v-if="model.channels.length > 3"
                      class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400"
                    >
                      +{{ model.channels.length - 3 }}
                    </span>
                  </div>
                </div>
              </div>
            </article>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Select from '@/components/common/Select.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserAvailableGroup,
  type UserSupportedModelPricing,
} from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import modelMarketplaceAPI, { type ModelMarketplaceConfigItem } from '@/api/modelMarketplace'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  type BillingMode,
} from '@/constants/channel'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatScaled } from '@/utils/pricing'
import {
  platformAccentBarClass,
  platformBadgeClass,
  platformBadgeLightClass,
  platformLabel,
} from '@/utils/platformColors'

interface ModelChannel {
  name: string
  description: string
}

interface MarketplaceModel {
  id: string
  name: string
  platform: string
  description: string
  pricing: UserSupportedModelPricing | null
  channels: ModelChannel[]
  groups: UserAvailableGroup[]
}

interface PricingRow {
  label: string
  value: string
}

interface PlatformGroup {
  platform: string
  models: MarketplaceModel[]
}

const ALL_PLATFORMS = 'all'
const visibleGroupLimit = 4
const perMillionScale = 1_000_000

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const channels = ref<UserAvailableChannel[]>([])
const managedModels = ref<MarketplaceModel[] | null>(null)
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const selectedPlatform = ref(ALL_PLATFORMS)
const selectedBillingMode = ref<BillingMode | 'all' | 'none'>('all')

const marketplaceModels = computed<MarketplaceModel[]>(() => {
  const map = new Map<string, MarketplaceModel>()

  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        const id = `${section.platform}::${model.name}`
        const current = map.get(id)
        if (!current) {
          map.set(id, {
            id,
            name: model.name,
            platform: model.platform || section.platform,
            description: channel.description || '',
            pricing: model.pricing,
            channels: [{ name: channel.name, description: channel.description || '' }],
            groups: uniqueGroups(section.groups),
          })
          continue
        }

        if (!current.pricing && model.pricing) current.pricing = model.pricing
        if (!current.channels.some((item) => item.name === channel.name)) {
          current.channels.push({ name: channel.name, description: channel.description || '' })
        }
        current.groups = uniqueGroups([...current.groups, ...section.groups])
      }
    }
  }

  if (managedModels.value !== null) return managedModels.value
  const dynamicModels = Array.from(map.values())
  return dynamicModels.length > 0 ? dynamicModels : builtInMarketplaceModels
})

const platformFilters = computed(() => [
  { value: ALL_PLATFORMS, label: t('modelMarketplace.filters.allPlatforms') },
  ...Array.from(new Set(marketplaceModels.value.map((item) => item.platform)))
    .sort((a, b) => platformLabel(a).localeCompare(platformLabel(b)))
    .map((platform) => ({ value: platform, label: platformLabel(platform) })),
])

const billingModeOptions = computed(() => [
  { value: 'all', label: t('modelMarketplace.filters.allBilling') },
  { value: BILLING_MODE_TOKEN, label: t('modelMarketplace.pricing.billingModeToken') },
  { value: BILLING_MODE_PER_REQUEST, label: t('modelMarketplace.pricing.billingModePerRequest') },
  { value: BILLING_MODE_IMAGE, label: t('modelMarketplace.pricing.billingModeImage') },
  { value: 'none', label: t('modelMarketplace.filters.noPricing') },
])

const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return marketplaceModels.value.filter((model) => {
    if (selectedPlatform.value !== ALL_PLATFORMS && model.platform !== selectedPlatform.value) {
      return false
    }
    if (selectedBillingMode.value === 'none' && model.pricing) return false
    if (
      selectedBillingMode.value !== 'all' &&
      selectedBillingMode.value !== 'none' &&
      model.pricing?.billing_mode !== selectedBillingMode.value
    ) {
      return false
    }
    if (!q) return true
    return (
      model.name.toLowerCase().includes(q) ||
      model.platform.toLowerCase().includes(q) ||
      model.channels.some((channel) =>
        `${channel.name} ${channel.description}`.toLowerCase().includes(q),
      ) ||
      model.groups.some((group) => group.name.toLowerCase().includes(q))
    )
  })
})

const groupedModels = computed<PlatformGroup[]>(() => {
  const map = new Map<string, MarketplaceModel[]>()
  for (const model of filteredModels.value) {
    const list = map.get(model.platform) || []
    list.push(model)
    map.set(model.platform, list)
  }
  return Array.from(map.entries())
    .map(([platform, models]) => ({ platform, models }))
})

const stats = computed(() => [
  { label: t('modelMarketplace.stats.models'), value: marketplaceModels.value.length },
  { label: t('modelMarketplace.stats.platforms'), value: platformFilters.value.length - 1 },
  { label: t('modelMarketplace.stats.groups'), value: uniqueGroupCount.value },
  { label: t('modelMarketplace.stats.channels'), value: uniqueChannelCount.value },
])

const uniqueGroupCount = computed(() => {
  const ids = new Set<number>()
  for (const model of marketplaceModels.value) {
    for (const group of model.groups) ids.add(group.id)
  }
  return ids.size
})

const uniqueChannelCount = computed(() => {
  const names = new Set<string>()
  for (const model of marketplaceModels.value) {
    for (const channel of model.channels) names.add(channel.name)
  }
  return names.size
})

const builtInGroups: UserAvailableGroup[] = [
  createBuiltInGroup(90001, 'OpenAI Pro 20x 号池', 'openai', 0.1),
  createBuiltInGroup(90002, 'OpenAI Pro 100+ 稳定池', 'openai', 0.12),
  createBuiltInGroup(90003, 'Claude Code 低倍率', 'anthropic', 0.02),
  createBuiltInGroup(90004, 'Claude Code 顶级模型', 'anthropic', 0.04),
]

const builtInMarketplaceModels: MarketplaceModel[] = [
  createBuiltInModel({
    name: 'gpt-5.6-sol',
    platform: 'openai',
    groups: [builtInGroups[0], builtInGroups[1]],
    channels: [
      { name: 'OpenAI Pro 20x', description: 'GPT-5.6 Sol，支持超长上下文、视觉输入、工具调用与高强度推理任务。' },
      { name: 'OpenAI Pro 100+', description: '100+ Pro 账号集群，面向高峰期稳定调度。' },
    ],
    pricing: tokenPricing(5, 30, null, 0.5),
  }),
  createBuiltInModel({
    name: 'gpt-5.6-terra',
    platform: 'openai',
    groups: [builtInGroups[0], builtInGroups[1]],
    channels: [
      { name: 'OpenAI Pro 20x', description: 'GPT-5.6 Terra，支持超长上下文、视觉输入、工具调用与复杂工程任务。' },
      { name: 'OpenAI Pro 100+', description: '100+ Pro 账号集群，面向高峰期稳定调度。' },
    ],
    pricing: tokenPricing(5, 30, null, 0.5),
  }),
  createBuiltInModel({
    name: 'gpt-5.6-luna',
    platform: 'openai',
    groups: [builtInGroups[0], builtInGroups[1]],
    channels: [
      { name: 'OpenAI Pro 20x', description: 'GPT-5.6 Luna，支持超长上下文、视觉输入、工具调用与深度推理。' },
      { name: 'OpenAI Pro 100+', description: '100+ Pro 账号集群，面向高峰期稳定调度。' },
    ],
    pricing: tokenPricing(5, 30, null, 0.5),
  }),
  createBuiltInModel({
    name: 'gpt-5-codex',
    platform: 'openai',
    groups: [builtInGroups[0], builtInGroups[1]],
    channels: [
      { name: 'OpenAI Pro 20x', description: '稳定 20x Pro 号池，适合 Codex、IDE 与高强度代码任务。' },
      { name: 'OpenAI Pro 100+', description: '100+ Pro 账号集群，面向高峰期稳定调度。' },
    ],
    pricing: tokenPricing(1.25, 10, 1.25, 0.125),
  }),
  createBuiltInModel({
    name: 'gpt-5',
    platform: 'openai',
    groups: [builtInGroups[0], builtInGroups[1]],
    channels: [
      { name: 'OpenAI Pro 20x', description: '通用旗舰模型，适合复杂推理、代码与长上下文任务。' },
    ],
    pricing: tokenPricing(1.25, 10, 1.25, 0.125),
  }),
  createBuiltInModel({
    name: 'gpt-5-mini',
    platform: 'openai',
    groups: [builtInGroups[0], builtInGroups[1]],
    channels: [
      { name: 'OpenAI Pro 20x', description: '高性价比快速模型，适合日常问答与轻量编码。' },
    ],
    pricing: tokenPricing(0.25, 2, 0.25, 0.025),
  }),
  createBuiltInModel({
    name: 'gpt-4.1',
    platform: 'openai',
    groups: [builtInGroups[0]],
    channels: [
      { name: 'OpenAI Pro 20x', description: '稳定通用模型，兼容大量 OpenAI 生态客户端。' },
    ],
    pricing: tokenPricing(2, 8, 0.5, 0.5),
  }),
  createBuiltInModel({
    name: 'claude-sonnet-4-5',
    platform: 'anthropic',
    groups: [builtInGroups[2], builtInGroups[3]],
    channels: [
      { name: 'Claude Code 低倍率', description: 'Claude Code 主力模型，适合代码生成、重构与复杂项目执行。' },
    ],
    pricing: tokenPricing(3, 15, 3.75, 0.3),
  }),
  createBuiltInModel({
    name: 'claude-opus-4-1',
    platform: 'anthropic',
    groups: [builtInGroups[3]],
    channels: [
      { name: 'Claude Code 顶级模型', description: '顶级推理与复杂代码任务模型，适合困难问题和深度规划。' },
    ],
    pricing: tokenPricing(15, 75, 18.75, 1.5),
  }),
  createBuiltInModel({
    name: 'claude-sonnet-4',
    platform: 'anthropic',
    groups: [builtInGroups[2], builtInGroups[3]],
    channels: [
      { name: 'Claude Code 低倍率', description: '成熟稳定的 Claude Code 模型，适合日常开发与长任务。' },
    ],
    pricing: tokenPricing(3, 15, 3.75, 0.3),
  }),
  createBuiltInModel({
    name: 'claude-haiku-4-5',
    platform: 'anthropic',
    groups: [builtInGroups[2]],
    channels: [
      { name: 'Claude Code 低倍率', description: '快速低成本模型，适合轻量代码、摘要和批量处理。' },
    ],
    pricing: tokenPricing(1, 5, 1.25, 0.1),
  }),
]

function createBuiltInGroup(
  id: number,
  name: string,
  platform: string,
  rateMultiplier: number,
): UserAvailableGroup {
  return {
    id,
    name,
    platform,
    subscription_type: 'standard',
    rate_multiplier: rateMultiplier,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: rateMultiplier,
    is_exclusive: false,
  }
}

function createBuiltInModel(input: {
  name: string
  platform: string
  groups: UserAvailableGroup[]
  channels: ModelChannel[]
  pricing: UserSupportedModelPricing
}): MarketplaceModel {
  return {
    id: `built-in::${input.platform}::${input.name}`,
    name: input.name,
    platform: input.platform,
    description: input.channels[0]?.description || '',
    pricing: input.pricing,
    groups: input.groups,
    channels: input.channels,
  }
}

function managedModelToMarketplace(
  model: ModelMarketplaceConfigItem,
  index: number,
): MarketplaceModel {
  const group: UserAvailableGroup = {
    id: -(index + 1),
    name: model.group_name || model.channel_name || model.platform,
    platform: model.platform,
    subscription_type: 'standard',
    rate_multiplier: model.rate_multiplier,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: model.rate_multiplier,
    is_exclusive: false,
  }
  return {
    id: `managed::${model.platform}::${model.id}`,
    name: model.id,
    platform: model.platform,
    description: model.description,
    pricing: managedPricing(model),
    groups: model.group_name || model.channel_name ? [group] : [],
    channels: model.channel_name
      ? [{ name: model.channel_name, description: model.channel_description || model.description }]
      : [],
  }
}

function managedPricing(model: ModelMarketplaceConfigItem): UserSupportedModelPricing {
  return {
    billing_mode: model.billing_mode,
    input_price: perTokenPrice(model.input_price_per_million),
    output_price: perTokenPrice(model.output_price_per_million),
    cache_write_price: perTokenPrice(model.cache_write_price_per_million),
    cache_read_price: perTokenPrice(model.cache_read_price_per_million),
    image_input_price: null,
    image_output_price: model.image_output_price_per_request,
    per_request_price: model.per_request_price,
    intervals: [],
  }
}

function perTokenPrice(value: number | null): number | null {
  return value == null ? null : value / perMillionScale
}

function tokenPricing(
  inputPerMillion: number,
  outputPerMillion: number,
  cacheWritePerMillion: number | null,
  cacheReadPerMillion: number | null,
): UserSupportedModelPricing {
  return {
    billing_mode: BILLING_MODE_TOKEN,
    input_price: inputPerMillion / perMillionScale,
    output_price: outputPerMillion / perMillionScale,
    cache_write_price: cacheWritePerMillion == null ? null : cacheWritePerMillion / perMillionScale,
    cache_read_price: cacheReadPerMillion == null ? null : cacheReadPerMillion / perMillionScale,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
  }
}

function uniqueGroups(groups: UserAvailableGroup[]): UserAvailableGroup[] {
  const map = new Map<number, UserAvailableGroup>()
  for (const group of groups) map.set(group.id, group)
  return Array.from(map.values()).sort((a, b) => {
    if (a.is_exclusive !== b.is_exclusive) return a.is_exclusive ? -1 : 1
    return a.name.localeCompare(b.name)
  })
}

function visibleGroups(model: MarketplaceModel): UserAvailableGroup[] {
  return model.groups.slice(0, visibleGroupLimit)
}

function billingModeLabel(pricing: UserSupportedModelPricing | null): string {
  switch (pricing?.billing_mode) {
    case BILLING_MODE_TOKEN:
      return t('modelMarketplace.pricing.billingModeToken')
    case BILLING_MODE_PER_REQUEST:
      return t('modelMarketplace.pricing.billingModePerRequest')
    case BILLING_MODE_IMAGE:
      return t('modelMarketplace.pricing.billingModeImage')
    default:
      return t('modelMarketplace.pricing.noPricing')
  }
}

function pricingRows(model: MarketplaceModel): PricingRow[] {
  const pricing = model.pricing
  if (!pricing) {
    return [
      { label: t('modelMarketplace.pricing.billingMode'), value: t('modelMarketplace.pricing.noPricing') },
      { label: t('modelMarketplace.pricing.groupRate'), value: rateSummary(model) },
    ]
  }

  if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    const rows: PricingRow[] = [
      {
        label: t('modelMarketplace.pricing.inputPrice'),
        value: `${formatScaled(pricing.input_price, perMillionScale)} ${t('modelMarketplace.pricing.unitPerMillion')}`,
      },
      {
        label: t('modelMarketplace.pricing.outputPrice'),
        value: `${formatScaled(pricing.output_price, perMillionScale)} ${t('modelMarketplace.pricing.unitPerMillion')}`,
      },
    ]
    if (pricing.cache_read_price != null) {
      rows.push({
        label: t('modelMarketplace.pricing.cacheReadPrice'),
        value: `${formatScaled(pricing.cache_read_price, perMillionScale)} ${t('modelMarketplace.pricing.unitPerMillion')}`,
      })
    }
    if (pricing.cache_write_price != null) {
      rows.push({
        label: t('modelMarketplace.pricing.cacheWritePrice'),
        value: `${formatScaled(pricing.cache_write_price, perMillionScale)} ${t('modelMarketplace.pricing.unitPerMillion')}`,
      })
    }
    if (pricing.image_output_price != null && pricing.image_output_price > 0) {
      rows.push({
        label: t('modelMarketplace.pricing.imageOutputPrice'),
        value: `${formatScaled(pricing.image_output_price, perMillionScale)} ${t('modelMarketplace.pricing.unitPerMillion')}`,
      })
    }
    rows.push({ label: t('modelMarketplace.pricing.groupRate'), value: rateSummary(model) })
    return rows.slice(0, 4)
  }

  if (pricing.billing_mode === BILLING_MODE_IMAGE) {
    return [
      {
        label: t('modelMarketplace.pricing.imageOutputPrice'),
        value: `${formatScaled(pricing.image_output_price, 1)} ${t('modelMarketplace.pricing.unitPerRequest')}`,
      },
      { label: t('modelMarketplace.pricing.groupRate'), value: rateSummary(model) },
    ]
  }

  return [
    {
      label: t('modelMarketplace.pricing.perRequestPrice'),
      value: `${formatScaled(pricing.per_request_price, 1)} ${t('modelMarketplace.pricing.unitPerRequest')}`,
    },
    { label: t('modelMarketplace.pricing.groupRate'), value: rateSummary(model) },
  ]
}

function rateSummary(model: MarketplaceModel): string {
  const rates = model.groups
    .map((group) => userGroupRates.value[group.id] ?? group.rate_multiplier)
    .filter((rate): rate is number => typeof rate === 'number')
  if (rates.length === 0) return '-'
  const min = Math.min(...rates)
  const max = Math.max(...rates)
  return min === max ? `${min}x` : `${min}x - ${max}x`
}

async function copyModelName(name: string): Promise<void> {
  await copyToClipboard(name, t('modelMarketplace.copied'))
}

async function loadChannels(): Promise<void> {
  loading.value = true
  try {
    const [list, rates, marketplaceConfig] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
      modelMarketplaceAPI.getModelMarketplaceConfig().catch((err: unknown) => {
        console.error('Failed to load model marketplace config:', err)
        return null
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
    managedModels.value = marketplaceConfig
      ? marketplaceConfig.models
          .filter((model) => model.enabled)
          .map(managedModelToMarketplace)
      : null
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>

<style scoped>
.marketplace-billing-select :deep(.select-trigger) {
  min-height: 2.25rem;
  border-radius: 0.5rem;
  padding: 0.45rem 0.75rem;
  font-size: 0.875rem;
}
</style>
