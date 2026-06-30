<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">拼车管理</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">管理车类型、当前轮次、交付信息和用户须知。</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadAll">刷新</button>
        </div>
        <div class="mt-5 grid gap-3 sm:grid-cols-4">
          <div v-for="item in overviewCards" :key="item.label" class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</p>
            <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
          </div>
        </div>
      </section>

      <div class="flex gap-2 overflow-x-auto rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
        <button v-for="tab in tabs" :key="tab.key" type="button" class="shrink-0 rounded-md px-4 py-2 text-sm font-medium" :class="activeTab === tab.key ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-dark-300'" @click="activeTab = tab.key">
          {{ tab.label }}
        </button>
      </div>

      <section v-if="activeTab === 'management'" class="space-y-5">
        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="space-y-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">已成团车辆</h3>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">只展示已满员、采购中、已发车、已结束的轮次。点击详情查看车辆与成员用量，点击交付配置完成发车。</p>
            </div>
            <div class="grid gap-3 md:grid-cols-[160px_220px_minmax(0,1fr)_auto]">
              <select v-model="managementStatus" class="input" @change="loadManagement">
                <option value="">可管理轮次</option>
                <option value="full">已满员</option>
                <option value="provisioning">采购中</option>
                <option value="active">已发车</option>
                <option value="ended">已结束</option>
                <option value="all">全部状态</option>
              </select>
              <select v-model.number="managementVehicleTypeId" class="input">
                <option :value="0">全部车型</option>
                <option v-for="type in managementFilterTypes" :key="type.id" :value="type.id">{{ type.name }}</option>
              </select>
              <input v-model="managementKeyword" class="input" placeholder="搜索轮次、车型、成员邮箱/昵称" />
              <button type="button" class="btn btn-secondary btn-sm" @click="resetManagementFilters">重置</button>
            </div>
          </div>
          <div v-if="managementRows.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">暂无可管理的已成团轮次</div>
          <div v-else class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs font-medium text-gray-500 dark:bg-dark-800 dark:text-dark-300">
                <tr>
                  <th class="px-4 py-3">轮次</th>
                  <th class="px-4 py-3">车型</th>
                  <th class="px-4 py-3">状态</th>
                  <th class="px-4 py-3">成员</th>
                  <th class="px-4 py-3">本车用量</th>
                  <th class="px-4 py-3">订阅/沟通</th>
                  <th class="px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="row in managementRows" :key="row.session.id" class="align-top">
                  <td class="px-4 py-4">
                    <p class="font-medium text-gray-900 dark:text-white">{{ row.session.session_no || row.session.id }}</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">满员：{{ formatTime(row.session.filled_at || row.session.created_at) }}</p>
                  </td>
                  <td class="px-4 py-4">
                    <p class="font-medium text-gray-900 dark:text-white">{{ row.session.edges?.vehicle_type?.name || row.session.vehicle_type_id }}</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ row.session.edges?.vehicle_type ? segmentLabel(row.session.edges.vehicle_type) : '-' }}</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ sessionProgressLabel(row.session) }} 人</p>
                  </td>
                  <td class="px-4 py-4">
                    <span class="rounded bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ statusLabel(row.session.status) }}</span>
                  </td>
                  <td class="px-4 py-4">
                    <p class="font-medium text-gray-900 dark:text-white">{{ row.participants.length }} 人</p>
                    <p class="mt-1 max-w-56 truncate text-xs text-gray-500 dark:text-dark-400">{{ memberSummary(row) }}</p>
                  </td>
                  <td class="px-4 py-4">
                    <p class="font-medium text-gray-900 dark:text-white">{{ row.usage.request_count }} 次</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ compactNumber(row.usage.total_tokens) }} tokens</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">用户扣费 ${{ money(row.usage.total_actual_cost) }}</p>
                  </td>
                  <td class="min-w-52 px-4 py-4 text-xs text-gray-500 dark:text-dark-400">
                    <p>分组：{{ stringField(row.session.account_info?.subscription_group_name, '未分配') }}</p>
                    <p class="mt-1">沟通：{{ communicationLabel(row.session.communication) }}</p>
                    <p class="mt-1">凭证：{{ row.session.edges?.vouchers?.length || 0 }} 个</p>
                  </td>
                  <td class="px-4 py-4 text-right">
                    <div class="flex justify-end gap-2">
                      <button type="button" class="btn btn-secondary btn-sm" @click="managementDetail = row">详情</button>
                      <button type="button" class="btn btn-secondary btn-sm" @click="openSessionEditor(row.session)">交付配置</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section v-if="activeTab === 'progress'" class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">拼车情况</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">展示当前每种车类型正在排队的招募轮次。满员后会自动进入“拼车管理”，并自动创建下一轮空车。</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadProgressSessions">刷新</button>
        </div>
        <div v-if="progressSessions.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">暂无招募中的拼车轮次</div>
        <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <article v-for="session in progressSessions" :key="session.id" class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
            <div class="mb-3 flex items-start justify-between gap-3">
              <div>
                <h4 class="font-semibold text-gray-900 dark:text-white">{{ session.edges?.vehicle_type?.name || session.vehicle_type_id }}</h4>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ session.session_no || session.id }} · {{ session.edges?.vehicle_type ? segmentLabel(session.edges.vehicle_type) : '拼车轮次' }}</p>
              </div>
              <span class="rounded bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">招募中</span>
            </div>
            <div class="mb-2 flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-dark-400">进度</span>
              <span class="font-semibold text-gray-900 dark:text-white">{{ sessionProgressLabel(session) }}</span>
            </div>
            <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
              <div class="h-full rounded-full bg-amber-500" :style="{ width: sessionProgressWidth(session) }"></div>
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-2 text-xs">
              <span class="rounded bg-emerald-50 px-2 py-1 font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">真实拼成 {{ realCompletedCount(session.vehicle_type_id) }}</span>
              <span class="text-gray-500 dark:text-dark-400">创建时间：{{ formatTime(session.created_at) }}</span>
            </div>
            <div v-if="session.edges?.participants?.length" class="mt-4 space-y-2">
              <div v-for="participant in session.edges.participants" :key="participant.id" class="flex items-center justify-between rounded bg-gray-50 px-3 py-2 text-xs dark:bg-dark-800">
                <span class="truncate text-gray-700 dark:text-dark-200">{{ participant.edges?.user?.email || `用户 ${participant.user_id}` }}</span>
                <span class="ml-2 shrink-0 text-gray-500 dark:text-dark-400">{{ statusLabel(participant.status) }}</span>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section v-if="activeTab === 'types'" class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">车类型</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">按产品、套餐和倍率区分展示，用户端会按这些字段分组。</p>
          </div>
          <button type="button" class="btn btn-primary btn-sm" @click="openTypeEditor()">新增</button>
        </div>
        <div v-if="types.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">暂无车类型</div>
        <div v-else class="space-y-6">
          <div v-for="group in groupedTypes" :key="group.key" class="space-y-3">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ group.label }}</h4>
            <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              <article v-for="item in group.items" :key="item.id" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="mb-2 flex flex-wrap gap-2">
                      <span class="rounded bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ segmentLabel(item) }}</span>
                      <span class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ item.seat_count }} 人车</span>
                      <span :class="item.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200' : 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-300'" class="rounded px-2 py-1 text-xs">
                        {{ item.enabled ? '已启用' : '未启用' }}
                      </span>
                    </div>
                    <h5 class="text-lg font-semibold text-gray-900 dark:text-white">{{ item.name }}</h5>
                    <p class="mt-1 line-clamp-2 min-h-10 text-sm text-gray-500 dark:text-dark-400">{{ item.description || '暂无描述' }}</p>
                  </div>
                </div>
                <div class="mt-4 grid grid-cols-3 gap-3 text-sm">
                  <div>
                    <p class="text-xs text-gray-400">总价</p>
                    <p class="font-semibold text-gray-900 dark:text-white">¥{{ money(item.total_price) }}</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-400">人均</p>
                    <p class="font-semibold text-emerald-600 dark:text-emerald-400">¥{{ money(item.unit_price) }}</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-400">周期</p>
                    <p class="font-semibold text-gray-900 dark:text-white">{{ item.service_days }}天</p>
                  </div>
                </div>
                <div class="mt-3 grid grid-cols-2 gap-3 text-sm">
                  <div class="rounded bg-gray-50 p-2 dark:bg-dark-800">
                    <p class="text-xs text-gray-400">可申请退款</p>
                    <p class="font-semibold text-gray-900 dark:text-white">{{ item.refund_wait_hours || 2 }} 小时后</p>
                  </div>
                  <div class="rounded bg-gray-50 p-2 dark:bg-dark-800">
                    <p class="text-xs text-gray-400">展示基数</p>
                    <p class="font-semibold text-gray-900 dark:text-white">{{ item.completed_base_count || 0 }}</p>
                  </div>
                </div>
                <div class="mt-4 flex items-center justify-between">
                  <div class="flex flex-wrap gap-2 text-xs">
                    <span v-if="item.require_static_ip" class="rounded bg-blue-50 px-2 py-1 text-blue-700 dark:bg-blue-900/30 dark:text-blue-200">静态 IP</span>
                    <span v-if="item.support_revenue_pool" class="rounded bg-amber-50 px-2 py-1 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">中转投入</span>
                  </div>
                  <div class="flex gap-2">
                    <button type="button" class="btn btn-secondary btn-sm" @click="openTypeEditor(item)">编辑</button>
                    <button type="button" class="btn btn-secondary btn-sm" @click="deleteType(item.id)">删除</button>
                  </div>
                </div>
              </article>
            </div>
          </div>
        </div>
      </section>

      <section v-if="activeTab === 'sessions'" class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">发车交付</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">这里用于查看各排队轮次并进入交付配置；已满员车辆优先在“拼车管理”处理。</p>
          </div>
          <select v-model="sessionStatus" class="input w-40" @change="loadSessions">
            <option value="">全部</option>
            <option value="recruiting">招募中</option>
            <option value="full">已满员</option>
            <option value="provisioning">采购中</option>
            <option value="active">已发车</option>
          </select>
        </div>
        <div class="space-y-3">
          <article v-for="session in sessions" :key="session.id" class="rounded-md border border-gray-100 p-4 dark:border-dark-700">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
              <div>
                <h4 class="font-semibold text-gray-900 dark:text-white">{{ session.edges?.vehicle_type?.name || session.vehicle_type_id }} · {{ session.session_no || session.id }}</h4>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ session.edges?.vehicle_type ? segmentLabel(session.edges.vehicle_type) : '拼车轮次' }} · 状态：{{ statusLabel(session.status) }} · {{ session.paid_count }}/{{ session.seat_count }}</p>
                <p class="mt-1 text-xs text-gray-400">创建时间：{{ formatTime(session.created_at) }}</p>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" @click="openSessionEditor(session)">交付配置</button>
            </div>
          </article>
        </div>
      </section>

      <section v-if="activeTab === 'notice'" class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">用户须知</h3>
        <div class="mt-4 grid gap-4 lg:grid-cols-2">
          <div class="space-y-3">
            <input v-model="noticeForm.title" class="input" placeholder="标题" />
            <textarea v-model="noticeForm.content_md" class="input min-h-80 font-mono text-sm" placeholder="Markdown 内容"></textarea>
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
              <input v-model="noticeForm.active" type="checkbox" />
              发布后设为当前生效版本
            </label>
            <button type="button" class="btn btn-primary" @click="saveNotice">发布新版本</button>
          </div>
          <div class="rounded-md border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
            <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">历史版本</h4>
            <div class="space-y-2">
              <div v-for="item in notices" :key="item.id" class="rounded bg-white p-3 text-sm dark:bg-dark-900">
                <div class="flex justify-between">
                  <span class="font-medium text-gray-900 dark:text-white">v{{ item.version }} {{ item.title }}</span>
                  <span v-if="item.active" class="text-emerald-600">生效中</span>
                </div>
                <p class="mt-1 line-clamp-2 text-gray-500 dark:text-dark-400">{{ item.content_md }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section v-if="activeTab === 'stats'" class="space-y-5">
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
          <div v-for="item in managementCards" :key="item.label" class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
          </div>
        </div>

        <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
          <div class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">车型分布</h3>
            <div v-if="managementSummary.by_segment.length === 0" class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无已成团数据</div>
            <div v-else class="space-y-4">
              <div v-for="item in managementSummary.by_segment" :key="item.label">
                <div class="mb-1 flex items-center justify-between gap-3 text-xs">
                  <span class="font-medium text-gray-700 dark:text-dark-200">{{ item.label }}</span>
                  <span class="text-gray-500 dark:text-dark-400">{{ item.sessions }} 车 · {{ item.paid_members }} 人 · ¥{{ money(item.amount) }}</span>
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
                  <div class="h-full rounded-full bg-primary-500" :style="{ width: `${Math.max(6, (Number(item.paid_members || 0) / maxSegmentMembers) * 100)}%` }"></div>
                </div>
              </div>
            </div>
          </div>

          <div class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">状态统计</h3>
            <div class="space-y-2 text-sm">
              <div v-for="status in ['full', 'provisioning', 'active', 'ended']" :key="status" class="flex items-center justify-between rounded bg-gray-50 px-3 py-2 dark:bg-dark-800">
                <span class="text-gray-600 dark:text-dark-300">{{ statusLabel(status) }}</span>
                <span class="font-semibold text-gray-900 dark:text-white">{{ managementSummary.by_status[status] || 0 }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div v-if="managementDetail" class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
      <div class="max-h-[94vh] w-full max-w-5xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
        <div class="flex items-start justify-between gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">车辆与成员用量详情</h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ managementDetail.session.edges?.vehicle_type?.name || managementDetail.session.vehicle_type_id }}
              · {{ managementDetail.session.session_no || managementDetail.session.id }}
              · {{ statusLabel(managementDetail.session.status) }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="managementDetail = null">关闭</button>
        </div>
        <div class="max-h-[76vh] space-y-5 overflow-y-auto p-5">
          <section class="grid gap-3 text-sm sm:grid-cols-2 xl:grid-cols-5">
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-dark-400">成员进度</p>
              <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ managementDetail.session.paid_count }}/{{ managementDetail.session.seat_count }}</p>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-dark-400">拼车收入</p>
              <p class="mt-1 font-semibold text-gray-900 dark:text-white">¥{{ money(sessionPaidAmount(managementDetail)) }}</p>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-dark-400">请求次数</p>
              <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ managementDetail.usage.request_count }} 次</p>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-dark-400">Tokens</p>
              <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ compactNumber(managementDetail.usage.total_tokens) }}</p>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-dark-400">用户扣费</p>
              <p class="mt-1 font-semibold text-gray-900 dark:text-white">${{ money(managementDetail.usage.total_actual_cost) }}</p>
            </div>
          </section>

          <section class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_300px]">
            <div class="rounded-lg border border-gray-100 dark:border-dark-700">
              <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                <h4 class="text-sm font-semibold text-gray-900 dark:text-white">成员用量</h4>
              </div>
              <div class="divide-y divide-gray-100 dark:divide-dark-700">
                <div v-for="member in managementDetail.participants" :key="member.participant.id" class="grid gap-3 px-4 py-3 text-sm md:grid-cols-[minmax(0,1.4fr)_repeat(4,minmax(0,0.8fr))]">
                  <div class="min-w-0">
                    <p class="truncate font-medium text-gray-900 dark:text-white">{{ member.user?.username || member.user?.email || `用户 ${member.participant.user_id}` }}</p>
                    <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ member.user?.email || `ID ${member.participant.user_id}` }} · {{ statusLabel(member.participant.status) }}</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-400">支付</p>
                    <p class="font-medium text-gray-900 dark:text-white">¥{{ money(member.participant.amount) }}</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-400">请求</p>
                    <p class="font-medium text-gray-900 dark:text-white">{{ member.usage.request_count }}</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-400">Tokens</p>
                    <p class="font-medium text-gray-900 dark:text-white">{{ compactNumber(member.usage.total_tokens) }}</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-400">扣费</p>
                    <p class="font-medium text-gray-900 dark:text-white">${{ money(member.usage.total_actual_cost) }}</p>
                  </div>
                </div>
              </div>
            </div>

            <aside class="space-y-4">
              <div class="rounded-lg border border-gray-100 p-4 text-sm dark:border-dark-700">
                <h4 class="mb-3 font-semibold text-gray-900 dark:text-white">订阅与沟通</h4>
                <p class="text-gray-500 dark:text-dark-400">订阅分组：{{ stringField(managementDetail.session.account_info?.subscription_group_name, '未分配') }}</p>
                <p class="mt-2 text-gray-500 dark:text-dark-400">有效期：{{ stringField(managementDetail.session.account_info?.subscription_validity_days, '-') }} 天</p>
                <p class="mt-2 text-gray-500 dark:text-dark-400">沟通：{{ communicationLabel(managementDetail.session.communication) }}</p>
                <p v-if="stringField(managementDetail.session.communication?.note)" class="mt-2 whitespace-pre-line text-gray-500 dark:text-dark-400">{{ stringField(managementDetail.session.communication?.note) }}</p>
              </div>
              <div class="rounded-lg border border-gray-100 p-4 text-sm dark:border-dark-700">
                <h4 class="mb-3 font-semibold text-gray-900 dark:text-white">交付凭证</h4>
                <div v-if="!managementDetail.session.edges?.vouchers?.length" class="text-gray-500 dark:text-dark-400">暂无凭证</div>
                <div v-else class="space-y-3">
                  <button
                    v-for="voucher in managementDetail.session.edges.vouchers"
                    :key="voucher.id"
                    type="button"
                    class="block w-full overflow-hidden rounded-md border border-gray-100 text-left transition hover:border-primary-300 hover:shadow-sm dark:border-dark-700"
                    @click="previewVoucher = voucher"
                  >
                    <img :src="voucher.file_url" :alt="voucher.file_name" class="h-28 w-full object-cover" />
                    <div class="p-3">
                      <p class="truncate font-medium text-gray-900 dark:text-white">{{ voucher.file_name }}</p>
                      <p v-if="voucher.description" class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-dark-400">{{ voucher.description }}</p>
                    </div>
                  </button>
                </div>
              </div>
            </aside>
          </section>
        </div>
      </div>
    </div>

    <div v-if="editingType" class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
      <div class="max-h-[90vh] w-full max-w-3xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ editingType.id ? '编辑车类型' : '新增车类型' }}</h3>
        </div>
        <div class="max-h-[70vh] space-y-4 overflow-y-auto p-5">
          <div class="grid gap-3 md:grid-cols-3">
            <label class="space-y-1">
              <span class="text-sm text-gray-500">产品</span>
              <input v-model="editingType.product" list="carpool-product-options" class="input" placeholder="openai / claudecode / glm / volcengine" />
              <datalist id="carpool-product-options">
                <option v-for="option in carpoolProductOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </datalist>
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">套餐</span>
              <input v-model="editingType.plan_tier" list="carpool-tier-options" class="input" placeholder="pro / plus / standard / enterprise" />
              <datalist id="carpool-tier-options">
                <option v-for="option in carpoolTierOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </datalist>
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">倍率</span>
              <input v-model="editingType.multiplier" class="input" placeholder="20x / 5x / 1x" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="space-y-1">
              <span class="text-sm text-gray-500">名称</span>
              <input v-model="editingType.name" class="input" />
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">人数</span>
              <input v-model.number="editingType.seat_count" type="number" min="1" class="input" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <label class="space-y-1">
              <span class="text-sm text-gray-500">总价</span>
              <input v-model.number="editingType.total_price" type="number" min="0" class="input" />
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">人均</span>
              <input v-model.number="editingType.unit_price" type="number" min="0" class="input" />
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">服务天数</span>
              <input v-model.number="editingType.service_days" type="number" min="1" class="input" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="space-y-1">
              <span class="text-sm text-gray-500">可申请退款时间（小时）</span>
              <input v-model.number="editingType.refund_wait_hours" type="number" min="1" class="input" />
              <span class="text-xs text-gray-400">用户支付后达到该时间，才可在“我的拼车”主动发起退款。</span>
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">已拼成展示基数</span>
              <input v-model.number="editingType.completed_base_count" type="number" min="0" class="input" />
              <span class="text-xs text-gray-400">用户端展示为：基数 + 真实拼成数量。</span>
            </label>
          </div>
          <label class="space-y-1">
            <span class="text-sm text-gray-500">描述</span>
            <textarea v-model="editingType.description" class="input min-h-24" />
          </label>
          <div class="grid gap-3 md:grid-cols-3">
            <label class="space-y-1">
              <span class="text-sm text-gray-500">排序</span>
              <input v-model.number="editingType.sort_order" type="number" class="input" />
            </label>
            <label class="space-y-1">
              <span class="text-sm text-gray-500">退款方式，逗号分隔</span>
              <input v-model="editingTypeRefundMethods" class="input" />
            </label>
          </div>
          <div class="flex flex-wrap gap-4 text-sm text-gray-700 dark:text-dark-200">
            <label class="flex items-center gap-2"><input v-model="editingType.enabled" type="checkbox" />启用</label>
            <label class="flex items-center gap-2"><input v-model="editingType.require_static_ip" type="checkbox" />静态住宅 IP</label>
            <label class="flex items-center gap-2"><input v-model="editingType.support_revenue_pool" type="checkbox" />预留中转投入</label>
          </div>
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" @click="editingType = null">取消</button>
          <button type="button" class="btn btn-primary" @click="saveEditingType">保存</button>
        </div>
      </div>
    </div>

    <div v-if="editingSession" class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
      <div class="max-h-[94vh] w-full max-w-5xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ editingSession.status === 'active' ? '编辑交付配置' : '发车配置' }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">选择订阅分组给成员开通权限；这个分组绑定的账号会作为本车账号池，用于展示整车 5h / 7d 用量。账号和代理/IP 仍在系统账号管理与分组里维护。</p>
        </div>
        <div class="max-h-[74vh] overflow-y-auto p-5">
          <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div class="space-y-5">
              <section class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
                <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">车内沟通</h4>
                <div class="grid gap-3 md:grid-cols-3">
                  <label class="space-y-1">
                    <span class="text-sm text-gray-500">沟通类型</span>
                    <select v-model="editingProvision.communication_type" class="input">
                      <option value="system_chat">系统内聊天</option>
                      <option value="qq">QQ群</option>
                      <option value="wechat">微信群</option>
                      <option value="other">其他</option>
                    </select>
                  </label>
                  <label class="space-y-1">
                    <span class="text-sm text-gray-500">群号/名称</span>
                    <input v-model="editingProvision.communication_group_no" class="input" placeholder="例如 1047258111" />
                  </label>
                  <label class="space-y-1">
                    <span class="text-sm text-gray-500">群链接</span>
                    <input v-model="editingProvision.communication_link" class="input" placeholder="https://..." />
                  </label>
                </div>
                <label class="mt-3 block space-y-1">
                  <span class="text-sm text-gray-500">沟通说明</span>
                  <textarea v-model="editingProvision.communication_note" class="input min-h-20" placeholder="例如：进群后请备注站内昵称，账号发车后统一同步使用规则。"></textarea>
                </label>
              </section>

              <section class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
                <div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <h4 class="text-sm font-semibold text-gray-900 dark:text-white">订阅分配</h4>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">选择订阅分组即可。这个分组绑定的账号会作为本车账号池，并用于用户端展示整体 5h / 7d 用量。</p>
                  </div>
                  <button type="button" class="btn btn-secondary btn-sm" @click="loadGroups">刷新分组</button>
                </div>
                <div class="grid gap-3 md:grid-cols-2">
                  <label class="space-y-1">
                    <span class="text-sm text-gray-500">订阅分组</span>
                    <Select
                      v-model="assignmentForm.group_id"
                      :options="subscriptionGroupOptions"
                      placeholder="请选择订阅分组"
                      searchable
                      search-placeholder="搜索分组名称 / 平台"
                      empty-text="暂无可用订阅分组"
                    >
                      <template #selected="{ option }">
                        <span v-if="option" class="flex min-w-0 items-center gap-2">
                          <span class="truncate">{{ option.name }}</span>
                          <span class="shrink-0 rounded bg-primary-50 px-1.5 py-0.5 text-[11px] text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ option.platformLabel }}</span>
                          <span class="shrink-0 text-xs text-gray-500 dark:text-dark-300">{{ option.rate }}x</span>
                        </span>
                        <span v-else class="text-gray-400 dark:text-dark-400">请选择订阅分组</span>
                      </template>
                      <template #option="{ option, selected }">
                        <div class="min-w-0 flex-1">
                          <div class="flex min-w-0 items-center gap-2">
                            <span class="truncate font-medium">{{ option.name }}</span>
                            <span class="shrink-0 rounded bg-primary-50 px-1.5 py-0.5 text-[11px] text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ option.platformLabel }}</span>
                            <span class="shrink-0 rounded bg-emerald-50 px-1.5 py-0.5 text-[11px] text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">{{ option.rate }}x</span>
                          </div>
                          <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ option.summary }}</p>
                        </div>
                        <Icon v-if="selected" name="check" size="sm" class="shrink-0 text-primary-500" :stroke-width="2" />
                      </template>
                    </Select>
                  </label>
                  <label class="space-y-1">
                    <span class="text-sm text-gray-500">有效期（天）</span>
                    <input v-model.number="assignmentForm.validity_days" type="number" min="1" max="36500" class="input" />
                  </label>
                </div>
                <div v-if="selectedSubscriptionGroup" class="mt-4 rounded-md bg-gray-50 p-4 text-sm dark:bg-dark-800">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-semibold text-gray-900 dark:text-white">{{ selectedSubscriptionGroup.name }}</span>
                    <span class="rounded bg-primary-50 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ platformLabel(selectedSubscriptionGroup.platform) }}</span>
                    <span class="rounded bg-emerald-50 px-2 py-0.5 text-xs text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">{{ selectedSubscriptionGroup.rate_multiplier }}x 倍率</span>
                    <span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">账号 {{ selectedSubscriptionGroup.active_account_count ?? selectedSubscriptionGroup.account_count ?? 0 }}</span>
                  </div>
                  <p v-if="selectedSubscriptionGroup.description" class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ selectedSubscriptionGroup.description }}</p>
                  <div class="mt-3 grid gap-2 text-xs text-gray-500 dark:text-dark-300 sm:grid-cols-3">
                    <span>日限：{{ limitLabel(selectedSubscriptionGroup.daily_limit_usd) }}</span>
                    <span>周限：{{ limitLabel(selectedSubscriptionGroup.weekly_limit_usd) }}</span>
                    <span>月限：{{ limitLabel(selectedSubscriptionGroup.monthly_limit_usd) }}</span>
                  </div>
                  <div class="mt-4 rounded-md border border-gray-100 bg-white dark:border-dark-700 dark:bg-dark-900">
                    <div class="flex items-center justify-between border-b border-gray-100 px-3 py-2 dark:border-dark-700">
                      <span class="text-xs font-semibold text-gray-700 dark:text-dark-200">绑定账号</span>
                      <span class="text-xs text-gray-500 dark:text-dark-400">{{ groupAccountLoading ? '加载中...' : `${selectedGroupAccounts.length} 个` }}</span>
                    </div>
                    <div v-if="groupAccountLoading" class="p-3 text-xs text-gray-500 dark:text-dark-400">正在读取该订阅分组绑定账号...</div>
                    <div v-else-if="selectedGroupAccounts.length === 0" class="p-3 text-xs leading-5 text-amber-600 dark:text-amber-200">该订阅分组暂未绑定账号，用户端账号整体用量会显示为空。请先在账号管理中把账号加入这个订阅分组。</div>
                    <div v-else class="max-h-44 divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
                      <div v-for="account in selectedGroupAccounts" :key="account.id" class="flex items-center justify-between gap-3 px-3 py-2 text-xs">
                        <div class="min-w-0">
                          <p class="truncate font-medium text-gray-900 dark:text-white">{{ account.name || `账号 ${account.id}` }}</p>
                          <p class="truncate text-gray-500 dark:text-dark-400">{{ platformLabel(account.platform) }} · {{ account.type }} · 并发 {{ account.concurrency }}</p>
                        </div>
                        <span :class="account.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200' : 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-300'" class="shrink-0 rounded px-2 py-1">{{ accountStatusLabel(account.status) }}</span>
                      </div>
                    </div>
                  </div>
                </div>
                <div v-else-if="subscriptionGroups.length === 0" class="mt-4 rounded-md bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                  暂无可用订阅分组，请先到“分组管理”创建订阅类型分组，并确认分组已启用。
                </div>

                <div class="mt-4 rounded-md border border-gray-100 dark:border-dark-700">
                  <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                    <span class="text-sm font-semibold text-gray-900 dark:text-white">将分配成员</span>
                    <span class="text-xs text-gray-500 dark:text-dark-400">{{ assignableParticipants.length }} 人</span>
                  </div>
                  <div v-if="assignableParticipants.length === 0" class="p-4 text-sm text-gray-500 dark:text-dark-400">当前轮次没有可分配的已付款成员。</div>
                  <div v-else class="max-h-44 divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
                    <div v-for="participant in assignableParticipants" :key="participant.id" class="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                      <div class="min-w-0">
                        <p class="truncate font-medium text-gray-900 dark:text-white">{{ participant.edges?.user?.username || participant.edges?.user?.email || `用户 ${participant.user_id}` }}</p>
                        <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ participant.edges?.user?.email || `ID ${participant.user_id}` }}</p>
                      </div>
                      <span class="shrink-0 rounded bg-emerald-50 px-2 py-1 text-xs text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">{{ statusLabel(participant.status) }}</span>
                    </div>
                  </div>
                </div>

                <label class="mt-3 block space-y-1">
                  <span class="text-sm text-gray-500">分配备注</span>
                  <textarea v-model="assignmentForm.notes" class="input min-h-20" placeholder="可记录本次分配说明，会写入订阅备注。"></textarea>
                </label>
              </section>

              <label class="block space-y-1">
                <span class="text-sm text-gray-500">管理员备注</span>
                <textarea v-model="editingProvision.admin_notes" class="input min-h-20" placeholder="仅后台管理使用，可记录采购进度、内部处理备注。"></textarea>
              </label>
            </div>

            <aside class="space-y-4 rounded-lg border border-gray-100 p-4 dark:border-dark-700">
              <div>
                <h4 class="text-sm font-semibold text-gray-900 dark:text-white">新增交付凭证</h4>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">支持上传实际购买截图，保存后会展示给该车成员。</p>
              </div>
              <ImageUpload
                v-model="editingProvision.voucher_file_url"
                mode="image"
                size="md"
                upload-label="上传图片"
                remove-label="移除"
                hint="建议上传购买/下单截图，单张不超过 600KB。"
                :max-size="600 * 1024"
                paste-enabled
              />
              <label class="block space-y-1">
                <span class="text-sm text-gray-500">凭证名称</span>
                <input v-model="editingProvision.voucher_file_name" class="input" placeholder="例如 OpenAI Pro 采购截图" />
              </label>
              <label class="block space-y-1">
                <span class="text-sm text-gray-500">凭证说明</span>
                <textarea v-model="editingProvision.voucher_description" class="input min-h-20" placeholder="例如：已完成账号和静态住宅 IP 购买。"></textarea>
              </label>

              <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
                <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">已有凭证</h4>
                <div v-if="sessionVouchers.length === 0" class="rounded-md bg-gray-50 p-4 text-center text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">暂无凭证</div>
                <div v-else class="space-y-3">
                  <div v-for="voucher in sessionVouchers" :key="voucher.id" class="overflow-hidden rounded-md border border-gray-100 dark:border-dark-700">
                    <button type="button" class="block w-full text-left" @click="previewVoucher = voucher">
                      <img :src="voucher.file_url" :alt="voucher.file_name" class="h-32 w-full object-cover" />
                    </button>
                    <div class="space-y-2 p-3">
                      <div class="flex items-start justify-between gap-2">
                        <div class="min-w-0">
                          <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ voucher.file_name }}</p>
                          <p class="text-xs text-gray-400">{{ formatTime(voucher.created_at) }}</p>
                        </div>
                        <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="deleteVoucher(voucher.id)">删除</button>
                      </div>
                      <p v-if="voucher.description" class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ voucher.description }}</p>
                    </div>
                  </div>
                </div>
              </div>
            </aside>
          </div>
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" @click="editingSession = null">取消</button>
          <button type="button" class="btn btn-primary" :disabled="assigning" @click="saveProvision">
            {{ assigning ? '处理中...' : editingSession.status === 'active' ? '保存交付配置' : '分配订阅并发车' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="previewVoucher" class="fixed inset-0 z-[70] flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm" @click.self="previewVoucher = null">
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
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { adminCarpoolAPI } from '@/api/admin/carpool'
import type { Account, AdminGroup } from '@/types'
import type {
  CarpoolAdminManagementResponse,
  CarpoolAdminSessionRow,
  CarpoolNoticeVersion,
  CarpoolSession,
  CarpoolVehicleType,
  CarpoolVoucher,
} from '@/types/carpool'
import { useAppStore } from '@/stores'

const appStore = useAppStore()
const activeTab = ref<'management' | 'progress' | 'types' | 'sessions' | 'notice' | 'stats'>('management')
const tabs = [
  { key: 'management', label: '拼车管理' },
  { key: 'progress', label: '拼车情况' },
  { key: 'types', label: '车类型' },
  { key: 'sessions', label: '发车交付' },
  { key: 'notice', label: '用户须知' },
  { key: 'stats', label: '数据统计' },
] as const
const overview = ref<Record<string, any>>({})
const management = ref<CarpoolAdminManagementResponse | null>(null)
const types = ref<CarpoolVehicleType[]>([])
const sessions = ref<CarpoolSession[]>([])
const progressSessionList = ref<CarpoolSession[]>([])
const notices = ref<CarpoolNoticeVersion[]>([])
const groups = ref<AdminGroup[]>([])
const sessionStatus = ref('full')
const managementStatus = ref('')
const managementVehicleTypeId = ref(0)
const managementKeyword = ref('')
const managementDetail = ref<CarpoolAdminSessionRow | null>(null)
const editingType = ref<CarpoolVehicleType | null>(null)
const editingTypeRefundMethods = ref('balance,gateway')
const editingSession = ref<CarpoolSession | null>(null)
const sessionVouchers = ref<CarpoolVoucher[]>([])
const previewVoucher = ref<CarpoolVoucher | null>(null)
const selectedGroupAccounts = ref<Account[]>([])
const groupAccountLoading = ref(false)
const assigning = ref(false)
const editingProvision = reactive({
  status: 'provisioning',
  communication_type: 'system_chat',
  communication_group_no: '',
  communication_link: '',
  communication_note: '',
  admin_notes: '',
  voucher_file_url: '',
  voucher_file_name: '',
  voucher_description: '',
})
const assignmentForm = reactive({
  group_id: 0,
  account_pool_group_id: 0,
  validity_days: 30,
  notes: '',
})
const noticeForm = reactive({
  title: '拼车用户须知',
  content_md: '',
  active: true,
})
const carpoolProductOptions = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'claudecode', label: 'ClaudeCode' },
  { value: 'glm', label: 'GLM' },
  { value: 'volcengine', label: '火山方舟' },
  { value: 'doubao', label: '豆包' },
  { value: 'qwen', label: '通义千问' },
]
const carpoolTierOptions = [
  { value: 'pro', label: 'Pro' },
  { value: 'plus', label: 'Plus' },
  { value: 'standard', label: 'Standard' },
  { value: 'enterprise', label: 'Enterprise' },
]

