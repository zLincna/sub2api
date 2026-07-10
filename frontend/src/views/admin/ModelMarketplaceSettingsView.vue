<template>
  <div class="space-y-6">
    <section class="card overflow-hidden">
      <div class="flex flex-col gap-4 border-b border-gray-100 px-6 py-5 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ localText('模型广场管理', 'Model Marketplace') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ localText('维护用户端展示的模型、价格与顺序。该配置仅用于展示，不影响网关调度和实际计费。', 'Maintain user-facing models, prices, and order. This display-only configuration does not affect gateway routing or billing.') }}
          </p>
        </div>
        <button type="button" class="btn btn-primary flex-shrink-0" @click="openCreate">
          <Icon name="plus" size="sm" />
          {{ localText('新增模型', 'Add model') }}
        </button>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-16">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>

      <div v-else-if="loadError" class="px-6 py-12 text-center">
        <Icon name="exclamationCircle" size="xl" class="mx-auto text-red-400" />
        <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ loadError }}</p>
        <button type="button" class="btn btn-secondary mt-4" @click="loadConfig">
          {{ localText('重新加载', 'Reload') }}
        </button>
      </div>

      <div v-else>
        <div class="border-b border-gray-100 bg-gray-50/70 px-6 py-3 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800/50 dark:text-gray-400">
          {{ localText('列表顺序即用户端同平台内的显示顺序。建议把最新模型放在前面。', 'List order controls display order within each platform. Keep newest models first.') }}
        </div>

        <div v-if="models.length === 0" class="px-6 py-14 text-center">
          <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
          <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">
            {{ localText('暂无模型，点击右上角新增。', 'No models yet. Add one from the top right.') }}
          </p>
        </div>

        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <div
            v-for="(model, index) in models"
            :key="`${model.platform}-${model.id}-${index}`"
            class="flex flex-col gap-4 px-5 py-4 transition-colors hover:bg-gray-50/60 dark:hover:bg-dark-800/40 lg:flex-row lg:items-center"
          >
            <div class="flex min-w-0 flex-1 items-start gap-3">
              <span class="mt-0.5 inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                <PlatformIcon :platform="model.platform as GroupPlatform" size="md" />
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="break-all font-mono text-sm font-semibold text-gray-900 dark:text-white">{{ model.id }}</span>
                  <span class="rounded-md bg-gray-100 px-2 py-0.5 text-[11px] font-medium uppercase text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ model.platform }}</span>
                  <span
                    class="rounded-md px-2 py-0.5 text-[11px] font-medium"
                    :class="model.enabled ? 'bg-green-50 text-green-700 dark:bg-green-900/25 dark:text-green-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'"
                  >
                    {{ model.enabled ? localText('已显示', 'Visible') : localText('已隐藏', 'Hidden') }}
                  </span>
                </div>
                <p class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
                  {{ model.description || localText('暂无描述', 'No description') }}
                </p>
                <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                  <span>{{ localText('渠道', 'Channel') }}：{{ model.channel_name || '-' }}</span>
                  <span>{{ localText('分组', 'Group') }}：{{ model.group_name || '-' }}</span>
                  <span>{{ localText('倍率', 'Rate') }}：{{ model.rate_multiplier }}x</span>
                  <span>{{ priceSummary(model) }}</span>
                </div>
              </div>
            </div>

            <div class="flex flex-shrink-0 items-center justify-end gap-1">
              <button type="button" class="btn-icon" :disabled="index === 0" :title="localText('上移', 'Move up')" @click="moveModel(index, -1)">
                <Icon name="arrowUp" size="sm" />
              </button>
              <button type="button" class="btn-icon" :disabled="index === models.length - 1" :title="localText('下移', 'Move down')" @click="moveModel(index, 1)">
                <Icon name="arrowDown" size="sm" />
              </button>
              <button type="button" class="btn-icon" :title="localText('编辑', 'Edit')" @click="openEdit(index)">
                <Icon name="edit" size="sm" />
              </button>
              <button type="button" class="btn-icon text-red-500 hover:text-red-600" :title="localText('删除', 'Delete')" @click="removeModel(index)">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-3 border-t border-gray-100 bg-gray-50/70 px-6 py-4 dark:border-dark-700 dark:bg-dark-800/50 sm:flex-row sm:items-center sm:justify-between">
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ localText(`共 ${models.length} 个模型，${enabledCount} 个显示中`, `${models.length} models, ${enabledCount} visible`) }}
          </p>
          <div class="flex justify-end gap-2">
            <button type="button" class="btn btn-secondary" :disabled="saving" @click="loadConfig">
              {{ localText('放弃修改', 'Discard') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="saving || !dirty" @click="saveConfig">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': saving }" />
              {{ saving ? localText('保存中...', 'Saving...') : localText('保存模型配置', 'Save models') }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <BaseDialog
      :show="showEditor"
      :title="editingIndex === null ? localText('新增模型', 'Add model') : localText('编辑模型', 'Edit model')"
      width="wide"
      @close="closeEditor"
    >
      <form class="space-y-5" @submit.prevent="applyEditor">
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ localText('模型 ID', 'Model ID') }}</label>
            <input v-model.trim="editor.id" class="input font-mono" required placeholder="gpt-5.6-sol" />
          </div>
          <div>
            <label class="input-label">{{ localText('平台', 'Platform') }}</label>
            <input v-model.trim="editor.platform" class="input" required list="model-marketplace-platforms" placeholder="openai" />
            <datalist id="model-marketplace-platforms">
              <option value="openai" />
              <option value="anthropic" />
              <option value="gemini" />
              <option value="grok" />
              <option value="antigravity" />
            </datalist>
          </div>
        </div>

        <div>
          <label class="input-label">{{ localText('模型描述', 'Description') }}</label>
          <textarea v-model.trim="editor.description" rows="2" class="input" :placeholder="localText('展示给用户的模型能力说明', 'User-facing model capability summary')" />
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ localText('渠道名称', 'Channel name') }}</label>
            <input v-model.trim="editor.channel_name" class="input" placeholder="OpenAI Pro 20x" />
          </div>
          <div>
            <label class="input-label">{{ localText('分组名称', 'Group name') }}</label>
            <input v-model.trim="editor.group_name" class="input" placeholder="OpenAI Pro 20x 号池" />
          </div>
        </div>

        <div>
          <label class="input-label">{{ localText('渠道描述', 'Channel description') }}</label>
          <input v-model.trim="editor.channel_description" class="input" :placeholder="localText('可选，鼠标悬停渠道标签时展示', 'Optional channel tooltip')" />
        </div>

        <div class="grid gap-4 sm:grid-cols-3">
          <div>
            <label class="input-label">{{ localText('计费方式', 'Billing mode') }}</label>
            <select v-model="editor.billing_mode" class="input">
              <option value="token">{{ localText('按 Token', 'Per token') }}</option>
              <option value="per_request">{{ localText('按次', 'Per request') }}</option>
              <option value="image">{{ localText('按图片', 'Per image') }}</option>
            </select>
          </div>
          <div>
            <label class="input-label">{{ localText('分组倍率', 'Rate multiplier') }}</label>
            <input v-model.number="editor.rate_multiplier" type="number" min="0" max="1000" step="0.001" class="input" />
          </div>
          <label class="flex items-end gap-3 pb-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="editor.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ localText('在用户端显示', 'Visible to users') }}
          </label>
        </div>

        <div v-if="editor.billing_mode === 'token'" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <PriceInput v-model="editor.input_price_per_million" :label="localText('输入 / 1M', 'Input / 1M')" />
          <PriceInput v-model="editor.output_price_per_million" :label="localText('输出 / 1M', 'Output / 1M')" />
          <PriceInput v-model="editor.cache_write_price_per_million" :label="localText('缓存写入 / 1M', 'Cache write / 1M')" />
          <PriceInput v-model="editor.cache_read_price_per_million" :label="localText('缓存读取 / 1M', 'Cache read / 1M')" />
        </div>
        <div v-else class="max-w-xs">
          <PriceInput
            v-if="editor.billing_mode === 'image'"
            v-model="editor.image_output_price_per_request"
            :label="localText('每张图片价格', 'Price per image')"
          />
          <PriceInput
            v-else
            v-model="editor.per_request_price"
            :label="localText('每次请求价格', 'Price per request')"
          />
        </div>

        <div class="flex justify-end gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" @click="closeEditor">{{ localText('取消', 'Cancel') }}</button>
          <button type="submit" class="btn btn-primary">{{ localText('应用', 'Apply') }}</button>
        </div>
      </form>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { adminAPI } from '@/api/admin'
