<template>
  <section class="card bg-base-100 border border-base-300" aria-labelledby="overview-activity-title">
    <div class="card-body gap-3">
      <h2 id="overview-activity-title" class="card-title text-base">Recent activity</h2>

      <div role="tablist" aria-label="Recent activity scope" class="tabs tabs-bordered">
        <button
          v-for="(tab, i) in tabs"
          :id="`activity-tab-${tab.scope}`"
          :key="tab.scope"
          ref="tabEls"
          role="tab"
          type="button"
          class="tab"
          :class="{ 'tab-active': scope === tab.scope }"
          :aria-selected="scope === tab.scope ? 'true' : 'false'"
          aria-controls="activity-panel"
          :tabindex="scope === tab.scope ? 0 : -1"
          @click="emit('update:scope', tab.scope)"
          @keydown="onKeydown($event, i)"
        >{{ tab.label }}</button>
      </div>

      <div id="activity-panel" role="tabpanel" :aria-labelledby="`activity-tab-${scope}`">
        <div v-if="loading" class="space-y-2">
          <div v-for="n in 3" :key="n" class="skeleton h-8 w-full" />
        </div>

        <div v-else-if="error" class="alert alert-error text-sm" role="alert">
          <span class="flex-1">Could not load recent activity: {{ error }}</span>
          <button class="btn btn-sm" data-retry @click="emit('retry')">Retry</button>
        </div>

        <p v-else-if="scope === 'sprint' && data && !data.sprint_active" class="text-sm text-base-content/60">
          No active sprint, so there is nothing to show for the current sprint.
        </p>

        <p v-else-if="!data?.items.length" class="text-sm text-base-content/60">
          No recent activity in the last 5 days.
        </p>

        <ul v-else class="divide-y divide-base-200">
          <li v-for="a in data.items" :key="a.id" data-activity class="py-2">
            <NuxtLink
              :to="`/projects/${projectKey}/issues/${a.issue_id}`"
              class="flex flex-wrap items-center gap-x-2 gap-y-1 hover:bg-base-200 rounded-box px-1"
            >
              <UserAvatar :avatar-url="a.actor.avatar_url" :name="a.actor.name" :user-id="a.actor.id" size="sm" />
              <span class="min-w-0">
                <span class="font-medium">{{ a.actor.name || 'Someone' }}</span>
                <template v-if="a.kind === 'status_changed'"> moved <span class="font-mono">{{ a.issue_key }}</span> to {{ statusLabel((a.new_status ?? '') as never) }}</template>
                <template v-else> commented on <span class="font-mono">{{ a.issue_key }}</span></template>
              </span>
              <span class="ml-auto text-xs text-base-content/60">{{ formatDateTime(a.created_at) }}</span>
            </NuxtLink>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import UserAvatar from '~/components/common/UserAvatar.vue'
import { statusLabel } from '~/utils/issueStyles'
import type { ActivityScope, OverviewList, RecentActivityItem } from '~/types/domain.types'

defineProps<{
  projectKey: string
  data: OverviewList<RecentActivityItem> | null
  loading: boolean
  error: string | null
  scope: ActivityScope
}>()
const emit = defineEmits<{ (e: 'update:scope', s: ActivityScope): void; (e: 'retry'): void }>()

const { formatDateTime } = useTimeFormatter()

const tabs: { scope: ActivityScope; label: string }[] = [
  { scope: 'all', label: 'All activity' },
  { scope: 'sprint', label: 'Current sprint' },
]
const tabEls = ref<HTMLElement[]>([])

// Arrow keys move to the other tab, wrapping, and select it (automatic activation).
function onKeydown(e: KeyboardEvent, index: number) {
  if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
  e.preventDefault()
  const step = e.key === 'ArrowRight' ? 1 : -1
  const next = (index + step + tabs.length) % tabs.length
  emit('update:scope', tabs[next]!.scope)
  nextTick(() => tabEls.value[next]?.focus())
}
</script>