const overviewCards = computed(() => {
  const sessionsMap = (overview.value.sessions || {}) as Record<string, number>
  return [
    { label: '招募中', value: sessionsMap.recruiting || 0 },
    { label: '已满员', value: sessionsMap.full || 0 },
    { label: '已发车', value: sessionsMap.active || 0 },
    { label: '待退款', value: overview.value.refund_pending || 0 },
  ]
})

const groupedTypes = computed(() => {
  const map = new Map<string, { key: string; label: string; items: CarpoolVehicleType[] }>()
  for (const item of types.value) {
    const key = `${normalizeCarpoolCode(item.product, 'custom')}:${normalizeCarpoolCode(item.plan_tier, 'custom')}:${normalizeCarpoolCode(item.multiplier, 'custom')}`
    if (!map.has(key)) map.set(key, { key, label: segmentLabel(item), items: [] })
    map.get(key)!.items.push(item)
  }
  return Array.from(map.values())
})

const subscriptionGroups = computed(() =>
  groups.value.filter((group) => group.status === 'active' && group.subscription_type === 'subscription')
)

const selectedSubscriptionGroup = computed(() =>
  subscriptionGroups.value.find((group) => group.id === assignmentForm.group_id) || null
)

const subscriptionGroupOptions = computed(() =>
  subscriptionGroups.value.map((group) => {
    const limits = [
      `日 ${limitLabel(group.daily_limit_usd)}`,
      `周 ${limitLabel(group.weekly_limit_usd)}`,
      `月 ${limitLabel(group.monthly_limit_usd)}`,
    ].join(' / ')
    return {
      value: group.id,
      label: `${group.name} ${platformLabel(group.platform)} ${group.rate_multiplier}x ${limits}`,
      name: group.name,
      platformLabel: platformLabel(group.platform),
      rate: group.rate_multiplier,
      summary: `${limits} · 账号 ${group.active_account_count ?? group.account_count ?? 0}`,
      description: [group.description || '', platformLabel(group.platform), limits].join(' '),
    }
  })
)