import type { GroupPlatform } from '@/types'
import type { ModelMarketplaceConfigItem } from '@/api/modelMarketplace'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const PriceInput = defineComponent({
  props: {
    modelValue: { type: Number as PropType<number | null>, default: null },
    label: { type: String, required: true },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('div', [
      h('label', { class: 'input-label' }, props.label),
      h('input', {
        class: 'input',
        type: 'number',
        min: '0',
        step: '0.000001',
        value: props.modelValue ?? '',
        placeholder: '-',
        onInput: (event: Event) => {
          const value = (event.target as HTMLInputElement).value
          emit('update:modelValue', value === '' ? null : Number(value))
        },
      }),
    ])
  },
})

const { locale } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const models = ref<ModelMarketplaceConfigItem[]>([])
const originalJSON = ref('[]')
const showEditor = ref(false)
const editingIndex = ref<number | null>(null)
const editor = ref<ModelMarketplaceConfigItem>(blankModel())

const dirty = computed(() => JSON.stringify(models.value) !== originalJSON.value)
const enabledCount = computed(() => models.value.filter((model) => model.enabled).length)

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

function blankModel(): ModelMarketplaceConfigItem {
  return {
    id: '',
    platform: 'openai',
    description: '',
    channel_name: '',
    channel_description: '',
    group_name: '',
    rate_multiplier: 1,
    billing_mode: 'token',
    input_price_per_million: null,
    output_price_per_million: null,
    cache_write_price_per_million: null,
    cache_read_price_per_million: null,
    image_output_price_per_request: null,
    per_request_price: null,
    enabled: true,
  }
}

