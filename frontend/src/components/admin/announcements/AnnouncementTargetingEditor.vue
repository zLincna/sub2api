<template>
  <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/50">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.announcements.form.targetingMode') }}
        </div>
        <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          {{ mode === 'all' ? t('admin.announcements.form.targetingAll') : t('admin.announcements.form.targetingCustom') }}
        </div>
      </div>

      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            type="radio"
            name="announcement-targeting-mode"
            value="all"
            :checked="mode === 'all'"
            @change="setMode('all')"
            class="h-4 w-4"
          />
          {{ t('admin.announcements.form.targetingAll') }}
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            type="radio"
            name="announcement-targeting-mode"
            value="custom"
            :checked="mode === 'custom'"
            @change="setMode('custom')"
            class="h-4 w-4"
          />
          {{ t('admin.announcements.form.targetingCustom') }}
        </label>
      </div>
    </div>

    <div v-if="mode === 'custom'" class="mt-4 space-y-4">
      <div class="flex items-center justify-between">
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          OR
          <span class="ml-1 text-xs font-normal text-gray-500 dark:text-dark-400">
            ({{ anyOf.length }}/50)
          </span>
        </div>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="anyOf.length >= 50"
          @click="addOrGroup"
        >
          <Icon name="plus" size="sm" class="mr-1" />
          {{ t('admin.announcements.form.addOrGroup') }}
        </button>
      </div>

      <div v-if="anyOf.length === 0" class="rounded-xl border border-dashed border-gray-300 p-4 text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
        {{ t('admin.announcements.form.targetingCustom') }}: {{ t('admin.announcements.form.addOrGroup') }}
      </div>

      <div
        v-for="(group, groupIndex) in anyOf"
        :key="groupIndex"
        class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.announcements.form.targetingCustom') }} #{{ groupIndex + 1 }}
              <span class="ml-2 text-xs font-normal text-gray-500 dark:text-dark-400">AND ({{ (group.all_of?.length || 0) }}/50)</span>
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.announcements.form.addAndCondition') }}
            </div>
          </div>

          <button
            type="button"
            class="btn btn-secondary"
            @click="removeOrGroup(groupIndex)"
          >
            <Icon name="trash" size="sm" class="mr-1" />
            {{ t('common.delete') }}
          </button>
        </div>

        <div class="mt-4 space-y-3">
          <div
            v-for="(cond, condIndex) in (group.all_of || [])"
            :key="condIndex"
            class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/30"
          >
            <div class="flex flex-col gap-3 md:flex-row md:items-end">
              <div class="w-full md:w-52">
                <label class="input-label">{{ t('admin.announcements.form.conditionType') }}</label>
                <Select
                  :model-value="cond.type"
                  :options="conditionTypeOptions"
                  @update:model-value="(v) => setConditionType(groupIndex, condIndex, v as any)"
                />
              </div>

              <div v-if="cond.type === 'subscription'" class="flex-1">
                <label class="input-label">{{ t('admin.announcements.form.selectPackages') }}</label>
                <GroupSelector
                  v-model="groupSelections[groupIndex][condIndex]"
                  :groups="subscriptionGroups"
                />
              </div>

              <div v-else-if="cond.type === 'group'" class="flex-1">
                <label class="input-label">{{ t('admin.announcements.form.selectAPIKeyGroups') }}</label>
                <GroupSelector
                  v-model="groupSelections[groupIndex][condIndex]"
                  :groups="groups"
                />
                <p class="input-hint">{{ t('admin.announcements.form.apiKeyGroupsHint') }}</p>
              </div>

              <div v-else-if="cond.type === 'user'" class="flex-1">
                <label class="input-label">{{ t('admin.announcements.form.selectUsers') }}</label>
                <div class="space-y-2">
                  <div class="relative">
                    <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                    <input
                      v-model="userSearch"
                      type="text"
                      class="input pl-9"
                      :placeholder="t('admin.announcements.searchUsers')"
                    />
                  </div>
                  <div class="flex gap-2">
                    <input
                      v-model="manualUserID"
                      type="number"
                      min="1"
                      class="input"
                      :placeholder="t('admin.announcements.form.addUserIDPlaceholder')"
                    />
                    <button
                      type="button"
                      class="btn btn-secondary shrink-0"
                      @click="addManualUserID(groupIndex, condIndex)"
                    >
                      <Icon name="plus" size="sm" class="mr-1" />
                      {{ t('admin.announcements.form.addUserID') }}
                    </button>
                  </div>
                  <div class="max-h-32 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-800">
                    <label
                      v-for="user in filteredUsers"
                      :key="user.id"
                      class="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm transition-colors hover:bg-white dark:hover:bg-dark-700"
                    >
                      <input
                        type="checkbox"
                        :checked="(userSelections[groupIndex]?.[condIndex] ?? []).includes(user.id)"
                        @change="setUserSelected(groupIndex, condIndex, user.id, ($event.target as HTMLInputElement).checked)"
                        class="h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
                      />
                      <span class="min-w-0 flex-1 truncate text-gray-900 dark:text-white">
                        {{ user.email || user.username || `#${user.id}` }}
                      </span>
                      <span class="shrink-0 text-xs text-gray-500 dark:text-dark-400">#{{ user.id }}</span>
                    </label>
                    <div v-if="filteredUsers.length === 0" class="py-2 text-center text-sm text-gray-500 dark:text-dark-400">
                      {{ t('empty.noData') }}
                    </div>
                  </div>
                  <p class="input-hint">
                    {{ t('common.selectedCount', { count: (userSelections[groupIndex]?.[condIndex] ?? []).length }) }}
                  </p>
                  <div
                    v-if="(userSelections[groupIndex]?.[condIndex] ?? []).length > 0"
                    class="flex flex-wrap gap-2"
                  >
                    <button
                      v-for="userID in userSelections[groupIndex][condIndex]"
                      :key="userID"
                      type="button"
                      class="inline-flex items-center gap-1 rounded-full border border-gray-200 bg-white px-2 py-1 text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200"
                      @click="setUserSelected(groupIndex, condIndex, userID, false)"
                    >
                      #{{ userID }}
                      <Icon name="x" size="xs" />
                    </button>
                  </div>
                </div>
              </div>

              <div v-else class="flex flex-1 flex-col gap-3 sm:flex-row">
                <div class="w-full sm:w-44">
                  <label class="input-label">{{ t('admin.announcements.form.operator') }}</label>
                  <Select
                    :model-value="cond.operator"
                    :options="balanceOperatorOptions"
                    @update:model-value="(v) => setOperator(groupIndex, condIndex, v as any)"
                  />
                </div>
                <div class="w-full sm:flex-1">
                  <label class="input-label">{{ t('admin.announcements.form.balanceValue') }}</label>
                  <input
                    :value="String(cond.value ?? '')"
                    type="number"
                    step="any"
                    class="input"
                    @input="(e) => setBalanceValue(groupIndex, condIndex, (e.target as HTMLInputElement).value)"
                  />
                </div>
              </div>

              <div class="flex justify-end">
                <button
                  type="button"
                  class="btn btn-secondary"
                  @click="removeAndCondition(groupIndex, condIndex)"
                >
                  <Icon name="trash" size="sm" class="mr-1" />
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>
          </div>

          <div class="flex justify-end">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="(group.all_of?.length || 0) >= 50"
              @click="addAndCondition(groupIndex)"
            >
              <Icon name="plus" size="sm" class="mr-1" />
              {{ t('admin.announcements.form.addAndCondition') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="validationError" class="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/30 dark:bg-red-900/10 dark:text-red-300">
        {{ validationError }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  AdminGroup,
  AdminUser,
  AnnouncementTargeting,
  AnnouncementCondition,
  AnnouncementConditionGroup,
  AnnouncementConditionType,
  AnnouncementOperator
} from '@/types'

import Select from '@/components/common/Select.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const props = defineProps<{
  modelValue: AnnouncementTargeting
  groups: AdminGroup[]
  users: AdminUser[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: AnnouncementTargeting): void
}>()

const anyOf = computed(() => props.modelValue?.any_of ?? [])

type Mode = 'all' | 'custom'
const mode = computed<Mode>(() => (anyOf.value.length === 0 ? 'all' : 'custom'))

const conditionTypeOptions = computed(() => [
  { value: 'subscription', label: t('admin.announcements.form.conditionSubscription') },
  { value: 'balance', label: t('admin.announcements.form.conditionBalance') },
  { value: 'group', label: t('admin.announcements.form.conditionAPIKeyGroup') },
  { value: 'user', label: t('admin.announcements.form.conditionUser') }
])

const subscriptionGroups = computed(() => props.groups.filter((g) => g.subscription_type === 'subscription'))

const balanceOperatorOptions = computed(() => [
  { value: 'gt', label: t('admin.announcements.operators.gt') },
  { value: 'gte', label: t('admin.announcements.operators.gte') },
  { value: 'lt', label: t('admin.announcements.operators.lt') },
  { value: 'lte', label: t('admin.announcements.operators.lte') },
  { value: 'eq', label: t('admin.announcements.operators.eq') }
])

function setMode(next: Mode) {
  if (next === 'all') {
    emit('update:modelValue', { any_of: [] })
    return
  }
  if (anyOf.value.length === 0) {
    emit('update:modelValue', { any_of: [{ all_of: [defaultSubscriptionCondition()] }] })
  }
}

function defaultSubscriptionCondition(): AnnouncementCondition {
  return {
    type: 'subscription' as AnnouncementConditionType,
    operator: 'in' as AnnouncementOperator,
    group_ids: []
  }
}

function defaultBalanceCondition(): AnnouncementCondition {
  return {
    type: 'balance' as AnnouncementConditionType,
    operator: 'gte' as AnnouncementOperator,
    value: 0
  }
}

function defaultGroupCondition(): AnnouncementCondition {
  return {
    type: 'group' as AnnouncementConditionType,
    operator: 'in' as AnnouncementOperator,
    group_ids: []
  }
}

function defaultUserCondition(): AnnouncementCondition {
  return {
    type: 'user' as AnnouncementConditionType,
    operator: 'in' as AnnouncementOperator,
    user_ids: []
  }
}

type TargetingDraft = {
  any_of: AnnouncementConditionGroup[]
}

function updateTargeting(mutator: (draft: TargetingDraft) => void) {
  const draft: TargetingDraft = JSON.parse(JSON.stringify(props.modelValue ?? { any_of: [] }))
  if (!draft.any_of) draft.any_of = []
  mutator(draft)
  emit('update:modelValue', draft)
}

function addOrGroup() {
  updateTargeting((draft) => {
    if (draft.any_of.length >= 50) return
    draft.any_of.push({ all_of: [defaultSubscriptionCondition()] })
  })
}

function removeOrGroup(groupIndex: number) {
  updateTargeting((draft) => {
    draft.any_of.splice(groupIndex, 1)
  })
}

function addAndCondition(groupIndex: number) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group.all_of) group.all_of = []
    if (group.all_of.length >= 50) return
    group.all_of.push(defaultSubscriptionCondition())
  })
}