const assignableParticipants = computed(() => {
  const participants = editingSession.value?.edges?.participants || []
  return participants.filter((participant) => ['paid', 'active'].includes(participant.status))
})

const managementSummary = computed(() => management.value?.summary || {
  completed_sessions: 0,
  paid_members: 0,
  active_members: 0,
  total_paid_amount: 0,
  total_tokens: 0,
  total_actual_cost: 0,
  by_status: {},
  by_segment: [],
})

const managementRows = computed(() => {
  const rows = management.value?.items || []
  const keyword = managementKeyword.value.trim().toLowerCase()
  return rows.filter((row) => {
    if (managementVehicleTypeId.value && row.session.vehicle_type_id !== managementVehicleTypeId.value) return false
    if (!keyword) return true
    const haystack = [
      row.session.session_no,
      row.session.edges?.vehicle_type?.name,
      row.session.edges?.vehicle_type ? segmentLabel(row.session.edges.vehicle_type) : '',
      row.participants.map((member) => [member.user?.username, member.user?.email, member.participant.user_id].filter(Boolean).join(' ')).join(' '),
    ].filter(Boolean).join(' ').toLowerCase()
    return haystack.includes(keyword)
  })
})

const managementFilterTypes = computed(() => {
  const map = new Map<number, CarpoolVehicleType>()
  for (const row of management.value?.items || []) {
    const type = row.session.edges?.vehicle_type
    if (type) map.set(type.id, type)
  }
  return Array.from(map.values()).sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
})

