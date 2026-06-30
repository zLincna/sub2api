<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <section class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('lottery.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.remaining', { count: status?.chances.remaining ?? 0 }) }}</p>
            </div>
            <div class="grid grid-cols-3 gap-2 text-center">
              <div class="rounded-md bg-emerald-50 px-3 py-2 dark:bg-emerald-900/20">
                <div class="text-lg font-semibold text-emerald-700 dark:text-emerald-300">{{ status?.chances.granted ?? 0 }}</div>
                <div class="text-xs text-emerald-700/70 dark:text-emerald-300/70">{{ t('lottery.granted') }}</div>
              </div>
              <div class="rounded-md bg-blue-50 px-3 py-2 dark:bg-blue-900/20">
                <div class="text-lg font-semibold text-blue-700 dark:text-blue-300">{{ status?.chances.used ?? 0 }}</div>
                <div class="text-xs text-blue-700/70 dark:text-blue-300/70">{{ t('lottery.used') }}</div>
              </div>
              <div class="rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-800">
                <div class="text-lg font-semibold text-gray-700 dark:text-gray-200">{{ status?.chances.expired ?? 0 }}</div>
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('lottery.expired') }}</div>
              </div>
            </div>
          </div>

          <div v-if="status && !status.config.enabled" class="mt-6 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200">
            {{ t('lottery.disabled') }}
          </div>

          <div class="mt-8 flex flex-col items-center gap-6">
            <div class="relative h-[22rem] w-[22rem] max-w-full sm:h-[28rem] sm:w-[28rem]">
              <div class="absolute left-1/2 top-[-2px] z-20 -translate-x-1/2">
                <div class="h-0 w-0 border-x-[18px] border-t-[38px] border-x-transparent border-t-red-500 drop-shadow-lg"></div>
              </div>
              <div
                class="lottery-wheel relative h-full w-full rounded-full border-[12px] border-white bg-white shadow-2xl dark:border-dark-800 dark:bg-dark-900"
                :class="{ spinning }"
              >
                <svg class="h-full w-full overflow-visible rounded-full" viewBox="0 0 400 400" aria-hidden="true">
                  <defs>
                    <filter id="lotterySegmentShadow" x="-20%" y="-20%" width="140%" height="140%">
                      <feDropShadow dx="0" dy="4" stdDeviation="5" flood-color="#0f172a" flood-opacity="0.18" />
                    </filter>
                  </defs>
                  <g filter="url(#lotterySegmentShadow)">
                    <path
                      v-for="segment in wheelSegments"
                      :key="segment.id"
                      :d="segment.path"
                      :fill="segment.color"
                      stroke="rgba(255,255,255,0.76)"
                      stroke-width="2"
                    />
                  </g>
                  <g v-for="segment in wheelSegments" :key="`${segment.id}-label`" :transform="segment.transform">
                    <text
                      text-anchor="middle"
                      dominant-baseline="middle"
                      class="lottery-segment-title"
                    >
                      <tspan x="0" dy="-0.45em">{{ segment.title }}</tspan>
                      <tspan x="0" dy="1.25em" class="lottery-segment-subtitle">{{ segment.subtitle }}</tspan>
                    </text>
                  </g>
                </svg>
                <div class="pointer-events-none absolute inset-[27%] rounded-full bg-slate-950/15 shadow-inner dark:bg-black/20"></div>
                <div class="absolute inset-0 flex items-center justify-center">
                  <button
                    type="button"
                    class="z-10 flex h-28 w-28 flex-col items-center justify-center rounded-full bg-red-500 text-white shadow-xl ring-8 ring-white/80 transition hover:bg-red-600 disabled:cursor-not-allowed disabled:bg-gray-400 dark:ring-dark-900/80 sm:h-32 sm:w-32"
                    :disabled="drawing || spinning || !status?.config.enabled || (status?.chances.remaining ?? 0) <= 0"
                    @click="draw"
                  >
                    <span class="text-base font-bold">{{ drawing ? t('lottery.drawing') : t('lottery.draw') }}</span>
                    <span class="text-xs opacity-90">{{ t('lottery.once') }}</span>
                  </button>
                </div>
              </div>
            </div>

            <div v-if="lastPrize" class="rounded-lg border border-emerald-200 bg-emerald-50 px-5 py-3 text-center dark:border-emerald-800/60 dark:bg-emerald-900/20">
              <p class="text-sm text-emerald-700 dark:text-emerald-300">{{ t('lottery.winPrefix') }}</p>
              <p class="text-xl font-bold text-emerald-800 dark:text-emerald-200">{{ lastPrize.name }} · {{ prizeSubtitle(lastPrize) }}</p>
              <p v-if="lastPrize.description" class="mt-1 max-w-xl text-sm text-emerald-700/80 dark:text-emerald-200/80">{{ lastPrize.description }}</p>
            </div>
          </div>
        </section>

        <aside class="space-y-4">
          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('lottery.rules') }}</h3>
            <p class="mt-3 whitespace-pre-line text-sm leading-6 text-gray-600 dark:text-dark-300">{{ status?.config.rule_text || '-' }}</p>
          </section>
          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('lottery.sources') }}</h3>
            <div class="mt-3 space-y-2 text-sm">
              <div v-for="source in sourceRows" :key="source.key" class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-dark-400">{{ source.label }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ source.value }}</span>
              </div>
            </div>
          </section>
        </aside>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('lottery.records') }}</h3>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadStatus">{{ t('common.refresh') }}</button>
        </div>
        <div v-if="!records.length" class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.noRecords') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead class="text-left text-xs uppercase text-gray-500 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2">{{ t('lottery.prize') }}</th>
                <th class="px-3 py-2">{{ t('lottery.amount') }}</th>
                <th class="px-3 py-2">{{ t('lottery.source') }}</th>
                <th class="px-3 py-2">{{ t('lottery.time') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="record in records" :key="record.id">
                <td class="px-3 py-2">
                  <div class="font-medium text-gray-900 dark:text-white">{{ record.prize_name }}</div>
                  <div v-if="record.prize_description" class="mt-0.5 max-w-md text-xs text-gray-500 dark:text-dark-400">{{ record.prize_description }}</div>
                </td>
                <td class="px-3 py-2 text-emerald-600 dark:text-emerald-400">${{ record.amount.toFixed(2) }}</td>
                <td class="px-3 py-2 text-gray-500 dark:text-dark-400">{{ sourceLabel(record.source_type) }}</td>
                <td class="px-3 py-2 text-gray-500 dark:text-dark-400">{{ formatTime(record.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { lotteryAPI, type LotteryPrize, type LotteryStatus } from '@/api/lottery'
import { useAppStore, useAuthStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const status = ref<LotteryStatus | null>(null)
const drawing = ref(false)
const spinning = ref(false)
const lastPrize = ref<LotteryPrize | null>(null)

const records = computed(() => status.value?.recent_draws ?? [])

const wheelPrizes = computed(() => {
  const prizes = status.value?.prizes?.length ? status.value.prizes : []
  if (!prizes.length) {
    return [{
      id: 0,
      name: t('lottery.noPrizeConfigured'),
      description: '',
      amount: 0,
      probability: 1,
      daily_stock: 0,
      daily_used: 0,
      total_stock: 0,
      total_used: 0,
      enabled: true,
      color: '#94a3b8',
      sort_order: 0,
      created_at: '',
      updated_at: ''
    }] as LotteryPrize[]
  }
  return prizes
})

const wheelSegments = computed(() => {
  const prizes = wheelPrizes.value
  const step = 360 / prizes.length
  return prizes.map((prize, idx) => {
    const start = -90 + idx * step
    const end = start + step
    const mid = start + step / 2
    const label = polarToCartesian(200, 200, 124, mid)
    return {
      id: prize.id || idx,
      color: prize.color || '#f59e0b',
      path: describeArcSegment(200, 200, 190, 64, start, end),
      transform: `translate(${label.x} ${label.y}) rotate(${textRotation(mid)})`,
      title: compactWheelText(prize.name, 8),
      subtitle: compactWheelText(prizeSubtitle(prize), 10)
    }
  })
})

const sourceRows = computed(() => {
  const bySource = status.value?.chances.by_source ?? {}
  return ['daily_login', 'spend', 'recharge'].map(key => ({
    key,
    label: sourceLabel(key),
    value: bySource[key] ?? 0
  }))
})

function sourceLabel(source: string): string {
  return t(`lottery.sourceTypes.${source}`, source)
}

function formatTime(value: string): string {
  return new Date(value).toLocaleString()
}

function prizeSubtitle(prize: Pick<LotteryPrize, 'amount' | 'description'>): string {
  if (prize.amount > 0) {
    return `$${prize.amount.toFixed(2)}`
  }
  return prize.description || t('lottery.customPrize')
}

function compactWheelText(value: string, maxLength: number): string {
  const chars = Array.from((value || '').trim())
  if (chars.length <= maxLength) return chars.join('')
  return `${chars.slice(0, Math.max(1, maxLength - 1)).join('')}…`
}

function polarToCartesian(cx: number, cy: number, radius: number, angleInDegrees: number) {
  const angleInRadians = (angleInDegrees * Math.PI) / 180
  return {
    x: cx + radius * Math.cos(angleInRadians),
    y: cy + radius * Math.sin(angleInRadians)
  }
}

function describeArcSegment(cx: number, cy: number, outerRadius: number, innerRadius: number, startAngle: number, endAngle: number): string {
  const outerStart = polarToCartesian(cx, cy, outerRadius, startAngle)
  const outerEnd = polarToCartesian(cx, cy, outerRadius, endAngle)
  const innerEnd = polarToCartesian(cx, cy, innerRadius, endAngle)
  const innerStart = polarToCartesian(cx, cy, innerRadius, startAngle)
  const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1'
  return [
    `M ${outerStart.x} ${outerStart.y}`,
    `A ${outerRadius} ${outerRadius} 0 ${largeArcFlag} 1 ${outerEnd.x} ${outerEnd.y}`,
    `L ${innerEnd.x} ${innerEnd.y}`,
    `A ${innerRadius} ${innerRadius} 0 ${largeArcFlag} 0 ${innerStart.x} ${innerStart.y}`,
    'Z'
  ].join(' ')
}

function textRotation(angle: number): number {
  const normalized = ((angle % 360) + 360) % 360
  if (normalized > 90 && normalized < 270) {
    return angle + 180
  }
  return angle
}

async function loadStatus() {
  try {
    status.value = await lotteryAPI.status()
  } catch (err) {
    appStore.showError(extractI18nErrorMessage(err, t, 'lottery.errors', t('common.error')))
  }
}

async function draw() {
  if (drawing.value || spinning.value) return
  drawing.value = true
  spinning.value = true
  try {
    const result = await lotteryAPI.draw()
    setTimeout(async () => {
      lastPrize.value = result.prize
      await loadStatus()
      await authStore.refreshUser()
      appStore.showSuccess(t('lottery.drawSuccess', { name: result.prize.name, award: prizeSubtitle(result.prize), amount: result.prize.amount.toFixed(2) }))
      spinning.value = false
      drawing.value = false
    }, 1200)
  } catch (err) {
    spinning.value = false
    drawing.value = false
    appStore.showError(extractI18nErrorMessage(err, t, 'lottery.errors', t('common.error')))
  }
}

onMounted(loadStatus)
</script>

<style scoped>
.lottery-wheel {
  position: relative;
  transition: transform 0.8s cubic-bezier(0.2, 0.72, 0.28, 1);
}

.lottery-wheel.spinning {
  animation: lottery-spin 1.2s cubic-bezier(0.2, 0.72, 0.28, 1) infinite;
}

@keyframes lottery-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(720deg); }
}

.lottery-segment-title {
  fill: white;
  font-size: 17px;
  font-weight: 800;
  letter-spacing: 0;
  paint-order: stroke;
  stroke: rgba(15, 23, 42, 0.38);
  stroke-linejoin: round;
  stroke-width: 4px;
}

.lottery-segment-subtitle {
  font-size: 13px;
  font-weight: 700;
  opacity: 0.95;
}
</style>
