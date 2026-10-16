import { ref, watch, type Ref } from 'vue'
import { deriveUsername, validateUsername, normalizeUsername } from '~/utils/username'
import type { UsernameCheckResponse } from '~/types/domain.types'

export type UsernameStatus =
  | { kind: 'empty' }
  | { kind: 'invalid' | 'reserved' | 'taken'; message: string }
  | { kind: 'checking' | 'available' | 'unknown' }

interface Options {
  /** Asks the server whether a (valid) name is free; null skips the live check
   *  (the first administrator step has no signed-in user to ask on behalf of). */
  check: ((username: string) => Promise<UsernameCheckResponse>) | null
  debounceMs?: number
}

/**
 * State for the username field on the create-user forms (FR-012, FR-013).
 * The field follows the email until the person edits it, then stays theirs.
 * Rule errors show immediately; for a valid name the server is asked about
 * availability once typing pauses. A failing or missing check shows no verdict
 * and never blocks typing; the server validates again on submit.
 */
export function useUsernameField(email: Ref<string>, opts: Options) {
  const debounceMs = opts.debounceMs ?? 400
  const username = ref('')
  const touched = ref(false)
  const status = ref<UsernameStatus>({ kind: 'empty' })

  let timer: ReturnType<typeof setTimeout> | undefined
  let generation = 0

  function evaluate() {
    clearTimeout(timer)
    const mine = ++generation
    const name = normalizeUsername(username.value)
    if (!name) {
      status.value = { kind: 'empty' }
      return
    }
    const problem = validateUsername(name)
    if (problem) {
      status.value = { kind: problem.reason, message: problem.message }
      return
    }
    if (!opts.check) {
      status.value = { kind: 'unknown' }
      return
    }
    status.value = { kind: 'checking' }
    timer = setTimeout(async () => {
      try {
        const res = await opts.check!(name)
        if (mine !== generation) return // the field changed while we waited
        if (res.available) status.value = { kind: 'available' }
        else status.value = { kind: (res.reason as 'invalid' | 'reserved' | 'taken') || 'taken', message: res.message }
      }
      catch {
        if (mine === generation) status.value = { kind: 'unknown' }
      }
    }, debounceMs)
  }

  watch(email, (e) => {
    if (touched.value) return
    username.value = e.includes('@') ? deriveUsername(e) : ''
    evaluate()
  }, { immediate: true })

  function onInput(value: string) {
    touched.value = true
    username.value = value
    evaluate()
  }

  /** Shows a server rejection (for example "already taken") on the field. */
  function setServerError(message: string) {
    generation++
    clearTimeout(timer)
    const lower = message.toLowerCase()
    const kind = lower.includes('taken') ? 'taken' : lower.includes('reserved') ? 'reserved' : 'invalid'
    status.value = { kind, message }
  }

  return { username, touched, status, onInput, setServerError }
}