const progressSessions = computed(() =>
  progressSessionList.value.filter((session) => session.status === 'recruiting')
)

const managementCards = computed(() => [
  { label: '已成团轮次', value: managementSummary.value.completed_sessions },
  { label: '已付款成员', value: managementSummary.value.paid_members },
  { label: '已分配成员', value: managementSummary.value.active_members },
  { label: '拼车收入', value: `¥${money(managementSummary.value.total_paid_amount)}` },
  { label: '用户消耗', value: `$${money(managementSummary.value.total_actual_cost)}` },
  { label: '总 Tokens', value: compactNumber(managementSummary.value.total_tokens) },
])

const maxSegmentMembers = computed(() =>
  Math.max(1, ...managementSummary.value.by_segment.map((item) => Number(item.paid_members || 0)))
)

async function loadAll() {
  const [overviewData, typeData, noticeData, managementData] = await Promise.all([
    adminCarpoolAPI.overview(),
    adminCarpoolAPI.listTypes(),
    adminCarpoolAPI.listNotices(),
    adminCarpoolAPI.management({ page: 1, page_size: 50, status: managementStatus.value || undefined }),
  ])
  overview.value = overviewData
  types.value = typeData
  notices.value = noticeData
  management.value = managementData
  if (!noticeForm.content_md && noticeData[0]) {
    noticeForm.title = noticeData[0].title
    noticeForm.content_md = noticeData[0].content_md
  }
  await loadSessions()
  await loadProgressSessions()
  await loadGroups()
}

