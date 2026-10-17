import { computed, onScopeDispose, ref, watch, type Ref } from 'vue'
import type { MemberResponse } from '~/types/domain.types'
import { applySuggestion, filterSuggestions, findMentionQuery, findUnknownMentions, type MentionQuery } from './useMentions'

interface Options {
  text: Ref<string>
  members: Ref<MemberResponse[]>
  currentUserId: Ref<number>
  /** Pause after typing before the non-member warning updates (ms). */
  warnDelay?: number
}

/**
 * State and keyboard handling for the @mention list under a textarea. The
 * caller owns the text and reports the caret after each edit; nothing here
 * makes a request, it only reads the already loaded member list.
 */
export function useMentionAutocomplete({ text, members, currentUserId, warnDelay = 400 }: Options) {
  const mq = ref<MentionQuery | null>(null)
  const activeIndex = ref(0)
  const dismissed = ref(false)
  const unknown = ref<string[]>([])
  let timer: ReturnType<typeof setTimeout> | undefined

  const open = computed(() => mq.value !== null && !dismissed.value && members.value.length > 0)
  const suggestions = computed(() =>
    open.value ? filterSuggestions(members.value, mq.value!.query, currentUserId.value) : [])

  /** `value` is the textarea's current text when the bound ref has not caught up yet. */
  function onInput(caret: number, value: string = text.value) {
    const next = findMentionQuery(value, caret)
    if (next?.query !== mq.value?.query || next?.start !== mq.value?.start) {
      activeIndex.value = 0
      dismissed.value = false
    }
    mq.value = members.value.length ? next : null

    clearTimeout(timer)
    if (!value) unknown.value = []
    else timer = setTimeout(() => { unknown.value = findUnknownMentions(value, members.value) }, warnDelay)
  }

  /** Caret moved without an edit (click, arrow keys when the list is closed). */
  function onCaret(caret: number, value: string = text.value) {
    const next = findMentionQuery(value, caret)
    mq.value = members.value.length ? next : null
  }

  function close() { dismissed.value = true }

  function select(member: MemberResponse) {
    const res = applySuggestion(text.value, mq.value!, member.username)
    text.value = res.text
    mq.value = null
    activeIndex.value = 0
    return res
  }

  /** Returns true when the key was used by the list (caller must not type it). */
  function onKeydown(e: KeyboardEvent): boolean {
    if (!open.value) return false
    const list = suggestions.value
    switch (e.key) {
      case 'ArrowDown':
      case 'ArrowUp': {
        if (!list.length) return false
        e.preventDefault()
        const step = e.key === 'ArrowDown' ? 1 : -1
        activeIndex.value = (activeIndex.value + step + list.length) % list.length
        return true
      }
      case 'Enter':
      case 'Tab': {
        const pick = list[activeIndex.value]
        if (!pick) return false
        e.preventDefault()
        select(pick)
        return true
      }
      case 'Escape':
        close()
        return true
      default:
        return false
    }
  }

  // The parent can empty the box without typing (after posting): drop the list and the warning.
  watch(text, (v) => {
    if (v) return
    clearTimeout(timer)
    mq.value = null
    unknown.value = []
  })

  onScopeDispose(() => clearTimeout(timer))

  return { open, suggestions, activeIndex, unknown, onInput, onCaret, onKeydown, select, close }
}
