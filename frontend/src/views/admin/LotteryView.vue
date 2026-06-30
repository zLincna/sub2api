<template>
  <div class="space-y-6">
    <section class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="flex flex-col gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700 lg:flex-row lg:items-start lg:justify-between">
        <div class="flex min-w-0 items-start gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300">
            <Icon name="gift" size="md" />
          </div>
          <div class="min-w-0">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.configTitle') }}</h2>
            <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-dark-400">{{ t('admin.lottery.configDesc') }}</p>
          </div>
        </div>
        <button type="button" class="btn btn-primary shrink-0" :disabled="savingConfig" @click="saveConfig">
          {{ savingConfig ? t('common.saving') : t('common.save') }}
        </button>
      </div>

      <div v-if="config" class="space-y-6 p-5">
        <div class="grid gap-3 lg:grid-cols-[1.1fr_1.1fr_0.8fr]">
          <label class="flex min-h-20 items-center justify-between rounded-md border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800/70">
            <span>
              <span class="block text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.enabled') }}</span>
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('admin.lottery.enabledHint') }}</span>
            </span>
            <input v-model="config.enabled" type="checkbox" class="toggle" />
          </label>
          <label class="flex min-h-20 items-center justify-between rounded-md border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800/70">
            <span>
              <span class="block text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.buttonEnabled') }}</span>
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('admin.lottery.buttonEnabledHint') }}</span>
            </span>
            <input v-model="config.button_enabled" type="checkbox" class="toggle" />
          </label>
          <div class="min-h-20 rounded-md border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800/70">
            <label class="input-label">{{ t('admin.lottery.timezone') }}</label>
            <input v-model="config.timezone" class="input" placeholder="Asia/Shanghai" />
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-3">
          <div class="rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.lottery.enabledPrizes') }}</div>
            <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ enabledPrizeCount }}/{{ prizes.length }}</div>
          </div>
          <div class="rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.lottery.totalWeight') }}</div>
            <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ totalProbabilityWeight }}</div>
          </div>
          <div class="rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.lottery.latestRecords') }}</div>
            <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ records.length }}</div>
          </div>
        </div>

        <div class="rounded-md border border-gray-200 p-4 dark:border-dark-700">
          <label class="input-label">{{ t('admin.lottery.ruleText') }}</label>
          <textarea v-model="config.rule_text" rows="3" class="input resize-y"></textarea>
        </div>

        <div class="grid gap-4 xl:grid-cols-3">
          <RuleCard
            v-model="config.login_grant"
            type="login"
            :title="t('admin.lottery.loginRule')"
            :allow-thresholds="false"
          />
          <RuleCard
            v-model="config.spend_grant"
            type="threshold"
            :title="t('admin.lottery.spendRule')"
            :allow-thresholds="true"
          />
          <RuleCard
            v-model="config.recharge_grant"
            type="threshold"
            :title="t('admin.lottery.rechargeRule')"
            :allow-thresholds="true"
          />
        </div>
      </div>
    </section>

    <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.prizesTitle') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.lottery.prizesDesc') }}
            <span class="ml-2 inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-800 dark:text-dark-300">
              {{ t('admin.lottery.totalWeight') }}: {{ totalProbabilityWeight }}
            </span>
          </p>
        </div>
        <button type="button" class="btn btn-primary shrink-0" @click="addPrize">{{ t('admin.lottery.addPrize') }}</button>
      </div>

      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.lottery.prizeName') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.prizeContent') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.amount') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.probability') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.stock') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.enabled') }}</th>
              <th class="px-3 py-2">{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="prize in prizes" :key="prize.id">
              <td class="px-3 py-2">
                <div class="flex items-center gap-2">
                  <span class="h-3 w-3 rounded-full" :style="{ backgroundColor: prize.color }"></span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ prize.name }}</span>
                </div>
              </td>
              <td class="max-w-xs px-3 py-2 text-gray-500 dark:text-dark-400">
                <span class="line-clamp-2">{{ prize.description || '-' }}</span>
              </td>
              <td class="px-3 py-2">${{ prize.amount.toFixed(2) }}</td>
              <td class="px-3 py-2">{{ prize.probability }}%</td>
              <td class="px-3 py-2 text-gray-500 dark:text-dark-400">
                {{ prize.daily_used }}/{{ prize.daily_stock || t('admin.lottery.unlimited') }} ·
                {{ prize.total_used }}/{{ prize.total_stock || t('admin.lottery.unlimited') }}
              </td>
              <td class="px-3 py-2">
                <span class="badge" :class="prize.enabled ? 'badge-success' : 'badge-gray'">{{ prize.enabled ? t('common.enabled') : t('common.disabled') }}</span>
              </td>
              <td class="px-3 py-2">
                <div class="flex gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="editPrize(prize)">{{ t('common.edit') }}</button>
                  <button type="button" class="btn btn-danger btn-sm" @click="deletePrize(prize.id)">{{ t('common.delete') }}</button>
                </div>
              </td>
            </tr>
            <tr v-if="prizes.length === 0">
              <td colspan="7" class="px-3 py-10 text-center text-sm text-gray-500 dark:text-dark-400">
                {{ t('admin.lottery.emptyPrizes') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.recordsTitle') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.lottery.recordsDesc') }}</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="loadRecords">{{ t('common.refresh') }}</button>
      </div>
      <div class="mb-4 grid gap-3 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60 lg:grid-cols-[1fr_1fr_1fr_1fr_auto]">
        <div>
          <label class="input-label">{{ t('admin.lottery.userId') }}</label>
          <input v-model.number="recordFilters.user_id" type="number" min="1" class="input" :placeholder="t('admin.lottery.userId')" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.lottery.userKeyword') }}</label>
          <input v-model="recordFilters.user_query" class="input" :placeholder="t('admin.lottery.userKeywordPlaceholder')" @keyup.enter="applyRecordFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.lottery.source') }}</label>
          <select v-model="recordFilters.source_type" class="input">
            <option value="">{{ t('common.all') }}</option>
            <option value="daily_login">{{ t('lottery.sourceTypes.daily_login') }}</option>
            <option value="spend">{{ t('lottery.sourceTypes.spend') }}</option>
            <option value="recharge">{{ t('lottery.sourceTypes.recharge') }}</option>
          </select>
        </div>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.lottery.startTime') }}</label>
            <input v-model="recordFilters.start_time" type="datetime-local" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.lottery.endTime') }}</label>
            <input v-model="recordFilters.end_time" type="datetime-local" class="input" />
          </div>
        </div>
        <div class="flex items-end gap-2">
          <button type="button" class="btn btn-primary" @click="applyRecordFilters">{{ t('common.search') }}</button>
          <button type="button" class="btn btn-secondary" @click="resetRecordFilters">{{ t('common.reset') }}</button>
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.lottery.user') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.prizeName') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.prizeContent') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.amount') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.source') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.time') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="record in records" :key="record.id">
              <td class="px-3 py-2">{{ record.user_email || record.user_id }}</td>
              <td class="px-3 py-2">{{ record.prize_name }}</td>
              <td class="max-w-sm px-3 py-2 text-gray-500 dark:text-dark-400">
                <span class="line-clamp-2">{{ record.prize_description || '-' }}</span>
              </td>
              <td class="px-3 py-2 text-emerald-600 dark:text-emerald-400">${{ record.amount.toFixed(2) }}</td>
              <td class="px-3 py-2">{{ t(`lottery.sourceTypes.${record.source_type}`, record.source_type) }}</td>
              <td class="px-3 py-2 text-gray-500 dark:text-dark-400">{{ new Date(record.created_at).toLocaleString() }}</td>
            </tr>
            <tr v-if="records.length === 0">
              <td colspan="6" class="px-3 py-10 text-center text-sm text-gray-500 dark:text-dark-400">
                {{ t('admin.lottery.emptyRecords') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 flex flex-col gap-3 border-t border-gray-100 pt-4 text-sm dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
        <div class="text-gray-500 dark:text-dark-400">
          {{ t('admin.lottery.recordTotal', { total: recordPagination.total, page: recordPagination.page, pages: recordPagination.pages }) }}
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="recordPagination.page <= 1" @click="goRecordPage(recordPagination.page - 1)">{{ t('admin.lottery.previousPage') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="recordPagination.page >= recordPagination.pages" @click="goRecordPage(recordPagination.page + 1)">{{ t('admin.lottery.nextPage') }}</button>
        </div>
      </div>
    </section>

    <BaseDialog :show="showPrizeDialog" :title="editingPrize?.id ? t('admin.lottery.editPrize') : t('admin.lottery.addPrize')" @close="showPrizeDialog = false">
      <form id="lottery-prize-form" class="space-y-4" @submit.prevent="savePrize">
        <div>
          <label class="input-label">{{ t('admin.lottery.prizeName') }}</label>
          <input v-model="prizeForm.name" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('admin.lottery.prizeContent') }}</label>
          <textarea v-model="prizeForm.description" rows="3" class="input resize-y" :placeholder="t('admin.lottery.prizeContentPlaceholder')"></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.lottery.prizeContentHint') }}</p>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div><label class="input-label">{{ t('admin.lottery.amount') }}</label><input v-model.number="prizeForm.amount" type="number" min="0" step="0.01" class="input" /></div>
          <div><label class="input-label">{{ t('admin.lottery.probability') }}</label><input v-model.number="prizeForm.probability" type="number" min="0" step="0.000001" class="input" /></div>
          <div><label class="input-label">{{ t('admin.lottery.dailyStock') }}</label><input v-model.number="prizeForm.daily_stock" type="number" min="0" class="input" /></div>
          <div><label class="input-label">{{ t('admin.lottery.totalStock') }}</label><input v-model.number="prizeForm.total_stock" type="number" min="0" class="input" /></div>
          <div><label class="input-label">{{ t('admin.lottery.color') }}</label><input v-model="prizeForm.color" type="color" class="h-10 w-full rounded border border-gray-300 bg-white p-1 dark:border-dark-600 dark:bg-dark-800" /></div>
          <div><label class="input-label">{{ t('admin.lottery.sortOrder') }}</label><input v-model.number="prizeForm.sort_order" type="number" class="input" /></div>
        </div>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
          <input v-model="prizeForm.enabled" type="checkbox" class="toggle" />
          {{ t('admin.lottery.enabled') }}
        </label>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showPrizeDialog = false">{{ t('common.cancel') }}</button>
        <button type="submit" form="lottery-prize-form" class="btn btn-primary">{{ t('common.save') }}</button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminLotteryAPI, type LotteryPrizeInput } from '@/api/admin/lottery'
import type { LotteryConfig, LotteryDrawRecord, LotteryLoginGrantConfig, LotteryPrize, LotteryThresholdGrantConfig } from '@/api/lottery'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const config = ref<LotteryConfig | null>(null)
const prizes = ref<LotteryPrize[]>([])
const records = ref<LotteryDrawRecord[]>([])
const savingConfig = ref(false)
const showPrizeDialog = ref(false)
const editingPrize = ref<LotteryPrize | null>(null)
const prizeForm = reactive<LotteryPrizeInput>({
  name: '',
  description: '',
  amount: 0,
  probability: 0,
  daily_stock: 0,
  total_stock: 0,
  enabled: true,
  color: '#f59e0b',
  sort_order: 0
})
const recordFilters = reactive({
  user_id: undefined as number | undefined,
  user_query: '',
  source_type: '',
  start_time: '',
  end_time: ''
})
const recordPagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 1
})