async function loadSessions() {
  const data = await adminCarpoolAPI.listSessions({ page: 1, page_size: 100, status: sessionStatus.value || undefined })
  sessions.value = data.items
}

async function loadProgressSessions() {
  const data = await adminCarpoolAPI.listSessions({ page: 1, page_size: 100, status: 'recruiting' })
  progressSessionList.value = data.items
}

async function loadManagement() {
  management.value = await adminCarpoolAPI.management({ page: 1, page_size: 50, status: managementStatus.value || undefined })
}

function resetManagementFilters() {
  managementStatus.value = ''
  managementVehicleTypeId.value = 0
  managementKeyword.value = ''
  loadManagement()
}

async function loadGroups() {
  groups.value = await adminAPI.groups.getAll()
}

async function loadSelectedGroupAccounts(groupID: number) {
  selectedGroupAccounts.value = []
  if (!groupID) return
  groupAccountLoading.value = true
  try {
    const data = await adminAPI.accounts.list(1, 50, { group: String(groupID), sort_by: 'priority', sort_order: 'asc' })
    selectedGroupAccounts.value = data.items || []
  } catch {
    appStore.showError('订阅分组绑定账号读取失败')
  } finally {
    groupAccountLoading.value = false
  }
}

function openTypeEditor(item?: CarpoolVehicleType) {
  editingType.value = item ? { ...item } : {
    id: 0,
    product: 'openai',
    plan_tier: 'pro',
    multiplier: '20x',
    name: 'OpenAI Pro 20x 车',
    seat_count: 2,
    total_price: 1300,
    unit_price: 650,
    service_days: 30,
    refund_wait_hours: 2,
    completed_base_count: 0,
    enabled: false,
    support_revenue_pool: true,
    require_static_ip: true,
    wait_duration_options: [2],
    refund_methods: ['balance', 'gateway'],
    description: '',
    sort_order: 0,
    created_at: '',
    updated_at: '',
  }
  editingTypeRefundMethods.value = (editingType.value.refund_methods || ['balance', 'gateway']).join(',')
}

