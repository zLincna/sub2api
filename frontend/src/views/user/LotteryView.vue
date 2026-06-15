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
            <div class="relative h-72 w-72 sm:h-80 sm:w-80">
              <div class="absolute left-1/2 top-0 z-10 -translate-x-1/2">
                <div class="h-0 w-0 border-x-[12px] border-t-[28px] border-x-transparent border-t-red-500 drop-shadow"></div>
              </div>
              <div
                class="lottery-wheel h-full w-full rounded-full border-[10px] border-white shadow-xl dark:border-dark-800"
                :class="{ spinning }"
                :style="wheelStyle"
              >
                <div class="absolute inset-8 rounded-full bg-white/80 shadow-inner dark:bg-dark-900/80"></div>
                <div class="absolute inset-0 flex items-center justify-center">
                  <button
                    type="button"
                    class="z-10 flex h-24 w-24 flex-col items-center justify-center rounded-full bg-red-500 text-white shadow-lg transition hover:bg-red-600 disabled:cursor-not-allowed disabled:bg-gray-400"
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
              <p class="text-xl font-bold text-emerald-800 dark:text-emerald-200">{{ lastPrize.name }} · ${{ lastPrize.amount.toFixed(2) }}</p>
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
                <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ record.prize_name }}</td>
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

const wheelStyle = computed(() => {
  const prizes = status.value?.prizes?.length ? status.value.prizes : []
  if (!prizes.length) {
    return { background: 'conic-gradient(#e5e7eb 0deg 360deg)' }
  }
  const step = 360 / prizes.length
  const stops = prizes.map((p, idx) => `${p.color || '#f59e0b'} ${idx * step}deg ${(idx + 1) * step}deg`)
  return { background: `conic-gradient(${stops.join(', ')})` }
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
      appStore.showSuccess(t('lottery.drawSuccess', { name: result.prize.name, amount: result.prize.amount.toFixed(2) }))
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
  transition: transform 0.8s ease;
}

.lottery-wheel.spinning {
  animation: lottery-spin 1.2s cubic-bezier(0.2, 0.72, 0.28, 1) infinite;
}

@keyframes lottery-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(720deg); }
}
</style>