function cloneModel(model: ModelMarketplaceConfigItem): ModelMarketplaceConfigItem {
  return { ...model }
}

function openCreate(): void {
  editingIndex.value = null
  editor.value = blankModel()
  showEditor.value = true
}

function openEdit(index: number): void {
  editingIndex.value = index
  editor.value = cloneModel(models.value[index])
  showEditor.value = true
}

function closeEditor(): void {
  showEditor.value = false
  editingIndex.value = null
}

function applyEditor(): void {
  const normalized = cloneModel(editor.value)
  normalized.id = normalized.id.trim()
  normalized.platform = normalized.platform.trim().toLowerCase()
  normalized.rate_multiplier = Math.max(0, Number(normalized.rate_multiplier) || 0)
  if (!normalized.id || !normalized.platform) {
    appStore.showError(localText('模型 ID 和平台不能为空', 'Model ID and platform are required'))
    return
  }
  const duplicate = models.value.some((model, index) =>
    index !== editingIndex.value &&
    model.platform.toLowerCase() === normalized.platform &&
    model.id.toLowerCase() === normalized.id.toLowerCase(),
  )
  if (duplicate) {
    appStore.showError(localText('同一平台下模型 ID 不能重复', 'Model IDs must be unique within a platform'))
    return
  }
  if (editingIndex.value === null) models.value.unshift(normalized)
  else models.value.splice(editingIndex.value, 1, normalized)
  closeEditor()
}

function moveModel(index: number, delta: number): void {
  const target = index + delta
  if (target < 0 || target >= models.value.length) return
  const [item] = models.value.splice(index, 1)
  models.value.splice(target, 0, item)
}

function removeModel(index: number): void {
  models.value.splice(index, 1)
}

function priceSummary(model: ModelMarketplaceConfigItem): string {
  if (model.billing_mode === 'token') {
    return `${localText('输入', 'In')} $${model.input_price_per_million ?? '-'} / ${localText('输出', 'Out')} $${model.output_price_per_million ?? '-'}`
  }
  const price = model.billing_mode === 'image' ? model.image_output_price_per_request : model.per_request_price
  return `${localText('单价', 'Price')} $${price ?? '-'}`
}

async function loadConfig(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const config = await adminAPI.settings.getModelMarketplaceConfig()
    models.value = (config.models || []).map(cloneModel)
    originalJSON.value = JSON.stringify(models.value)
  } catch (error: unknown) {
    loadError.value = extractApiErrorMessage(error, localText('加载模型配置失败', 'Failed to load model configuration'))
  } finally {
    loading.value = false
  }
}

async function saveConfig(): Promise<void> {
  saving.value = true
  try {
    const config = await adminAPI.settings.updateModelMarketplaceConfig({ models: models.value })
    models.value = (config.models || []).map(cloneModel)
    originalJSON.value = JSON.stringify(models.value)
    appStore.showSuccess(localText('模型广场配置已保存', 'Model marketplace saved'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, localText('保存模型配置失败', 'Failed to save model configuration')))
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>
