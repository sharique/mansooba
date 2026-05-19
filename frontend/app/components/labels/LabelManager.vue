<template>
  <div>
    <h2 class="text-lg font-semibold mb-4">Labels</h2>
    <div class="flex gap-2 items-end mb-6">
      <div class="form-control">
        <label class="label label-text">Name</label>
        <input v-model="form.name" type="text" class="input input-bordered input-sm" placeholder="e.g. bug" />
      </div>
      <div class="form-control">
        <label class="label label-text">Color</label>
        <div class="flex gap-1">
          <button
            v-for="color in palette"
            :key="color"
            class="w-6 h-6 rounded-full border-2"
            :class="form.color === color ? 'border-primary' : 'border-transparent'"
            :style="{ backgroundColor: color }"
            @click="form.color = color"
          />
        </div>
      </div>
      <button class="btn btn-sm btn-primary" :disabled="!form.name || !form.color" @click="create">Add label</button>
    </div>
    <div class="space-y-2">
      <div v-for="label in store.projectLabels" :key="label.id" class="flex items-center justify-between">
        <LabelBadge :label="label" />
        <button class="btn btn-xs btn-ghost text-error" @click="remove(label.id)">Delete</button>
      </div>
      <p v-if="store.projectLabels.length === 0" class="text-base-content/40 text-sm">No labels yet.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useLabelsStore } from '~/stores/labels.store'
import { LabelPalette } from '~/types/domain.types'

const props = defineProps<{ projectKey: string }>()
const store = useLabelsStore()
const palette = LabelPalette
const form = reactive({ name: '', color: '' })

onMounted(() => store.fetchProjectLabels(props.projectKey))

async function create() {
  if (!form.name || !form.color) return
  await store.createLabel(props.projectKey, form.name, form.color)
  form.name = ''
  form.color = ''
}
async function remove(id: number) { await store.deleteLabel(props.projectKey, id) }
</script>
