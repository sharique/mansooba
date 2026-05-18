<template>
  <div>
    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li><NuxtLink :to="`/projects/${key}`">{{ key }}</NuxtLink></li>
        <li>{{ issuesStore.current?.key ?? id }}</li>
      </ul>
    </div>

    <div v-if="loading" class="space-y-4">
      <div class="skeleton h-8 w-96" />
      <div class="skeleton h-4 w-64" />
      <div class="skeleton h-32 w-full" />
    </div>

    <div v-else-if="issuesStore.current">
      <IssuesIssueDetail
        :issue="issuesStore.current"
        :project-key="key"
        :members="members"
        @deleted="navigateTo(`/projects/${key}`)"
      />
    </div>

    <div v-else class="text-center py-20 text-base-content/50">
      <p>Issue not found.</p>
      <NuxtLink :to="`/projects/${key}`" class="btn btn-ghost mt-4">Back to project</NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { MemberResponse } from '~/types/domain.types'
import { projectsService } from '~/services/projects.service'

const route = useRoute()
const key = route.params.key as string
const id = route.params.id as string

const issuesStore = useIssuesStore()
const { showError } = useToast()
const loading = ref(true)
const members = ref<MemberResponse[]>([])

onMounted(async () => {
  try {
    await Promise.all([
      issuesStore.fetchOne(key, Number(id)),
      projectsService.listMembers(key).then(v => members.value = v).catch(() => {}),
    ])
  }
  catch {
    showError('Failed to load issue')
  }
  finally {
    loading.value = false
  }
})
</script>