async function saveEditingType() {
  if (!editingType.value) return
  await saveType({
    ...editingType.value,
    wait_duration_options: [Number(editingType.value.refund_wait_hours || 2)],
    refund_methods: parseStringList(editingTypeRefundMethods.value),
  })
  editingType.value = null
}

async function saveType(item: CarpoolVehicleType) {
  const payload = {
    product: normalizeCarpoolCode(item.product, 'custom'),
    plan_tier: normalizeCarpoolCode(item.plan_tier, 'custom'),
    multiplier: normalizeCarpoolCode(item.multiplier, 'custom'),
    name: item.name,
    seat_count: Number(item.seat_count),
    total_price: Number(item.total_price),
    unit_price: Number(item.unit_price),
    service_days: Number(item.service_days || 30),
    refund_wait_hours: Number(item.refund_wait_hours || 2),
    completed_base_count: Number(item.completed_base_count || 0),
    enabled: Boolean(item.enabled),
    support_revenue_pool: Boolean(item.support_revenue_pool),
    require_static_ip: Boolean(item.require_static_ip),
    wait_duration_options: [Number(item.refund_wait_hours || 2)],
    refund_methods: item.refund_methods?.length ? item.refund_methods : ['balance', 'gateway'],
    description: item.description || '',
    sort_order: Number(item.sort_order || 0),
  }
  if (item.id) {
    await adminCarpoolAPI.updateType(item.id, payload)
  } else {
    await adminCarpoolAPI.createType(payload)
  }
  appStore.showSuccess('已保存')
  await loadAll()
}