const enabledPrizeCount = computed(() => prizes.value.filter(prize => prize.enabled).length)
const totalProbabilityWeight = computed(() => Number(prizes.value
  .filter(prize => prize.enabled)
  .reduce((sum, prize) => sum + Number(prize.probability || 0), 0)
  .toFixed(6)))

const RuleCard = defineComponent({
  props: {
    modelValue: { type: Object as PropType<LotteryLoginGrantConfig | LotteryThresholdGrantConfig>, required: true },
    title: { type: String, required: true },
    type: { type: String as PropType<'login' | 'threshold'>, required: true },
    allowThresholds: { type: Boolean, default: false }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const update = (patch: Record<string, unknown>) => emit('update:modelValue', { ...props.modelValue, ...patch })
    const thresholds = computed(() => (props.modelValue as LotteryThresholdGrantConfig).thresholds ?? [])
    return () => h('div', { class: 'flex min-h-full flex-col rounded-md border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-800/50' }, [
      h('div', { class: 'mb-4 flex items-center justify-between' }, [
        h('h3', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, props.title),
        h('input', {
          type: 'checkbox',
          class: 'toggle',
          checked: props.modelValue.enabled,
          onChange: (e: Event) => update({ enabled: (e.target as HTMLInputElement).checked })
        })
      ]),
      props.type === 'login'
        ? h('div', { class: 'rounded-md border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900' }, [
          h('label', { class: 'input-label' }, t('admin.lottery.dailyChances')),
          h('input', {
            class: 'input',
            type: 'number',
            min: '0',
            value: (props.modelValue as LotteryLoginGrantConfig).daily_chances,
            onInput: (e: Event) => update({ daily_chances: Number((e.target as HTMLInputElement).value) })
          })
        ])
        : h('div', { class: 'space-y-2 rounded-md border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900' }, [
          h('div', { class: 'grid grid-cols-[1fr_1fr_auto] gap-2 text-xs font-medium text-gray-500 dark:text-dark-400' }, [
            h('span', t('admin.lottery.thresholdAmount')),
            h('span', t('admin.lottery.thresholdChances')),
            h('span', { class: 'w-14 text-right' }, t('common.actions'))
          ]),
          thresholds.value.length === 0
            ? h('div', { class: 'rounded-md border border-dashed border-gray-200 py-5 text-center text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400' }, t('admin.lottery.emptyTiers'))
            : thresholds.value.map((rule, idx) => h('div', { class: 'grid grid-cols-[1fr_1fr_auto] gap-2' }, [
              h('input', { class: 'input', type: 'number', step: '0.01', min: '0', value: rule.amount, onInput: (e: Event) => {
              const next = [...thresholds.value]
              next[idx] = { ...next[idx], amount: Number((e.target as HTMLInputElement).value) }
              update({ thresholds: next })
              } }),
              h('input', { class: 'input', type: 'number', min: '0', value: rule.chances, onInput: (e: Event) => {
              const next = [...thresholds.value]
              next[idx] = { ...next[idx], chances: Number((e.target as HTMLInputElement).value) }
              update({ thresholds: next })
              } }),
              h('button', { class: 'btn btn-danger btn-sm', type: 'button', onClick: () => update({ thresholds: thresholds.value.filter((_, i) => i !== idx) }) }, t('common.delete'))
            ])),
          h('button', { class: 'btn btn-secondary btn-sm', type: 'button', onClick: () => update({ thresholds: [...thresholds.value, { amount: 0, chances: 1 }] }) }, t('admin.lottery.addTier'))
        ]),
      h('div', { class: 'mt-4 grid grid-cols-2 gap-2' }, [
        h('select', { class: 'input', value: props.modelValue.expiry_mode, onChange: (e: Event) => update({ expiry_mode: (e.target as HTMLSelectElement).value }) }, [
          h('option', { value: 'end_of_day' }, t('admin.lottery.expireEndOfDay')),
          h('option', { value: 'hours' }, t('admin.lottery.expireHours'))
        ]),
        h('input', { class: 'input', type: 'number', min: '1', value: props.modelValue.expiry_hours, onInput: (e: Event) => update({ expiry_hours: Number((e.target as HTMLInputElement).value) }) })
      ])
    ])
  }
})

