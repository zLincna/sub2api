<template>
  <div class="space-y-6">
    <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="mb-5 flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.configTitle') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.lottery.configDesc') }}</p>
        </div>
        <button class="btn btn-primary" :disabled="savingConfig" @click="saveConfig">
          {{ savingConfig ? t('common.saving') : t('common.save') }}
        </button>
      </div>

      <div v-if="config" class="space-y-6">
        <div class="grid gap-4 md:grid-cols-3">
          <label class="flex items-center justify-between rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.lottery.enabled') }}</span>
            <input v-model="config.enabled" type="checkbox" class="toggle" />
          </label>
          <label class="flex items-center justify-between rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.lottery.buttonEnabled') }}</span>
            <input v-model="config.button_enabled" type="checkbox" class="toggle" />
          </label>
          <div>
            <label class="input-label">{{ t('admin.lottery.timezone') }}</label>
            <input v-model="config.timezone" class="input" placeholder="Asia/Shanghai" />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.lottery.ruleText') }}</label>
          <textarea v-model="config.rule_text" rows="4" class="input"></textarea>
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
      <div class="mb-5 flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.prizesTitle') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.lottery.prizesDesc') }}</p>
        </div>
        <button class="btn btn-primary" @click="addPrize">{{ t('admin.lottery.addPrize') }}</button>
      </div>

      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.lottery.prizeName') }}</th>
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
                <button class="btn btn-secondary btn-sm mr-2" @click="editPrize(prize)">{{ t('common.edit') }}</button>
                <button class="btn btn-danger btn-sm" @click="deletePrize(prize.id)">{{ t('common.delete') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.recordsTitle') }}</h2>
        <button class="btn btn-secondary btn-sm" @click="loadRecords">{{ t('common.refresh') }}</button>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.lottery.user') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.prizeName') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.amount') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.source') }}</th>
              <th class="px-3 py-2">{{ t('admin.lottery.time') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="record in records" :key="record.id">
              <td class="px-3 py-2">{{ record.user_email || record.user_id }}</td>
              <td class="px-3 py-2">{{ record.prize_name }}</td>
              <td class="px-3 py-2 text-emerald-600 dark:text-emerald-400">${{ record.amount.toFixed(2) }}</td>
              <td class="px-3 py-2">{{ t(`lottery.sourceTypes.${record.source_type}`, record.source_type) }}</td>
              <td class="px-3 py-2 text-gray-500 dark:text-dark-400">{{ new Date(record.created_at).toLocaleString() }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <BaseDialog :show="showPrizeDialog" :title="editingPrize?.id ? t('admin.lottery.editPrize') : t('admin.lottery.addPrize')" @close="showPrizeDialog = false">
      <form id="lottery-prize-form" class="space-y-4" @submit.prevent="savePrize">
        <div>
          <label class="input-label">{{ t('admin.lottery.prizeName') }}</label>
          <input v-model="prizeForm.name" class="input" required />
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
        <button class="btn btn-secondary" @click="showPrizeDialog = false">{{ t('common.cancel') }}</button>
        <button type="submit" form="lottery-prize-form" class="btn btn-primary">{{ t('common.save') }}</button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
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
  amount: 0,
  probability: 0,
  daily_stock: 0,
  total_stock: 0,
  enabled: true,
  color: '#f59e0b',
  sort_order: 0
})

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
    return () => h('div', { class: 'rounded-md border border-gray-200 p-4 dark:border-dark-700' }, [
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
        ? h('div', [
          h('label', { class: 'input-label' }, t('admin.lottery.dailyChances')),
          h('input', {
            class: 'input',
            type: 'number',
            min: '0',
            value: (props.modelValue as LotteryLoginGrantConfig).daily_chances,
            onInput: (e: Event) => update({ daily_chances: Number((e.target as HTMLInputElement).value) })
          })
        ])
        : h('div', { class: 'space-y-2' }, [
          ...thresholds.value.map((rule, idx) => h('div', { class: 'grid grid-cols-[1fr_1fr_auto] gap-2' }, [
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
  const page = await adminLotteryAPI.listRecords({ page: 1, page_size: 20 })
  records.value = page.items || []
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
  Object.assign(prizeForm, { name: '', amount: 0, probability: 0, daily_stock: 0, total_stock: 0, enabled: true, color: '#f59e0b', sort_order: prizes.value.length * 10 })
  showPrizeDialog.value = true
}

function editPrize(prize: LotteryPrize) {
  editingPrize.value = prize
  Object.assign(prizeForm, {
    name: prize.name,
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