async function deleteType(id: number) {
  if (!id) return
  await adminCarpoolAPI.deleteType(id)
  appStore.showSuccess('已删除')
  await loadAll()
}

async function openSessionEditor(session: CarpoolSession) {
  editingSession.value = cloneSession(session)
  sessionVouchers.value = session.edges?.vouchers || []
  try {
    sessionVouchers.value = await adminCarpoolAPI.listVouchers(session.id)
  } catch {
    // The eager-loaded list is enough for display if the explicit refresh fails.
  }
}

function cloneSession(session: CarpoolSession) {
  const communication = session.communication || {}
  const account = session.account_info || {}
  editingProvision.status = 'active'
  editingProvision.communication_type = stringField(communication.type, 'system_chat')
  editingProvision.communication_group_no = stringField(communication.group_no)
  editingProvision.communication_link = stringField(communication.link)
  editingProvision.communication_note = stringField(communication.note)
  assignmentForm.group_id = numberField(account.subscription_group_id)
  assignmentForm.account_pool_group_id = assignmentForm.group_id
  assignmentForm.validity_days = numberField(account.subscription_validity_days, session.edges?.vehicle_type?.service_days || 30)
  assignmentForm.notes = stringField(account.subscription_notes) || `拼车轮次 ${session.session_no || session.id} 发车分配`
  editingProvision.admin_notes = session.admin_notes || ''
  editingProvision.voucher_file_url = ''
  editingProvision.voucher_file_name = ''
  editingProvision.voucher_description = ''
  return session
}

