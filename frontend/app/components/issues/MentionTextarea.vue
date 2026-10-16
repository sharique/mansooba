<template>
  <div class="relative">
    <textarea
      ref="el"
      :value="modelValue"
      class="textarea textarea-bordered w-full text-sm"
      :rows="rows"
      :placeholder="placeholder"
      role="combobox"
      aria-autocomplete="list"
      :aria-expanded="ac.open.value ? 'true' : 'false'"
      :aria-controls="listId"
      :aria-activedescendant="activeId"
      @input="onInput"
      @keydown="onKeydown"
      @keyup="onCaretKey"
      @click="reportCaret"
      @blur="onBlur"
    />
    <ul
      v-if="ac.open.value"
      :id="listId"
      role="listbox"
      class="absolute z-20 left-0 right-0 mt-1 max-h-72 overflow-y-auto rounded-box border border-base-300 bg-base-100 shadow-lg"
    >
      <li
        v-for="(m, i) in ac.suggestions.value"
        :id="`${listId}-${m.user_id}`"
        :key="m.user_id"
        ref="options"
        role="option"
        :aria-selected="i === ac.activeIndex.value ? 'true' : 'false'"
        class="flex min-h-11 flex-col justify-center px-3 py-1 cursor-pointer"
        :class="i === ac.activeIndex.value ? 'bg-base-200' : ''"
        @mousedown.prevent="pick(m)"
      >
        <span class="truncate text-sm">{{ m.name }}</span>
        <span class="truncate text-xs text-base-content/60">@{{ m.username }}</span>
      </li>
      <li v-if="!ac.suggestions.value.length" class="px-3 py-3 text-sm text-base-content/60">
        No matching members
      </li>
    </ul>
    <div role="status" class="mt-1 text-xs text-warning">
      <p v-for="h in ac.unknown.value" :key="h">@{{ h }} is not a member of this project</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, toRef, useId, watch } from 'vue'
import type { MemberResponse } from '~/types/domain.types'
import { useMentionAutocomplete } from '~/composables/useMentionAutocomplete'

const props = withDefaults(defineProps<{
  modelValue: string
  members: MemberResponse[]
  currentUserId: number
  rows?: number
  placeholder?: string
}>(), { rows: 3, placeholder: '' })
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const el = ref<HTMLTextAreaElement | null>(null)
const options = ref<HTMLElement[]>([])
const listId = `mention-list-${useId()}`

// The composable edits this ref; the parent owns the real value.
const text = computed({ get: () => props.modelValue, set: v => emit('update:modelValue', v) })
const ac = useMentionAutocomplete({
  text,
  members: toRef(props, 'members'),
  currentUserId: toRef(props, 'currentUserId'),
})

const activeId = computed(() => {
  const m = ac.suggestions.value[ac.activeIndex.value]
  return ac.open.value && m ? `${listId}-${m.user_id}` : undefined
})

watch(() => ac.activeIndex.value, async () => {
  await nextTick()
  options.value[ac.activeIndex.value]?.scrollIntoView?.({ block: 'nearest' })
})

function onInput(e: Event) {
  const ta = e.target as HTMLTextAreaElement
  emit('update:modelValue', ta.value)
  // The composable reads the value through `text`, which follows the prop;
  // set it directly so the query is computed from what was just typed.
  ac.onInput(ta.selectionStart ?? ta.value.length, ta.value)
}

function reportCaret() {
  if (el.value) ac.onCaret(el.value.selectionStart ?? 0, el.value.value)
}

function onCaretKey(e: KeyboardEvent) {
  if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) reportCaret()
}

function onKeydown(e: KeyboardEvent) {
  ac.onKeydown(e)
}

async function pick(m: MemberResponse) {
  const res = ac.select(m)
  await nextTick()
  el.value?.focus()
  el.value?.setSelectionRange(res.caret, res.caret)
}

function onBlur() {
  // Selecting uses mousedown, so a short delay is only needed for touch taps.
  setTimeout(() => ac.close(), 100)
}
</script>
