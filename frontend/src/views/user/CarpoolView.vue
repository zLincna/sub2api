<template>
  <AppLayout>
    <div class="space-y-6">
      <template v-if="paymentPhase === 'paying'">
        <PaymentStatusPanel
          :order-id="paymentState.orderId"
          :qr-code="paymentState.qrCode"
          :expires-at="paymentState.expiresAt"
          :payment-type="paymentState.paymentType"
          :pay-url="paymentState.payUrl"
          :order-type="paymentState.orderType"
          :currency="paymentState.currency"
          @done="resetPayment"
          @settled="onPaymentSettled"
        />
      </template>

      <template v-else>
        <div v-if="loading" class="flex justify-center py-16">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
        </div>

        <div v-else class="space-y-4">
          <div class="flex flex-col gap-3 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center sm:justify-between">
            <div class="inline-flex w-full rounded-lg bg-gray-100 p-1 dark:bg-dark-800 sm:w-auto">
              <button
                v-for="tab in pageTabs"
                :key="tab.key"
                type="button"
                class="flex-1 rounded-md px-5 py-2.5 text-center text-sm font-medium transition sm:flex-none"
                :class="activeTab === tab.key ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="activeTab = tab.key"
              >
                {{ tab.label }}
              </button>
            </div>
            <button v-if="activeTab === 'hall'" type="button" class="btn btn-secondary btn-sm sm:w-auto" @click="loadAll">刷新</button>
          </div>

          <div v-if="activeTab === 'hall'" class="min-w-0">
            <div v-if="cards.length === 0" class="rounded-lg border border-gray-200 bg-white p-12 text-center text-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-400">
              拼车大厅暂未开放
            </div>

            <section v-else class="space-y-6">
              <div v-for="group in groupedCards" :key="group.key" class="space-y-3">
                <div class="flex items-center justify-between">
                  <div>
                    <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ group.label }}</h3>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ group.cards.length }} 种车正在排队</p>
                  </div>
                </div>
                <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                  <article
                    v-for="card in group.cards"
                    :key="card.vehicle_type.id"
                    class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-900"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0">
                        <div class="mb-2 flex flex-wrap gap-2">
                          <span class="rounded bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ segmentLabel(card.vehicle_type) }}</span>
                          <span class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ card.vehicle_type.seat_count }} 人车</span>
                        </div>
                        <h4 class="text-lg font-semibold text-gray-900 dark:text-white">{{ card.vehicle_type.name }}</h4>
                        <p class="mt-2 whitespace-pre-line text-sm leading-6 text-gray-500 dark:text-dark-400">{{ card.vehicle_type.description || '满员后管理员统一采购账号和静态住宅 IP。' }}</p>
                      </div>
                      <div class="flex shrink-0 flex-col items-end gap-2">
                        <span class="rounded-md bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">
                          {{ card.paid_count }}/{{ card.seat_count }}
                        </span>
                        <span class="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-[11px] font-semibold text-amber-700 shadow-sm dark:border-amber-900/50 dark:bg-amber-900/25 dark:text-amber-200">
                          已拼成 {{ displayCompletedCount(card) }}
                        </span>
                      </div>
                    </div>
                    <div class="mt-5 space-y-3">
                      <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                        <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: progressWidth(progress(card)) }"></div>
                      </div>
                      <div class="grid grid-cols-3 gap-2 text-sm">
                        <div>
                          <p class="text-xs text-gray-400">包车价</p>
                          <p class="font-semibold text-gray-900 dark:text-white">¥{{ money(card.vehicle_type.total_price) }}</p>
                        </div>
                        <div>
                          <p class="text-xs text-gray-400">人均</p>
                          <p class="font-semibold text-emerald-600 dark:text-emerald-400">¥{{ money(card.vehicle_type.unit_price) }}</p>
                        </div>
                        <div>
                          <p class="text-xs text-gray-400">可退款</p>
                          <p class="font-semibold text-gray-900 dark:text-white">{{ refundWaitLabel(card.vehicle_type) }}</p>
                        </div>
                      </div>
                      <div class="flex flex-wrap gap-2 text-xs">
                        <span v-if="card.vehicle_type.require_static_ip" class="rounded bg-blue-50 px-2 py-1 text-blue-700 dark:bg-blue-900/30 dark:text-blue-200">静态住宅 IP</span>
                        <span v-if="card.vehicle_type.support_revenue_pool" class="rounded bg-amber-50 px-2 py-1 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">预留中转投入</span>
                      </div>
                      <button
                        type="button"
                        class="btn w-full"
                        :class="myParticipationForCard(card) ? 'btn-secondary cursor-not-allowed opacity-80' : 'btn-primary'"
                        :disabled="Boolean(myParticipationForCard(card))"
                        @click="selectCard(card)"
                      >
                        {{ hallCardButtonLabel(card) }}
                      </button>
                    </div>
                  </article>
                </div>
              </div>
            </section>
          </div>

          <section v-else class="min-w-0 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-4 flex items-center justify-between">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">我的拼车</h3>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">这里只展示已完成支付的拼车记录。拼车成功后等待管理员发车，发车后可在详情中查看沟通方式、订阅分组和交付凭证。</p>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" @click="loadMine">刷新</button>
            </div>
            <div v-if="mine.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">暂无拼车记录</div>
            <div v-else class="space-y-4">
              <article v-for="item in mine" :key="item.id" class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
                <div class="flex flex-col gap-3 border-b border-gray-100 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-800/60 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <div class="mb-2 flex flex-wrap gap-2">
                      <span class="rounded bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ item.edges?.vehicle_type ? segmentLabel(item.edges.vehicle_type) : '拼车订单' }}</span>
                      <span class="rounded bg-white px-2 py-1 text-xs text-gray-500 dark:bg-dark-900 dark:text-dark-300">{{ item.edges?.session?.session_no || '等待分配轮次' }}</span>
                    </div>
                    <h4 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ item.edges?.vehicle_type?.name || item.vehicle_type_id }}</h4>
                    <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ participantStageText(item) }}</p>
                  </div>
                  <div class="flex shrink-0 flex-col items-start gap-2 sm:items-end">
                    <span :class="['rounded px-2.5 py-1 text-xs font-medium', carpoolStateBadgeClass(item.edges?.session?.status || item.status)]">{{ statusLabel(item.edges?.session?.status || item.status) }}</span>
                    <span class="text-sm font-semibold text-gray-900 dark:text-white">¥{{ money(item.amount) }}</span>
                  </div>
                </div>
                <div class="p-4">
                  <div class="mb-3 flex items-center justify-between text-xs text-gray-500 dark:text-dark-400">
                    <span>拼车进度</span>
                    <span class="font-semibold text-gray-900 dark:text-white">{{ sessionProgressLabel(item) }}</span>
                  </div>
                  <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                    <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: progressWidth(participantProgress(item)) }"></div>
                  </div>
                  <div class="mt-4 grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-4">
                    <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                      <p class="text-xs text-gray-400">等待截止</p>
                      <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatTime(item.wait_until) }}</p>
                    </div>
                    <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                      <p class="text-xs text-gray-400">拼车成功时间</p>
                      <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatTime(item.edges?.session?.filled_at) }}</p>
                    </div>
                    <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                      <p class="text-xs text-gray-400">服务到期时间</p>
                      <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatTime(item.edges?.session?.service_ended_at) }}</p>
                    </div>
                    <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                      <p class="text-xs text-gray-400">到期剩余</p>
                      <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ timeLeftLabel(item.edges?.session?.service_ended_at) }}</p>
                    </div>
                  </div>
                </div>
                <div class="flex flex-col gap-2 border-t border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
                  <p class="text-xs text-gray-500 dark:text-dark-400">支付时间：{{ formatTime(item.paid_at || item.created_at) }}</p>
                  <div class="flex justify-end gap-2">
                  <button v-if="canRequestRefund(item)" type="button" class="btn btn-secondary btn-sm text-red-600 dark:text-red-300" @click="openRefundDialog(item)">发起退款</button>
                  <button type="button" class="btn btn-secondary btn-sm" @click="openDetail(item)">查看详情</button>
                  </div>
                </div>
              </article>
            </div>
          </section>
        </div>
      </template>
    </div>

    <div v-if="detailItem" class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
      <div class="max-h-[92vh] w-full max-w-6xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
        <div class="flex items-start justify-between gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div class="min-w-0">
            <h3 class="truncate text-lg font-semibold text-gray-900 dark:text-white">{{ detailVehicleName }}</h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ detailSession?.session_no || '等待成团' }} · {{ statusLabel(detailSession?.status || detailItem.status) }}</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="closeDetail">关闭</button>
        </div>
        <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
          <div class="inline-flex w-full rounded-lg bg-gray-100 p-1 dark:bg-dark-800 sm:w-auto">
            <button
              v-for="tab in detailTabs"
              :key="tab.key"
              type="button"
              class="flex-1 rounded-md px-4 py-2 text-center text-sm font-medium transition sm:flex-none"
              :class="detailTab === tab.key ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
              @click="detailTab = tab.key"
            >
              {{ tab.label }}
            </button>
          </div>
        </div>
        <div class="max-h-[76vh] overflow-y-auto p-5">
          <div v-if="detailLoading" class="flex justify-center py-16">
            <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
          </div>
          <div v-else class="space-y-5">
            <section v-if="detailTab === 'info'" class="overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700">
              <div class="flex flex-col gap-2 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h4 class="text-sm font-semibold text-gray-900 dark:text-white">拼车信息</h4>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">进度、交付信息和凭证集中展示在这里；用量请切换到“车成员与用量”。</p>
                </div>
                <span class="text-xs text-gray-500 dark:text-dark-400">{{ detailSession?.session_no || '等待成团' }} · {{ statusLabel(detailSession?.status || detailItem.status) }}</span>
              </div>

              <div class="space-y-5 p-4">
                <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
                  <div class="mb-2 flex items-center justify-between text-sm">
                    <span class="font-semibold text-gray-900 dark:text-white">拼车进度</span>
                    <span class="text-gray-500 dark:text-dark-400">{{ sessionProgressLabel(detailItem) }}</span>
                  </div>
                  <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                    <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: progressWidth(participantProgress(detailItem)) }"></div>
                  </div>
                  <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">{{ participantStageText(detailItem) }}</p>
                </div>

                <div v-if="detailStateNotice" class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-100">
                  {{ detailStateNotice }}
                </div>

                <div class="grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-4">
                  <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
                    <p class="text-xs text-gray-500 dark:text-dark-400">支付金额</p>
                    <p class="mt-1 font-semibold text-gray-900 dark:text-white">¥{{ money(detailItem.amount) }}</p>
                  </div>
                  <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
                    <p class="text-xs text-gray-500 dark:text-dark-400">拼车成功时间</p>
                    <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatTime(detailSession?.filled_at) }}</p>
                  </div>
                  <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
                    <p class="text-xs text-gray-500 dark:text-dark-400">服务开始</p>
                    <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatTime(detailSession?.service_started_at || detailSession?.provisioned_at) }}</p>
                  </div>
                  <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
                    <p class="text-xs text-gray-500 dark:text-dark-400">服务到期</p>
                    <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatTime(detailSession?.service_ended_at) }}</p>
                  </div>
                </div>

                <div class="grid gap-4 lg:grid-cols-2">
                  <section class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
                    <h4 class="text-sm font-semibold text-gray-900 dark:text-white">发车后如何使用</h4>
                    <ol class="mt-3 space-y-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                      <li>1. 管理员发车后会分配订阅分组，并上传交付凭证。</li>
                      <li>2. 进入“订阅”或 API Key 配置页使用已分配的分组；车内沟通信息以本详情为准。</li>
                      <li>3. 账号整体用量、自己和其他成员用量请切换到“车成员与用量”查看。</li>
                    </ol>
                  </section>

                  <section v-if="deliverySummary(detailItem)" class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
                    <h4 class="text-sm font-semibold text-gray-900 dark:text-white">交付信息</h4>
                    <p class="mt-2 whitespace-pre-line text-sm leading-6 text-gray-600 dark:text-dark-300">{{ deliverySummary(detailItem) }}</p>
                  </section>
                </div>

                <section v-if="detailVouchers.length" class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
                  <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">交付凭证</h4>
                  <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    <button
                      v-for="voucher in detailVouchers"
                      :key="voucher.id"
                      type="button"
                      class="overflow-hidden rounded-md border border-gray-100 bg-gray-50 text-left transition hover:border-primary-300 hover:shadow-sm dark:border-dark-700 dark:bg-dark-800"
                      @click="previewVoucher = voucher"
                    >
                      <img :src="voucher.file_url" :alt="voucher.file_name" class="h-32 w-full object-cover" />
                      <div class="p-3">
                        <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ voucher.file_name }}</p>
                        <p v-if="voucher.description" class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-dark-400">{{ voucher.description }}</p>
                      </div>
                    </button>
                  </div>
                </section>
              </div>
            </section>

            <section v-else class="space-y-4">
              <section class="overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700">
                <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                  <div>
                    <h4 class="text-sm font-semibold text-gray-900 dark:text-white">账号整体用量</h4>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">展示管理员分配订阅分组后，账号池整体 5h / 7d 窗口。</p>
                  </div>
                  <span class="text-xs text-gray-500 dark:text-dark-400">{{ accountWindowPairs.length }} 个账号</span>
                </div>
                <div class="p-4">
                  <div v-if="accountWindowPairs.length === 0" class="rounded-md bg-gray-50 p-4 text-sm leading-6 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                    发车后管理员分配订阅分组，账号池整体用量会在这里展示。
                  </div>
                  <div v-else class="grid gap-3 lg:grid-cols-2">
                    <div v-for="item in accountWindowPairs" :key="item.account_id" class="rounded-md border border-gray-100 p-3 dark:border-dark-700">
                      <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ item.account_name || `账号 ${item.account_id}` }}</p>
                      <div class="mt-3 grid gap-3 sm:grid-cols-2">
                        <div v-for="win in item.windows" :key="win.window" class="rounded bg-gray-50 p-3 dark:bg-dark-800">
                          <div class="mb-2 flex flex-wrap items-center gap-1.5 text-[11px] text-gray-500 dark:text-dark-400">
                            <span class="rounded bg-white px-2 py-1 dark:bg-dark-900">{{ compactNumber(win.usage?.request_count || 0) }} req</span>
                            <span class="rounded bg-white px-2 py-1 dark:bg-dark-900">{{ compactNumber(win.usage?.total_tokens || 0) }}</span>
                            <span class="rounded bg-white px-2 py-1 dark:bg-dark-900" title="账号成本">A ${{ money(win.usage?.total_actual_cost || 0) }}</span>
                            <span class="rounded bg-white px-2 py-1 dark:bg-dark-900" title="用户计费">U ${{ money(win.usage?.total_cost || 0) }}</span>
                          </div>
                          <div class="flex items-center gap-2 text-xs">
                            <span :class="['w-9 shrink-0 rounded px-1.5 py-0.5 text-center font-medium', usageWindowBadgeClass(win.window)]">{{ windowShortLabel(win.window) }}</span>
                            <div class="h-1.5 w-16 shrink-0 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                              <div :class="['h-full rounded-full', usageWindowBarClass(win.utilization)]" :style="{ width: Math.min(100, Math.max(0, win.utilization || 0)) + '%' }"></div>
                            </div>
                            <span class="w-10 shrink-0 text-right font-medium text-gray-700 dark:text-dark-200">{{ usagePercent(win.utilization) }}</span>
                            <span class="shrink-0 text-gray-400">{{ resetCountdown(win.resets_at, win.utilization) }}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              <section class="overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700">
                <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                  <div>
                    <h4 class="text-sm font-semibold text-gray-900 dark:text-white">车成员与用量</h4>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">展示当前自己的用量和其他成员的用量。</p>
                  </div>
                  <span class="text-xs text-gray-500 dark:text-dark-400">{{ detailData?.members?.length || 0 }} 人</span>
                </div>
                <div class="p-4">
                  <div v-if="!detailData?.members?.length" class="rounded-md bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-800 dark:text-dark-400">暂无成员用量数据</div>
                  <div v-else class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  <article v-for="member in sortedMembers" :key="member.participant_id" class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
                    <div class="flex items-center gap-3">
                      <img v-if="member.avatar_url" :src="member.avatar_url" :alt="member.display_name" class="h-11 w-11 rounded-full object-cover" />
                      <div v-else class="flex h-11 w-11 items-center justify-center rounded-full bg-primary-50 text-base font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ member.initial || 'U' }}</div>
                      <div class="min-w-0 flex-1">
                        <div class="flex min-w-0 items-center gap-2">
                          <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ member.display_name }}</p>
                          <span v-if="member.is_self" class="shrink-0 rounded bg-emerald-50 px-1.5 py-0.5 text-[11px] text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">我</span>
                        </div>
                        <p class="text-xs text-gray-500 dark:text-dark-400">{{ statusLabel(member.status) }}</p>
                      </div>
                    </div>
                    <div class="mt-4 grid grid-cols-3 gap-2 text-xs">
                      <div class="rounded bg-gray-50 p-2 dark:bg-dark-800">
                        <p class="text-gray-400">总请求</p>
                        <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ compactNumber(member.usage.request_count) }}</p>
                      </div>
                      <div class="rounded bg-gray-50 p-2 dark:bg-dark-800">
                        <p class="text-gray-400">5h</p>
                        <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ compactNumber(member.windows.five_hour.request_count) }} 次</p>
                      </div>
                      <div class="rounded bg-gray-50 p-2 dark:bg-dark-800">
                        <p class="text-gray-400">7d</p>
                        <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ compactNumber(member.windows.seven_day.request_count) }} 次</p>
                      </div>
                    </div>
                    <div class="mt-3 rounded bg-gray-50 p-2 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                      总消耗 ${{ money(member.usage.total_actual_cost) }} · {{ compactNumber(member.usage.total_tokens) }} tokens
                    </div>
                  </article>
                  </div>
                </div>
              </section>
            </section>
          </div>
        </div>
      </div>
    </div>

    <div v-if="previewVoucher" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm" @click.self="previewVoucher = null">
      <div class="flex max-h-[92vh] w-full max-w-5xl flex-col overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-dark-900">
        <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div class="min-w-0">
            <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ previewVoucher.file_name || '交付凭证' }}</h3>
            <p v-if="previewVoucher.description" class="mt-1 line-clamp-1 text-sm text-gray-500 dark:text-dark-400">{{ previewVoucher.description }}</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="previewVoucher = null">关闭</button>
        </div>
        <div class="flex min-h-0 flex-1 items-center justify-center bg-gray-950 p-4">
          <img :src="previewVoucher.file_url" :alt="previewVoucher.file_name" class="max-h-[74vh] max-w-full rounded object-contain" />
        </div>
      </div>
    </div>

    <div v-if="selected" class="fixed inset-0 z-50 flex items-center justify-center bg-gray-950/60 p-4 backdrop-blur-md">
      <div class="max-h-[94vh] w-full max-w-[860px] overflow-hidden rounded-xl bg-white shadow-2xl ring-1 ring-gray-900/10 dark:bg-dark-900 dark:ring-white/10">
        <div class="flex items-start justify-between gap-4 px-6 pb-5 pt-7 sm:px-9">
          <div>
            <h3 class="text-2xl font-semibold text-gray-950 dark:text-white">{{ selected.vehicle_type.name }} · 确认拼车</h3>
            <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-300">{{ segmentLabel(selected.vehicle_type) }}，支付 ¥{{ money(selected.vehicle_type.unit_price) }} 后进入当前拼车队列。</p>
          </div>
          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-gray-700 transition hover:bg-gray-100 hover:text-gray-950 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            aria-label="关闭"
            @click="selected = null"
          >
            <Icon name="x" size="lg" :stroke-width="1.8" />
          </button>
        </div>

        <div class="max-h-[68vh] space-y-6 overflow-y-auto px-6 pb-6 sm:px-9">
          <div class="grid overflow-hidden rounded-lg border border-teal-100 bg-gradient-to-r from-teal-50/80 to-cyan-50/50 shadow-sm dark:border-teal-900/50 dark:from-teal-950/30 dark:to-cyan-950/20 sm:grid-cols-3">
            <div class="flex items-center gap-4 border-b border-teal-100 px-5 py-5 dark:border-teal-900/50 sm:border-b-0 sm:border-r">
              <div class="flex h-11 w-11 shrink-0 items-center justify-center text-teal-500 dark:text-teal-300">
                <Icon name="users" size="xl" :stroke-width="1.8" />
              </div>
              <div>
                <p class="text-sm font-medium text-gray-500 dark:text-dark-300">拼车人数</p>
                <p class="mt-1 text-xl font-semibold text-gray-950 dark:text-white">{{ selected.seat_count }} 人</p>
              </div>
            </div>
            <div class="flex items-center gap-4 border-b border-teal-100 px-5 py-5 dark:border-teal-900/50 sm:border-b-0 sm:border-r">
              <div class="flex h-11 w-11 shrink-0 items-center justify-center text-teal-500 dark:text-teal-300">
                <Icon name="tag" size="xl" :stroke-width="1.8" />
              </div>
              <div>
                <p class="text-sm font-medium text-gray-500 dark:text-dark-300">单人价格</p>
                <p class="mt-1 text-xl font-semibold text-gray-950 dark:text-white">¥{{ money(selected.vehicle_type.unit_price) }}</p>
              </div>
            </div>
            <div class="flex items-center gap-4 px-5 py-5">
              <div class="flex h-11 w-11 shrink-0 items-center justify-center text-teal-500 dark:text-teal-300">
                <Icon name="wallet" size="xl" :stroke-width="1.8" />
              </div>
              <div>
                <p class="text-sm font-medium text-gray-500 dark:text-dark-300">合计金额</p>
                <p class="mt-1 text-2xl font-bold text-teal-600 dark:text-teal-300">¥{{ money(selected.vehicle_type.total_price) }}</p>
              </div>
            </div>
          </div>

          <section>
            <h4 class="mb-3 text-sm font-semibold text-gray-950 dark:text-white">支付方式</h4>
            <div class="grid gap-4 sm:grid-cols-2">
              <button
                type="button"
                class="flex min-h-[96px] items-center justify-between rounded-lg border border-teal-500 bg-white p-5 text-left shadow-sm ring-1 ring-teal-100 transition hover:bg-teal-50/40 dark:bg-dark-900 dark:ring-teal-900/40 dark:hover:bg-teal-950/20"
                @click="joinForm.payment_type = 'alipay'"
              >
                <span class="flex items-center gap-4">
                  <span class="flex h-11 w-11 items-center justify-center rounded-lg bg-[#1677ff] text-2xl font-bold text-white">支</span>
                  <span>
                    <span class="flex items-center gap-2 text-base font-semibold text-gray-950 dark:text-white">
                      支付宝
                      <span class="rounded bg-teal-50 px-2 py-0.5 text-xs font-medium text-teal-700 dark:bg-teal-900/30 dark:text-teal-200">推荐</span>
                    </span>
                    <span class="mt-2 block text-sm text-gray-500 dark:text-dark-300">当前支持</span>
                  </span>
                </span>
                <span class="flex h-6 w-6 items-center justify-center rounded-full bg-teal-600 text-white">
                  <Icon name="check" size="xs" :stroke-width="2.5" />
                </span>
              </button>

              <div class="flex min-h-[96px] items-center justify-between rounded-lg border border-gray-200 bg-gray-50 p-5 text-left opacity-75 dark:border-dark-700 dark:bg-dark-800">
                <span class="flex items-center gap-4">
                  <span class="flex h-11 w-11 items-center justify-center rounded-full bg-gray-300 text-white dark:bg-dark-600">
                    <Icon name="chat" size="lg" />
                  </span>
                  <span>
                    <span class="flex items-center gap-2 text-base font-semibold text-gray-500 dark:text-dark-300">
                      微信支付
                      <span class="rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500 dark:bg-dark-700 dark:text-dark-300">即将支持</span>
                    </span>
                    <span class="mt-2 block text-sm text-gray-400 dark:text-dark-400">敬请期待</span>
                  </span>
                </span>
              </div>
            </div>
            <p class="mt-4 flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300">
              <Icon name="shield" size="sm" class="text-teal-600 dark:text-teal-300" />
              支付后 {{ refundWaitLabel(selected.vehicle_type) }} 内可申请退款，可在“我的拼车”中发起。
            </p>
          </section>

          <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
            <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h4 class="text-sm font-semibold text-gray-950 dark:text-white">{{ notice?.title || '拼车用户须知' }}</h4>
              <div class="flex items-center gap-3">
                <span class="text-xs text-gray-400">v{{ notice?.version || 1 }}</span>
                <Icon name="chevronUp" size="sm" class="text-gray-500 dark:text-dark-300" />
              </div>
            </div>
            <div ref="noticeBox" class="h-44 overflow-y-auto bg-gray-50 p-4 text-sm leading-7 text-gray-700 dark:bg-dark-800 dark:text-dark-200" @scroll="onNoticeScroll">
              <div class="carpool-notice-markdown" v-html="renderedNoticeHtml"></div>
            </div>
            <label class="mt-4 flex items-center gap-2 px-4 text-sm text-gray-700 dark:text-dark-200">
              <input v-model="noticeAccepted" type="checkbox" class="rounded border-gray-300 text-teal-600 focus:ring-teal-500" :disabled="!noticeScrolled" />
              <span>我已阅读并同意 <span class="font-medium text-teal-600 dark:text-teal-300">《拼车用户须知》</span></span>
            </label>
            <p v-if="!noticeScrolled" class="mb-4 ml-11 mt-1 text-xs text-orange-500 dark:text-orange-300">请先滚动阅读到须知底部</p>
            <p v-else-if="!noticeAccepted" class="mb-4 ml-11 mt-1 text-xs text-orange-500 dark:text-orange-300">请先勾选协议后再支付</p>
          </section>
        </div>
        <div class="flex justify-end gap-4 px-6 pb-7 pt-3 sm:px-9">
          <button type="button" class="rounded-lg border border-gray-200 bg-white px-8 py-3 text-sm font-semibold text-gray-800 transition hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-100 dark:hover:bg-dark-800" @click="selected = null">取消</button>
          <button type="button" class="rounded-lg bg-teal-600 px-9 py-3 text-sm font-semibold text-white shadow-lg shadow-teal-600/20 transition hover:bg-teal-700 disabled:cursor-not-allowed disabled:opacity-60" :disabled="joining || !noticeAccepted" @click="joinSelected">
            {{ joining ? '处理中...' : '确认并支付' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="refundTarget" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
      <div class="w-full max-w-md overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">发起拼车退款</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">系统会将该拼车记录转入待退款，管理员会按所选方式处理。</p>
        </div>
        <div class="space-y-4 p-5">
          <div class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
            <div class="flex justify-between gap-3">
              <span class="text-gray-500 dark:text-dark-400">拼车</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ refundTarget.edges?.vehicle_type?.name || refundTarget.vehicle_type_id }}</span>
            </div>
            <div class="mt-2 flex justify-between gap-3">
              <span class="text-gray-500 dark:text-dark-400">金额</span>
              <span class="font-medium text-gray-900 dark:text-white">¥{{ money(refundTarget.amount) }}</span>
            </div>
          </div>
          <label class="block">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">退款方式</span>
            <select v-model="refundForm.refund_method" class="input mt-2 h-11">
              <option v-for="method in refundMethodsFor(refundTarget)" :key="method" :value="method">{{ refundLabel(method) }}</option>
            </select>
          </label>
          <p class="text-xs leading-5 text-amber-600 dark:text-amber-300">发起后该座位会释放；如果想继续等待拼车，请不要发起退款。</p>
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" @click="refundTarget = null">取消</button>
          <button type="button" class="btn btn-primary" :disabled="refundSubmitting" @click="submitRefundRequest">
            {{ refundSubmitting ? '处理中...' : '确认发起退款' }}
          </button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AppLayout from '@/components/layout/AppLayout.vue'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { carpoolAPI } from '@/api/carpool'
