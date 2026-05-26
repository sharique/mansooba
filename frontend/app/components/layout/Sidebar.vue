<script lang="ts">
export interface NavLink { label: string, to: string, icon: string }

export function projectNavLinks(key: string): NavLink[] {
  return [
    { label: 'Overview', to: `/projects/${key}`,          icon: 'mdi:view-dashboard-outline' },
    { label: 'Board',    to: `/projects/${key}/board`,    icon: 'mdi:view-column-outline' },
    { label: 'Backlog',  to: `/projects/${key}/backlog`,  icon: 'mdi:format-list-bulleted' },
    { label: 'Sprints',  to: `/projects/${key}/backlog`,  icon: 'mdi:run-fast' },
    { label: 'Reports',  to: `/projects/${key}/reports`,  icon: 'mdi:chart-line' },
    { label: 'Settings', to: `/projects/${key}/settings`, icon: 'mdi:cog-outline' },
  ]
}
</script>

<script setup lang="ts">
import { useProjectsStore } from '~/stores/projects.store'

const route = useRoute()
const projectsStore = useProjectsStore()

const primary: NavLink[] = [
  { label: 'My Desk',  to: '/',         icon: 'mdi:monitor-dashboard' },
  { label: 'Projects', to: '/projects', icon: 'mdi:folder-multiple-outline' },
  { label: 'Reports',  to: '/reports',  icon: 'mdi:chart-box-outline' },
]

const currentKey = computed(() =>
  typeof route.params.key === 'string' ? route.params.key : null,
)

const projectLinks = computed(() => currentKey.value ? projectNavLinks(currentKey.value) : [])

const recentProjects = computed(() => projectsStore.projects.slice(0, 5))

function isActive(to: string): boolean {
  return route.path === to
}
</script>

<template>
  <aside class="w-60 bg-neutral text-neutral-content flex flex-col h-full">
    <!-- brand -->
    <NuxtLink to="/" class="flex items-center gap-2 px-4 h-14 font-bold text-lg shrink-0">
      <span class="inline-block w-6 h-6 rounded-md bg-primary" />
      Mansooba
    </NuxtLink>

    <nav class="flex-1 overflow-y-auto px-2 pb-4 space-y-1">
      <NuxtLink
        v-for="link in primary"
        :key="link.to"
        :to="link.to"
        class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors"
        :class="isActive(link.to) ? 'bg-primary text-primary-content font-semibold' : 'hover:bg-white/10'"
      >
        <Icon :name="link.icon" class="w-5 h-5 opacity-90" />
        {{ link.label }}
      </NuxtLink>

      <!-- contextual project section -->
      <template v-if="projectLinks.length">
        <div class="px-3 pt-4 pb-1 text-[10px] uppercase tracking-wide opacity-50">
          {{ currentKey }}
        </div>
        <NuxtLink
          v-for="link in projectLinks"
          :key="link.to + link.label"
          :to="link.to"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors"
          :class="isActive(link.to) ? 'bg-primary text-primary-content font-semibold' : 'hover:bg-white/10'"
        >
          <Icon :name="link.icon" class="w-5 h-5 opacity-90" />
          {{ link.label }}
        </NuxtLink>
      </template>

      <!-- recent projects -->
      <template v-if="recentProjects.length">
        <div class="px-3 pt-4 pb-1 text-[10px] uppercase tracking-wide opacity-50">Recent projects</div>
        <NuxtLink
          v-for="p in recentProjects"
          :key="p.id"
          :to="`/projects/${p.key}`"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm hover:bg-white/10 transition-colors"
        >
          <span class="w-2 h-2 rounded-full bg-accent shrink-0" />
          <span class="truncate">{{ p.name }}</span>
        </NuxtLink>
      </template>
    </nav>
  </aside>
</template>