function removeAndCondition(groupIndex: number, condIndex: number) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return
    group.all_of.splice(condIndex, 1)
  })
}

function setConditionType(groupIndex: number, condIndex: number, nextType: AnnouncementConditionType) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return

    if (nextType === 'subscription') {
      group.all_of[condIndex] = defaultSubscriptionCondition()
    } else if (nextType === 'group') {
      group.all_of[condIndex] = defaultGroupCondition()
    } else if (nextType === 'user') {
      group.all_of[condIndex] = defaultUserCondition()
    } else {
      group.all_of[condIndex] = defaultBalanceCondition()
    }
  })
}

function setOperator(groupIndex: number, condIndex: number, op: AnnouncementOperator) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return

    const cond = group.all_of[condIndex]
    if (!cond) return

    cond.operator = op
  })
}

function setBalanceValue(groupIndex: number, condIndex: number, raw: string) {
  const n = raw === '' ? 0 : Number(raw)
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return

    const cond = group.all_of[condIndex]
    if (!cond) return

    cond.value = Number.isFinite(n) ? n : 0
  })
}

const groupSelections = reactive<Record<number, Record<number, number[]>>>({})
const userSelections = reactive<Record<number, Record<number, number[]>>>({})
const userSearch = ref('')
const manualUserID = ref('')

