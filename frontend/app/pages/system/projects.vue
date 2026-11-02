<template>
  <div class="max-w-4xl mx-auto py-8 px-4">
    <h1 class="text-2xl font-bold mb-2">Projects without an admin</h1>
    <p class="text-sm text-base-content/70 mb-6">
      These projects have no active admin, so nobody can manage their people or settings.
      Taking admin makes you an admin of that project.
    </p>

    <div v-if="store.error" class="alert alert-error mb-4 flex items-center gap-2">
      <span>{{ store.error }}</span>
      <button class="btn btn-sm" @click="store.fetch()">Retry</button>
    </div>

    <div v-if="store.loading" class="space-y-2">
      <div v-for="n in 3" :key="n" class="skeleton h-10 w-full rounded" />
    </div>

    <div v-else-if="store.projects.length" class="overflow-x-auto">
      <table class="table w-full">
        <thead>
          <tr>
            <th>Key</th>
            <th>Name</th>
            <th>Members</th>
            <th><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in store.projects" :key="p.key">
            <td class="font-mono">{{ p.key }}</td>
            <td>{{ p.name }}</td>
            <td>{{ p.member_count }}</td>
            <td class="text-right">
              <button
                class="btn btn-sm btn-primary"
                :aria-label="`Take admin of ${p.name}`"
                :disabled="claiming === p.key"
                @click="takeAdmin(p)"
              >
                Take admin
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <p v-else-if="!store.error" class="text-base-content/70 py-8 text-center">
      Every project has an active admin.
    </p>
  </div>
</template>

<script setup lang="ts">
import type { ProjectWithoutAdmin } from '~/types/domain.types'

const authStore = useAuthStore()
const store = useAdminProjectsStore()
const { showSuccess, showError } = useToast()
const claiming = ref<string | null>(null)

onMounted(async () => {
  if (!authStore.isAdmin) {
    await navigateTo('/')
    return
  }
  await store.fetch()
})

async function takeAdmin(p: ProjectWithoutAdmin) {
  if (!confirm(`Become an admin of "${p.name}"? Its members will be told.`)) return
  claiming.value = p.key
  try {
    await store.claim(p.key)
    showSuccess(`You are now an admin of "${p.name}"`)
  }
  catch (e) {
    showError((e as Error).message)
  }
  finally {
    claiming.value = null
  }
}
</script>
