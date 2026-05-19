<template>
  <div class="relative">
    <div class="flex flex-wrap gap-1 items-center">
      <IssuesLabelBadge
        v-for="l in attachedLabels"
        :key="l.id"
        :label="l"
        class="cursor-pointer"
        :title="`Remove ${l.name}`"
        @click="detach(l.id)"
      />
      <div class="dropdown dropdown-bottom">
        <button tabindex="0" class="btn btn-xs btn-ghost">+ Label</button>
        <ul tabindex="0" class="dropdown-content menu bg-base-100 border border-base-200 rounded-box z-[10] p-1 shadow w-40">
          <li v-for="l in availableLabels" :key="l.id">
            <button class="flex items-center gap-2 text-sm" @click="attach(l)">
              <span class="w-3 h-3 rounded-full" :style="{ backgroundColor: l.color }" />
              {{ l.name }}
            </button>
          </li>
          <li v-if="availableLabels.length === 0">
            <span class="text-base-content/40 text-xs px-2">No labels</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useLabelsStore } from '~/stores/labels.store'
import type { Label } from '~/types/domain.types'

const props = defineProps<{ issueId: number; projectKey: string }>()
const store = useLabelsStore()

onMounted(() => Promise.all([
  store.fetchProjectLabels(props.projectKey),
  store.fetchIssueLabels(props.issueId),
]))

const attachedLabels = computed(() => store.issueLabels[props.issueId] ?? [])
const attachedIds = computed(() => new Set(attachedLabels.value.map(l => l.id)))
const availableLabels = computed(() => store.projectLabels.filter(l => !attachedIds.value.has(l.id)))

async function attach(label: Label) { await store.attachLabel(props.issueId, label) }
async function detach(labelId: number) { await store.detachLabel(props.issueId, labelId) }
</script>