const filteredUsers = computed(() => {
  const q = userSearch.value.trim().toLowerCase()
  if (!q) return props.users.slice(0, 100)
  return props.users.filter((user) => {
    const email = user.email?.toLowerCase() ?? ''
    const username = user.username?.toLowerCase() ?? ''
    return email.includes(q) || username.includes(q) || String(user.id).includes(q)
  }).slice(0, 100)
})

function ensureGroupSelectionPath(groupIndex: number, condIndex: number) {
  if (!groupSelections[groupIndex]) groupSelections[groupIndex] = {}
  if (!groupSelections[groupIndex][condIndex]) groupSelections[groupIndex][condIndex] = []
}

function ensureUserSelectionPath(groupIndex: number, condIndex: number) {
  if (!userSelections[groupIndex]) userSelections[groupIndex] = {}
  if (!userSelections[groupIndex][condIndex]) userSelections[groupIndex][condIndex] = []
}

// Sync from modelValue to subscriptionSelections (one-way: model -> local state)
watch(
  () => props.modelValue,
  (v) => {
    const groups = v?.any_of ?? []
    for (let gi = 0; gi < groups.length; gi++) {
      const allOf = groups[gi]?.all_of ?? []
      for (let ci = 0; ci < allOf.length; ci++) {
        const c = allOf[ci]
        if (c?.type === 'subscription' || c?.type === 'group') {
          ensureGroupSelectionPath(gi, ci)
          // Only update if different to avoid triggering unnecessary updates
          const newIds = (c.group_ids ?? []).slice()
          const currentIds = groupSelections[gi]?.[ci] ?? []
          if (JSON.stringify(newIds.sort()) !== JSON.stringify(currentIds.sort())) {
            groupSelections[gi][ci] = newIds
          }
        } else if (c?.type === 'user') {
          ensureUserSelectionPath(gi, ci)
          const newIds = (c.user_ids ?? []).slice()
          const currentIds = userSelections[gi]?.[ci] ?? []
          if (JSON.stringify(newIds.sort()) !== JSON.stringify(currentIds.sort())) {
            userSelections[gi][ci] = newIds
          }
        }
      }
    }
  },
  { immediate: true }
)