async function loadAll() {
  await Promise.all([loadConfig(), loadPrizes(), loadRecords()])
}

async function loadConfig() {
  config.value = await adminLotteryAPI.getConfig()
}

async function loadPrizes() {
  prizes.value = await adminLotteryAPI.listPrizes()
}

async function loadRecords() {
  const page = await adminLotteryAPI.listRecords(buildRecordQuery())
  records.value = page.items || []
  recordPagination.total = page.total
  recordPagination.page = page.page
  recordPagination.page_size = page.page_size
  recordPagination.pages = page.pages
}

async function saveConfig() {
  if (!config.value) return
  savingConfig.value = true
  try {
    config.value = await adminLotteryAPI.updateConfig(config.value)
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.lottery.errors', t('common.error')))
  } finally {
    savingConfig.value = false
  }
}

function addPrize() {
  editingPrize.value = null
  Object.assign(prizeForm, { name: '', description: '', amount: 0, probability: 0, daily_stock: 0, total_stock: 0, enabled: true, color: '#f59e0b', sort_order: prizes.value.length * 10 })
  showPrizeDialog.value = true
}

function editPrize(prize: LotteryPrize) {
  editingPrize.value = prize
  Object.assign(prizeForm, {
    name: prize.name,
    description: prize.description || '',
    amount: prize.amount,
    probability: prize.probability,
    daily_stock: prize.daily_stock,
    total_stock: prize.total_stock,
    enabled: prize.enabled,
    color: prize.color,
    sort_order: prize.sort_order
  })
  showPrizeDialog.value = true
}