import type { CarpoolAccountWindowUsage, CarpoolCard, CarpoolNoticeVersion, CarpoolParticipant, CarpoolUserDetail, CarpoolVehicleType, CarpoolVoucher } from '@/types/carpool'
import type { CreateOrderResult } from '@/types/payment'
import { decidePaymentLaunch, normalizeVisibleMethod, type PaymentRecoverySnapshot } from '@/components/payment/paymentFlow'
import { getPaymentPopupFeatures } from '@/components/payment/providerConfig'
import { useAppStore } from '@/stores'

type PaymentOutcome = 'success' | 'cancelled' | 'expired'

const router = useRouter()
const appStore = useAppStore()
const loading = ref(false)
const joining = ref(false)
const activeTab = ref<'hall' | 'mine'>('hall')
const cards = ref<CarpoolCard[]>([])
const mine = ref<CarpoolParticipant[]>([])
const notice = ref<CarpoolNoticeVersion | null>(null)
const selected = ref<CarpoolCard | null>(null)
const detailItem = ref<CarpoolParticipant | null>(null)
const detailData = ref<CarpoolUserDetail | null>(null)
const detailLoading = ref(false)
const detailTab = ref<'info' | 'members'>('info')
const previewVoucher = ref<CarpoolVoucher | null>(null)
const refundTarget = ref<CarpoolParticipant | null>(null)
const refundSubmitting = ref(false)
const noticeScrolled = ref(false)
const noticeAccepted = ref(false)
const paymentPhase = ref<'select' | 'paying'>('select')
const pageTabs = [
  { key: 'hall', label: '拼车大厅' },
  { key: 'mine', label: '我的拼车' },
] as const
const detailTabs = [
  { key: 'info', label: '拼车信息' },
  { key: 'members', label: '车成员与用量' },
] as const
const paymentState = ref<PaymentRecoverySnapshot>({
  orderId: 0,
  amount: 0,
  qrCode: '',
  expiresAt: '',
  paymentType: '',
  payUrl: '',
  outTradeNo: '',
  clientSecret: '',
  intentId: '',
  currency: '',
  countryCode: '',
  paymentEnv: '',
  payAmount: 0,
  orderType: 'carpool',
  paymentMode: '',
  resumeToken: '',
  createdAt: Date.now(),
})