// Sync from subscriptionSelections to modelValue (one-way: local state -> model)
// Use a debounced approach to avoid infinite loops
let syncTimeout: ReturnType<typeof setTimeout> | null = null
function syncSelectionsToModel() {
  // Debounce the sync to avoid rapid fire updates
  if (syncTimeout) clearTimeout(syncTimeout)

  syncTimeout = setTimeout(() => {
    // Build the new targeting state
    const newTargeting: TargetingDraft = JSON.parse(JSON.stringify(props.modelValue ?? { any_of: [] }))
    if (!newTargeting.any_of) newTargeting.any_of = []

    const groups = newTargeting.any_of ?? []
    for (let gi = 0; gi < groups.length; gi++) {
      const allOf = groups[gi]?.all_of ?? []
      for (let ci = 0; ci < allOf.length; ci++) {
        const c = allOf[ci]
        if (c?.type === 'subscription' || c?.type === 'group') {
          ensureGroupSelectionPath(gi, ci)
          c.operator = 'in' as AnnouncementOperator
          c.group_ids = (groupSelections[gi]?.[ci] ?? []).slice()
        } else if (c?.type === 'user') {
          ensureUserSelectionPath(gi, ci)
          c.operator = 'in' as AnnouncementOperator
          c.user_ids = (userSelections[gi]?.[ci] ?? []).slice()
        }
      }
    }

    // Only emit if there's an actual change (deep comparison)
    if (JSON.stringify(props.modelValue) !== JSON.stringify(newTargeting)) {
      emit('update:modelValue', newTargeting)
    }
  }, 0)
}

watch(
  () => groupSelections,
  syncSelectionsToModel,
  { deep: true }
)

watch(
  () => userSelections,
  syncSelectionsToModel,
  { deep: true }
)

function setUserSelected(groupIndex: number, condIndex: number, userID: number, checked: boolean) {
  ensureUserSelectionPath(groupIndex, condIndex)
  const current = userSelections[groupIndex][condIndex] ?? []
  userSelections[groupIndex][condIndex] = checked
    ? Array.from(new Set([...current, userID]))
    : current.filter((id) => id !== userID)
}

function addManualUserID(groupIndex: number, condIndex: number) {
  const userID = Number(manualUserID.value)
  if (!Number.isInteger(userID) || userID <= 0) return
  setUserSelected(groupIndex, condIndex, userID, true)
  manualUserID.value = ''
}

const validationError = computed(() => {
  if (mode.value !== 'custom') return ''

  const groups = anyOf.value
  if (groups.length === 0) return t('admin.announcements.form.addOrGroup')

  if (groups.length > 50) return 'any_of > 50'

  for (const g of groups) {
    const allOf = g?.all_of ?? []
    if (allOf.length === 0) return t('admin.announcements.form.addAndCondition')
    if (allOf.length > 50) return 'all_of > 50'

    for (const c of allOf) {
      if (c.type === 'subscription') {
        if (!c.group_ids || c.group_ids.length === 0) return t('admin.announcements.form.selectPackages')
      } else if (c.type === 'group') {
        if (!c.group_ids || c.group_ids.length === 0) return t('admin.announcements.form.selectAPIKeyGroups')
      } else if (c.type === 'user') {
        if (!c.user_ids || c.user_ids.length === 0) return t('admin.announcements.form.selectUsers')
      }
    }
  }

  return ''
})
</script>