async function saveProvision() {
  if (!editingSession.value) return
  if (!assignmentForm.group_id) {
    appStore.showWarning('请选择订阅分组')
    return
  }
  assignmentForm.account_pool_group_id = assignmentForm.group_id
  if (assignmentForm.validity_days <= 0) {
    appStore.showWarning('有效期必须大于 0 天')
    return
  }
  const userIds = Array.from(new Set(assignableParticipants.value.map((participant) => participant.user_id)))
  if (userIds.length === 0) {
    appStore.showWarning('当前轮次没有可分配的已付款成员')
    return
  }

  assigning.value = true
  try {
    const wasActive = editingSession.value.status === 'active'
    const group = selectedSubscriptionGroup.value
    const notes = assignmentForm.notes.trim() || `拼车轮次 ${editingSession.value.session_no || editingSession.value.id} 发车分配`
    const result = await adminAPI.subscriptions.bulkAssign({
      user_ids: userIds,
      group_id: assignmentForm.group_id,
      validity_days: Number(assignmentForm.validity_days),
      notes,
    })
    if (result.failed_count > 0) {
      appStore.showError(`订阅分配失败 ${result.failed_count} 人：${(result.errors || []).join('；')}`)
      return
    }

    await adminCarpoolAPI.provisionSession(editingSession.value.id, {
      status: 'active',
      communication: compactRecord({
        type: editingProvision.communication_type,
        group_no: editingProvision.communication_group_no,
        link: editingProvision.communication_link,
        note: editingProvision.communication_note,
      }),
      account_info: compactRecord({
        subscription_group_id: assignmentForm.group_id,
        subscription_group_name: group?.name || '',
        account_pool_group_id: assignmentForm.group_id,
        account_pool_group_name: group?.name || '',
        subscription_validity_days: Number(assignmentForm.validity_days),
        subscription_notes: notes,
        assigned_user_count: userIds.length,
        assigned_at: new Date().toISOString(),
      }),
      proxy_info: {},
      admin_notes: editingProvision.admin_notes,
    })

    if (editingProvision.voucher_file_url) {
      await adminCarpoolAPI.createVoucher(editingSession.value.id, {
        file_url: editingProvision.voucher_file_url,
        file_name: editingProvision.voucher_file_name || '拼车交付凭证',
        description: editingProvision.voucher_description,
      })
    }
    editingSession.value = null
    appStore.showSuccess(`${wasActive ? '交付配置已保存' : '已分配订阅并发车'}：新建 ${result.created_count}，复用 ${result.reused_count}`)
    await loadManagement()
    await loadSessions()
  } finally {
    assigning.value = false
  }
}

async function deleteVoucher(id: number) {
  if (!editingSession.value) return
  await adminCarpoolAPI.deleteVoucher(id)
  sessionVouchers.value = await adminCarpoolAPI.listVouchers(editingSession.value.id)
  appStore.showSuccess('凭证已删除')
}

async function saveNotice() {
  await adminCarpoolAPI.createNotice({ ...noticeForm })
  appStore.showSuccess('用户须知已发布')
  await loadAll()
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

function parseStringList(raw: string) {
  const values = raw.split(',').map((item) => item.trim()).filter(Boolean)
  return values.length ? values : ['balance', 'gateway']
}

function realCompletedCount(vehicleTypeId?: number) {
  const map = (overview.value.completed_by_vehicle_type || {}) as Record<string, number>
  return Number(map[String(vehicleTypeId || 0)] || 0)
}

function compactRecord(input: Record<string, unknown>) {
  const entries = Object.entries(input)
    .map(([key, value]) => [key, typeof value === 'string' ? value.trim() : value] as const)
    .filter(([, value]) => value !== '' && value !== null && value !== undefined)
  return Object.fromEntries(entries)
}

function stringField(value: unknown, fallback = '') {
  return typeof value === 'string' ? value : fallback
}

function numberField(value: unknown, fallback = 0) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function memberSummary(row: CarpoolAdminSessionRow) {
  const names = row.participants
    .slice(0, 3)
    .map((member) => member.user?.username || member.user?.email || `用户 ${member.participant.user_id}`)
  const suffix = row.participants.length > names.length ? ` 等 ${row.participants.length} 人` : ''
  return names.length ? `${names.join('、')}${suffix}` : '暂无成员'
}

function sessionPaidAmount(row: CarpoolAdminSessionRow) {
  return row.participants.reduce((sum, member) => sum + Number(member.participant.amount || 0), 0)
}

function statusLabel(value?: string) {
  const labels: Record<string, string> = {
    recruiting: '招募中',
    full: '已满员',
    provisioning: '采购中',
    active: '已发车',
    failed: '拼车失败',
    cancelled: '已取消',
    ended: '已结束',
    pending_payment: '待支付',
    paid: '已付款',
    refund_pending: '待退款',
    refunded_balance: '已退余额',
    refunded_gateway: '已原路退',
  }
  return labels[String(value || '')] || value || '-'
}

function communicationLabel(value?: Record<string, unknown>) {
  if (!value) return '未配置'
  const type = stringField(value.type)
  const groupNo = stringField(value.group_no)
  const link = stringField(value.link)
  if (!type && !groupNo && !link) return '未配置'
  const typeLabels: Record<string, string> = {
    system_chat: '系统聊天',
    qq: 'QQ群',
    wechat: '微信群',
    other: '其他',
  }
  return [typeLabels[type] || type, groupNo || link].filter(Boolean).join(' · ')
}

function compactNumber(value: number) {
  const n = Number(value || 0)
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(Math.round(n))
}

function sessionPaidCount(session: CarpoolSession) {
  return Number(session.paid_count || 0)
}

function sessionSeatCount(session: CarpoolSession) {
  return Number(session.seat_count || 0)
}

function sessionProgressLabel(session: CarpoolSession) {
  return `${sessionPaidCount(session)}/${sessionSeatCount(session)}`
}

function sessionProgressWidth(session: CarpoolSession) {
  const total = Math.max(1, sessionSeatCount(session))
  return `${Math.min(100, (sessionPaidCount(session) / total) * 100)}%`
}

function platformLabel(value?: string) {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
  }
  return labels[String(value || '').toLowerCase()] || value || '-'
}

function limitLabel(value?: number | null) {
  return value == null || value <= 0 ? '不限' : `$${Number(value).toFixed(2)}`
}

function money(value: number) {
  return Number(value || 0).toFixed(2)
}

function formatTime(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function accountStatusLabel(status?: string) {
  const labels: Record<string, string> = {
    active: '正常',
    inactive: '停用',
    error: '异常',
  }
  return labels[String(status || '')] || status || '-'
}

watch(() => assignmentForm.group_id, (groupID) => {
  assignmentForm.account_pool_group_id = Number(groupID || 0)
  loadSelectedGroupAccounts(Number(groupID || 0))
})

onMounted(loadAll)
</script>