const joinForm = reactive({
  payment_type: 'alipay',
})

const refundForm = reactive({
  refund_method: 'balance',
})

marked.setOptions({
  breaks: true,
  gfm: true,
})

const groupedCards = computed(() => {
  const map = new Map<string, { key: string; label: string; cards: CarpoolCard[] }>()
  for (const card of cards.value) {
    const vt = card.vehicle_type
    const key = `${normalizeCarpoolCode(vt.product, 'custom')}:${normalizeCarpoolCode(vt.plan_tier, 'custom')}:${normalizeCarpoolCode(vt.multiplier, 'custom')}`
    if (!map.has(key)) {
      map.set(key, { key, label: segmentLabel(vt), cards: [] })
    }
    map.get(key)!.cards.push(card)
  }
  return Array.from(map.values())
})

const detailSession = computed(() => detailData.value?.session || detailItem.value?.edges?.session || null)

const detailVehicleName = computed(() => (
  detailData.value?.session?.edges?.vehicle_type?.name ||
  detailItem.value?.edges?.vehicle_type?.name ||
  String(detailItem.value?.vehicle_type_id || '拼车详情')
))

const detailVouchers = computed(() => detailSession.value?.edges?.vouchers || [])

const sortedMembers = computed(() => {
  return [...(detailData.value?.members || [])].sort((a, b) => {
    if (a.is_self !== b.is_self) return a.is_self ? -1 : 1
    return Number(b.usage?.total_actual_cost || 0) - Number(a.usage?.total_actual_cost || 0)
  })
})

