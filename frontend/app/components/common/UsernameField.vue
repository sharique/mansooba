<template>
  <div class="form-control gap-1" data-testid="username-field">
    <label :for="id" class="label label-text font-medium">Username</label>
    <input
      :id="id"
      :value="field.username.value"
      type="text"
      autocomplete="off"
      autocapitalize="none"
      spellcheck="false"
      class="input input-bordered w-full"
      :class="{
        'input-error': ['invalid', 'reserved', 'taken'].includes(field.status.value.kind),
        'input-success': field.status.value.kind === 'available',
      }"
      :disabled="disabled"
      :aria-describedby="`${id}-help`"
      @input="field.onInput(($event.target as HTMLInputElement).value)"
    />
    <p :id="`${id}-help`" class="text-sm text-base-content/60">
      Used for @mentions. Cannot be changed later.
    </p>
    <p
      v-if="message"
      role="status"
      data-testid="username-status"
      class="text-sm"
      :class="tone"
    >
      {{ message }}
    </p>
  </div>
</template>

<script setup lang="ts">
import type { useUsernameField } from '~/composables/useUsernameField'

const props = defineProps<{
  field: ReturnType<typeof useUsernameField>
  id: string
  disabled?: boolean
}>()

const message = computed(() => {
  const s = props.field.status.value
  switch (s.kind) {
    case 'available': return 'Available'
    case 'checking': return 'Checking…'
    case 'invalid':
    case 'reserved':
    case 'taken': return s.message
    default: return ''
  }
})

const tone = computed(() => {
  const k = props.field.status.value.kind
  if (k === 'available') return 'text-success'
  if (k === 'checking') return 'text-base-content/60'
  return 'text-error'
})
</script>