function buildRecordQuery() {
  return {
    page: recordPagination.page,
    page_size: recordPagination.page_size,
    user_id: recordFilters.user_id || undefined,
    user_query: recordFilters.user_query.trim() || undefined,
    source_type: recordFilters.source_type || undefined,
    start_time: toApiDateTime(recordFilters.start_time),
    end_time: toApiDateTime(recordFilters.end_time)
  }
}

function toApiDateTime(value: string): string | undefined {
  if (!value) return undefined
  return new Date(value).toISOString()
}

function applyRecordFilters() {
  recordPagination.page = 1
  loadRecords().catch(err => appStore.showError(extractI18nErrorMessage(err, t, 'admin.lottery.errors', t('common.error'))))
}

function resetRecordFilters() {
  Object.assign(recordFilters, {
    user_id: undefined,
    user_query: '',
    source_type: '',
    start_time: '',
    end_time: ''
  })
  applyRecordFilters()
}

function goRecordPage(page: number) {
  recordPagination.page = Math.min(Math.max(1, page), recordPagination.pages || 1)
  loadRecords().catch(err => appStore.showError(extractI18nErrorMessage(err, t, 'admin.lottery.errors', t('common.error'))))
}

async function savePrize() {
  try {
    if (editingPrize.value) {
      await adminLotteryAPI.updatePrize(editingPrize.value.id, prizeForm)
    } else {
      await adminLotteryAPI.createPrize(prizeForm)
    }
    showPrizeDialog.value = false
    await loadPrizes()
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.lottery.errors', t('common.error')))
  }
}

async function deletePrize(id: number) {
  if (!window.confirm(t('admin.lottery.deletePrizeConfirm'))) return
  try {
    await adminLotteryAPI.deletePrize(id)
    await loadPrizes()
    appStore.showSuccess(t('common.deleted'))
  } catch (err) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.lottery.errors', t('common.error')))
  }
}

onMounted(() => {
  loadAll().catch(err => appStore.showError(extractI18nErrorMessage(err, t, 'admin.lottery.errors', t('common.error'))))
})
</script>