const accountWindowPairs = computed(() => {
  const map = new Map<number, { account_id: number; account_name: string; windows: CarpoolAccountWindowUsage[] }>()
  for (const win of detailData.value?.account_windows || []) {
    if (!map.has(win.account_id)) {
      map.set(win.account_id, { account_id: win.account_id, account_name: win.account_name, windows: [] })
    }
    map.get(win.account_id)!.windows.push(win)
  }
  return Array.from(map.values()).map((item) => ({
    ...item,
    windows: item.windows.sort((a, b) => windowSort(a.window) - windowSort(b.window)),
  }))
})

const detailStateNotice = computed(() => {
  const session = detailSession.value
  if (!detailItem.value || !session) return ''
  if (session.status === 'full') return '拼车已成功，正在等待管理员采购账号、配置代理和分配订阅分组。完成后这里会展示交付信息，成员用量页也会显示账号池窗口。'
  if (session.status === 'provisioning') return '管理员正在进行采购和订阅分配，当前暂未完成发车。请等待短信或站内状态更新。'
  if (session.status === 'active' && !deliverySummary(detailItem.value)) return '本轮已经发车，但暂未填写交付说明或凭证。如无法使用，请联系管理员确认订阅分组。'
  return ''
})

const renderedNoticeHtml = computed(() => {
  const content = notice.value?.content_md?.trim() || [
    '拼车为多人共同等待成团，人满后由管理员采购和交付。',
    '',
    '如果未在等待时间内成团，系统将按照规则支持发起退款。',
    '',
    '发车后的账号、代理、使用方式和沟通方式以管理员交付信息为准。',
    '',
    '付款成功即视为您已阅读并同意《拼车用户须知》，并同意遵守相关规则。',
  ].join('\n')
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

function progress(card: CarpoolCard) {
  return Math.min(100, Math.round((card.paid_count / Math.max(card.seat_count, 1)) * 100))
}

function progressWidth(value: number) {
  return `${Math.min(100, Math.max(0, value))}%`
}

function displayCompletedCount(card: CarpoolCard) {
  return Number(card.display_completed_count ?? ((card.vehicle_type.completed_base_count || 0) + (card.completed_count || 0)))
}

function refundWaitLabel(vt?: Pick<CarpoolVehicleType, 'refund_wait_hours'>) {
  const hours = Number(vt?.refund_wait_hours || 2)
  if (hours % 24 === 0) return `${hours / 24} 天`
  return `${hours} 小时`
}

function selectCard(card: CarpoolCard) {
  if (myParticipationForCard(card)) return
  selected.value = card
  joinForm.payment_type = 'alipay'
  noticeScrolled.value = false
  noticeAccepted.value = false
}

function myParticipationForCard(card: CarpoolCard) {
  const currentSessionId = card.session?.id
  if (!currentSessionId) return null
  return mine.value.find((item) =>
    ['paid', 'active'].includes(item.status) && (
      item.session_id === currentSessionId ||
      item.edges?.session?.id === currentSessionId
    )
  ) || null
}

function hallCardButtonLabel(card: CarpoolCard) {
  const item = myParticipationForCard(card)
  if (!item) return '立即拼车'
  const session = item.edges?.session || card.session
  if (session.status === 'full') return '拼车已成功，等待发车'
  if (session.status === 'provisioning') return '采购配置中'
  if (session.status === 'active') return '已发车'
  return `已发起拼车 · ${session.paid_count}/${session.seat_count}`
}

function onNoticeScroll(event: Event) {
  const el = event.target as HTMLElement
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 8) {
    noticeScrolled.value = true
  }
}

async function loadAll() {
  loading.value = true
  try {
    const [cardData, noticeData, myData] = await Promise.all([
      carpoolAPI.listCards(),
      carpoolAPI.currentNotice(),
      carpoolAPI.my(),
    ])
    cards.value = cardData
    notice.value = noticeData
    mine.value = myData
  } finally {
    loading.value = false
  }
}

async function loadMine() {
  mine.value = await carpoolAPI.my()
}

async function openDetail(item: CarpoolParticipant) {
  detailItem.value = item
  detailData.value = null
  detailTab.value = 'info'
  detailLoading.value = true
  try {
    detailData.value = await carpoolAPI.myDetail(item.id)
    if (detailData.value.participant) {
      detailItem.value = {
        ...item,
        ...detailData.value.participant,
        edges: {
          ...item.edges,
          ...detailData.value.participant.edges,
          session: detailData.value.session || item.edges?.session,
        },
      }
    }
  } catch (error) {
    appStore.showError('拼车详情加载失败，请稍后重试')
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detailItem.value = null
  detailData.value = null
  detailLoading.value = false
  detailTab.value = 'info'
  previewVoucher.value = null
}

async function joinSelected() {
  if (!selected.value || !notice.value) return
  joining.value = true
  try {
    const resp = await carpoolAPI.join({
      vehicle_type_id: selected.value.vehicle_type.id,
      notice_version_id: notice.value.id,
      notice_accepted: noticeAccepted.value,
      payment_type: joinForm.payment_type,
      return_url: `${window.location.origin}/payment/result`,
      payment_source: 'carpool',
    })
    selected.value = null
    launchPayment(resp.order)
  } finally {
    joining.value = false
  }
}

function launchPayment(result: CreateOrderResult) {
  const visibleMethod = normalizeVisibleMethod(joinForm.payment_type) || joinForm.payment_type
  const stripeRouteUrl = result.client_secret && visibleMethod !== 'airwallex'
    ? router.resolve({
      path: '/payment/stripe',
      query: {
        order_id: String(result.order_id),
        client_secret: result.client_secret,
        method: visibleMethod === 'wxpay' ? 'wechat_pay' : visibleMethod === 'alipay' ? 'alipay' : undefined,
        resume_token: result.resume_token || undefined,
      },
    }).href
    : ''
  const airwallexRouteUrl = result.client_secret && result.intent_id
    ? router.resolve({
      path: '/payment/airwallex',
      query: {
        order_id: String(result.order_id),
        out_trade_no: result.out_trade_no || undefined,
        resume_token: result.resume_token || undefined,
      },
    }).href
    : ''
  const decision = decidePaymentLaunch(result, {
    visibleMethod,
    orderType: 'carpool',
    isMobile: /Android|iPhone|iPad|iPod|Mobile/i.test(window.navigator.userAgent),
    isWechatBrowser: /MicroMessenger/i.test(window.navigator.userAgent),
    forceQRCode: visibleMethod === 'alipay',
    stripePopupUrl: stripeRouteUrl,
    stripeRouteUrl,
    airwallexRouteUrl,
  })
  paymentState.value = decision.paymentState
  paymentPhase.value = 'paying'

  if (decision.kind === 'qr_waiting') {
    return
  }
  if (decision.kind === 'redirect_waiting' && decision.paymentState.payUrl) {
    openPaymentWindow(decision.paymentState.payUrl)
    return
  }
  if (decision.kind === 'unhandled') {
    appStore.showError('暂时无法发起支付，请稍后重试或更换支付方式')
    resetPayment()
    return
  }
  if (decision.kind === 'stripe_route' || decision.kind === 'airwallex_route') {
    window.location.href = decision.paymentState.payUrl
    return
  }
  if (decision.kind === 'stripe_popup') {
    openPaymentWindow(decision.paymentState.payUrl)
  }
}

function openPaymentWindow(url: string) {
  const win = window.open(url, 'paymentPopup', getPaymentPopupFeatures())
  if (!win || win.closed) {
    appStore.showInfo('请点击页面中的“重新打开支付页面”继续支付')
  }
}

function resetPayment() {
  paymentPhase.value = 'select'
  loadAll()
}

function onPaymentSuccess() {
  appStore.showSuccess('支付成功，已加入拼车队列')
  activeTab.value = 'mine'
  resetPayment()
}

function onPaymentSettled(outcome: PaymentOutcome) {
  if (outcome === 'success') {
    onPaymentSuccess()
    return
  }
  if (outcome === 'cancelled') {
    appStore.showInfo('拼车支付已取消，未完成支付的记录不会进入我的拼车')
  }
  activeTab.value = 'mine'
  resetPayment()
}

function segmentLabel(vt: Pick<CarpoolVehicleType, 'product' | 'plan_tier' | 'multiplier'>) {
  const product = productLabel(vt.product)
  const tier = tierLabel(vt.plan_tier)
  const multiplier = String(vt.multiplier || '').toUpperCase()
  return [product, tier, multiplier].filter(Boolean).join(' ')
}

function productLabel(value?: string) {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    claudecode: 'ClaudeCode',
    claude_code: 'ClaudeCode',
    glm: 'GLM',
    volcengine: '火山方舟',
    volcano: '火山方舟',
    doubao: '豆包',
    qwen: '通义千问',
    custom: '自定义产品',
  }
  const normalized = normalizeCarpoolCode(value, '')
  return labels[normalized] || value?.trim() || '自定义产品'
}

function tierLabel(value?: string) {
  const labels: Record<string, string> = {
    pro: 'Pro',
    plus: 'Plus',
    standard: 'Standard',
    enterprise: 'Enterprise',
    custom: '自定义套餐',
  }
  const normalized = normalizeCarpoolCode(value, '')
  return labels[normalized] || value?.trim() || '自定义套餐'
}

function normalizeCarpoolCode(value: string | undefined, fallback: string) {
  const normalized = String(value || '').trim().toLowerCase().replace(/\s+/g, '_')
  return normalized || fallback
}

function refundLabel(method: string) {
  return method === 'gateway' ? '原路退款' : '退到余额'
}

function refundMethodsFor(item: CarpoolParticipant) {
  const methods = item.edges?.vehicle_type?.refund_methods || []
  return methods.length ? methods : ['balance', 'gateway']
}

function statusLabel(status?: string) {
  const labels: Record<string, string> = {
    recruiting: '招募中',
    full: '拼车成功',
    provisioning: '采购中',
    active: '已发车',
    ended: '已结束',
    failed: '拼车失败',
    pending_payment: '待支付',
    paid: '已支付',
    refund_pending: '待退款',
    refunded_balance: '已退余额',
    refunded_gateway: '已原路退款',
    cancelled: '已取消',
  }
  return labels[String(status || '')] || status || '-'
}

function carpoolStateBadgeClass(status?: string) {
  const value = String(status || '')
  if (['active'].includes(value)) return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200'
  if (['full', 'provisioning'].includes(value)) return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200'
  if (['refund_pending', 'refunded_balance', 'refunded_gateway', 'failed', 'cancelled'].includes(value)) return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-200'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
}

function sessionProgressLabel(item: CarpoolParticipant) {
  const session = item.edges?.session
  if (!session) {
    const vt = item.edges?.vehicle_type
    return `0/${vt?.seat_count || 0}`
  }
  return `${session.paid_count}/${session.seat_count}`
}

function participantProgress(item: CarpoolParticipant) {
  const session = item.edges?.session
  if (!session) return 0
  return Math.min(100, Math.round((session.paid_count / Math.max(session.seat_count, 1)) * 100))
}

function participantStageText(item: CarpoolParticipant) {
  const session = item.edges?.session
  if (item.status === 'refund_pending') return '当前拼车已进入退款处理，请等待管理员或支付通道完成退款。'
  if (item.status === 'refunded_balance') return '本次拼车款项已退回余额。'
  if (item.status === 'refunded_gateway') return '本次拼车款项已原路退回支付渠道。'
  if (!session) return '已完成支付，系统正在确认拼车队列。'
  const progressText = `${session.paid_count}/${session.seat_count}`
  const labels: Record<string, string> = {
    recruiting: `已支付，当前进度 ${progressText}，请等待拼团成功。`,
    full: `拼车已成功，当前进度 ${progressText}，等待管理员采购账号和静态住宅 IP。`,
    provisioning: '管理员正在采购账号、配置代理和订阅分组。完成后会在详情中展示交付信息。',
    active: '已发车，请查看详情中的沟通方式、订阅分组和交付凭证。',
    ended: '本轮拼车服务已结束，如需继续使用可重新参加新车。',
    failed: '本轮拼车失败，请关注退款状态或联系管理员。',
    cancelled: '本轮拼车已取消。',
  }
  return labels[session.status] || '拼车状态已更新，请查看详情或联系管理员确认。'
}

function canRequestRefund(item: CarpoolParticipant) {
  if (item.status !== 'paid') return false
  const sessionStatus = item.edges?.session?.status
  if (sessionStatus && sessionStatus !== 'recruiting') return false
  if (!item.wait_until) return false
  return Date.now() >= new Date(item.wait_until).getTime()
}

function openRefundDialog(item: CarpoolParticipant) {
  refundTarget.value = item
  refundForm.refund_method = refundMethodsFor(item)[0] || 'balance'
}

async function submitRefundRequest() {
  if (!refundTarget.value) return
  refundSubmitting.value = true
  try {
    await carpoolAPI.requestRefund(refundTarget.value.id, { refund_method: refundForm.refund_method })
    appStore.showSuccess('已发起退款申请，请等待管理员处理')
    refundTarget.value = null
    await Promise.all([loadMine(), loadAll()])
  } catch (error) {
    appStore.showError('发起退款失败，请确认是否已到可退款时间')
  } finally {
    refundSubmitting.value = false
  }
}

function deliverySummary(item: CarpoolParticipant) {
  const session = item.edges?.session
  if (!session) return ''
  const lines: string[] = []
  const communication = session.communication || {}
  const account = session.account_info || {}
  const proxy = session.proxy_info || {}
  const contact = [textValue(communication.type), textValue(communication.group_no), textValue(communication.link)].filter(Boolean).join(' · ')
  if (contact) lines.push(`沟通方式：${contact}`)
  if (textValue(communication.note)) lines.push(`沟通说明：${textValue(communication.note)}`)
  const accountText = [textValue(account.login_account), textValue(account.account_hint)].filter(Boolean).join(' · ')
  if (accountText) lines.push(`账号信息：${accountText}`)
  if (textValue(account.delivery_note)) lines.push(`交付说明：${textValue(account.delivery_note)}`)
  const proxyText = [textValue(proxy.provider), textValue(proxy.region), textValue(proxy.ip)].filter(Boolean).join(' · ')
  if (proxyText) lines.push(`代理/IP：${proxyText}`)
  if (textValue(proxy.expire_at)) lines.push(`代理到期：${textValue(proxy.expire_at)}`)
  return lines.join('\n')
}

function textValue(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function money(value: number) {
  return Number(value || 0).toFixed(2)
}

function compactNumber(value?: number) {
  const num = Number(value || 0)
  if (!Number.isFinite(num)) return '0'
  if (Math.abs(num) >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`
  if (Math.abs(num) >= 10_000) return `${(num / 1_000).toFixed(1)}K`
  return String(Math.round(num))
}

function windowShortLabel(window: string) {
  if (window === '5h') return '5h'
  if (window === '7d') return '7d'
  return window || '-'
}

function usagePercent(value?: number) {
  const num = Number(value || 0)
  if (!Number.isFinite(num)) return '-'
  return `${Math.round(num)}%`
}

function windowSort(window: string) {
  if (window === '5h') return 1
  if (window === '7d') return 2
  return 9
}

function usageWindowBadgeClass(window: string) {
  if (window === '7d') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200'
  return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-200'
}

function usageWindowBarClass(utilization?: number) {
  const num = Number(utilization || 0)
  if (num >= 100) return 'bg-red-500'
  if (num >= 80) return 'bg-amber-500'
  return 'bg-green-500'
}

function resetCountdown(value?: string, utilization?: number) {
  if (!value) return Number(utilization || 0) <= 0 ? '现在' : '-'
  const diffMs = new Date(value).getTime() - Date.now()
  if (!Number.isFinite(diffMs) || diffMs <= 0) {
    return Number(utilization || 0) > 0 ? '待刷新' : '现在'
  }
  const totalMinutes = Math.floor(diffMs / 60000)
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours >= 24) {
    const days = Math.floor(hours / 24)
    return `${days}d ${hours % 24}h`
  }
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function timeLeftLabel(value?: string) {
  if (!value) return '待发车'
  const diffMs = new Date(value).getTime() - Date.now()
  if (!Number.isFinite(diffMs)) return '-'
  if (diffMs <= 0) return '已到期'
  const totalHours = Math.floor(diffMs / 3600000)
  const days = Math.floor(totalHours / 24)
  const hours = totalHours % 24
  if (days > 0) return `${days}天${hours}小时`
  const minutes = Math.max(0, Math.floor((diffMs % 3600000) / 60000))
  if (hours > 0) return `${hours}小时${minutes}分钟`
  return `${minutes}分钟`
}

function formatTime(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

onMounted(loadAll)
</script>

<style scoped>
.carpool-notice-markdown :deep(*) {
  max-width: 100%;
}

.carpool-notice-markdown :deep(p) {
  margin: 0 0 0.65rem;
}

.carpool-notice-markdown :deep(p:last-child),
.carpool-notice-markdown :deep(ul:last-child),
.carpool-notice-markdown :deep(ol:last-child),
.carpool-notice-markdown :deep(blockquote:last-child) {
  margin-bottom: 0;
}

.carpool-notice-markdown :deep(ul),
.carpool-notice-markdown :deep(ol) {
  margin: 0 0 0.75rem 1.25rem;
  padding-left: 0.75rem;
}

.carpool-notice-markdown :deep(ul) {
  list-style: disc;
}

.carpool-notice-markdown :deep(ol) {
  list-style: decimal;
}

.carpool-notice-markdown :deep(li) {
  margin: 0.25rem 0;
  padding-left: 0.15rem;
}

.carpool-notice-markdown :deep(strong) {
  font-weight: 700;
  color: rgb(17 24 39);
}

.dark .carpool-notice-markdown :deep(strong) {
  color: rgb(243 244 246);
}

.carpool-notice-markdown :deep(a) {
  color: rgb(13 148 136);
  font-weight: 600;
  text-decoration: none;
}

.carpool-notice-markdown :deep(a:hover) {
  text-decoration: underline;
}

.carpool-notice-markdown :deep(blockquote) {
  margin: 0.75rem 0;
  border-left: 3px solid rgb(45 212 191);
  padding-left: 0.75rem;
  color: rgb(75 85 99);
}

.dark .carpool-notice-markdown :deep(blockquote) {
  color: rgb(209 213 219);
}
</style>
