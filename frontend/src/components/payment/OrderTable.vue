<template>
  <DataTable :columns="columns" :data="orders" :loading="loading">
    <template #cell-id="{ value }">
      <span class="font-mono text-sm">#{{ value }}</span>
    </template>
    <template #cell-out_trade_no="{ value }">
      <span class="text-sm text-gray-900 dark:text-white">{{ value }}</span>
    </template>
    <template v-if="showUser" #cell-user_email="{ value, row }">
      <div class="text-sm">
        <span class="text-gray-900 dark:text-white">{{ value || row.user_name || '#' + row.user_id }}</span>
        <span v-if="row.user_notes" class="ml-1 text-xs text-gray-400">({{ row.user_notes }})</span>
      </div>
    </template>
    <template #cell-pay_amount="{ value, row }">
      <div class="text-sm">
        <span class="font-medium text-gray-900 dark:text-white">{{ paymentAmountSymbol(row) }}{{ value.toFixed(2) }}</span>
        <span v-if="row.fee_rate > 0" class="ml-1 text-xs text-gray-400" :title="t('payment.orders.fee') + ': ' + row.fee_rate + '%'">
          ({{ t('payment.orders.fee') }} {{ row.fee_rate }}%)
        </span>
        <div v-if="row.amount !== row.pay_amount" class="text-xs text-gray-500">
          {{ t('payment.orders.creditedAmount') }}: {{ creditedAmountSymbol }}{{ row.amount.toFixed(2) }}
        </div>
      </div>
    </template>
    <template #cell-payment_type="{ value }">
      <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('payment.methods.' + value, value) }}</span>
    </template>
    <template #cell-order_type="{ value, row }">
      <div class="text-sm">
        <span class="font-medium text-gray-900 dark:text-white">{{ orderTypeLabel(value) }}</span>
        <div v-if="carpoolSummary(row)" class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
          {{ carpoolSummary(row) }}
        </div>
      </div>
    </template>
    <template #cell-status="{ value }">
      <OrderStatusBadge :status="value" />
    </template>
    <template #cell-created_at="{ value }">
      <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatDate(value) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <slot name="actions" :row="row" />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaymentOrder } from '@/types/payment'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import { currencySymbol } from '@/components/payment/currency'

const { t } = useI18n()

const props = defineProps<{
  orders: PaymentOrder[]
  loading: boolean
  showUser?: boolean
}>()

function formatDate(dateStr: string) { return new Date(dateStr).toLocaleString() }

const creditedAmountSymbol = currencySymbol('USD')

function paymentAmountSymbol(order: PaymentOrder): string {
  return currencySymbol(order.currency)
}

const columns = computed((): Column[] => {
  const cols: Column[] = [
    { key: 'id', label: t('payment.orders.orderId') },
    { key: 'out_trade_no', label: t('payment.orders.orderNo') },
  ]
  if (props.showUser) {
    cols.push({ key: 'user_email', label: t('payment.admin.colUser') })
  }
  cols.push(
    { key: 'pay_amount', label: t('payment.orders.payAmount') },
    { key: 'order_type', label: '订单类型' },
    { key: 'payment_type', label: t('payment.orders.paymentMethod') },
    { key: 'status', label: t('payment.orders.status') },
    { key: 'created_at', label: t('payment.orders.createdAt') },
    { key: 'actions', label: t('common.actions') },
  )
  return cols
})

function orderTypeLabel(value: string) {
  const labels: Record<string, string> = {
    balance: '余额充值',
    subscription: '订阅套餐',
    carpool: '拼车订单',
  }
  return labels[value] || value
}

function carpoolSummary(row: PaymentOrder) {
  const item = row.edges?.carpool_participants?.[0]
  const vt = item?.edges?.vehicle_type
  if (!item || !vt) return ''
  return `${segmentLabel(vt)} · ${vt.name} · ${carpoolStatusLabel(item.status)}`
}

function segmentLabel(vt: { product?: string; plan_tier?: string; multiplier?: string }) {
  const productLabels: Record<string, string> = { openai: 'OpenAI', claudecode: 'ClaudeCode', claude_code: 'ClaudeCode' }
  const tierLabels: Record<string, string> = { pro: 'Pro', plus: 'Plus' }
  const product = productLabels[String(vt.product || '').toLowerCase()] || vt.product || 'OpenAI'
  const tier = tierLabels[String(vt.plan_tier || '').toLowerCase()] || vt.plan_tier || 'Pro'
  const multiplier = String(vt.multiplier || '').toUpperCase()
  return [product, tier, multiplier].filter(Boolean).join(' ')
}

function carpoolStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending_payment: '待支付',
    paid: '已支付',
    active: '已发车',
    refund_pending: '待退款',
    refunded_balance: '已退余额',
    refunded_gateway: '已原路退款',
    cancelled: '已取消',
  }
  return labels[status] || status
}
</script>
