const toasts = ref<{ id: number; type: 'success' | 'error'; message: string }[]>([])

export function useToast() {
  const show = (type: 'success' | 'error', message: string) => {
    const id = Date.now()
    toasts.value.push({ id, type, message })
    setTimeout(() => {
      toasts.value = toasts.value.filter(t => t.id !== id)
    }, 3000)
  }

  return {
    toasts,
    showSuccess: (m: string) => show('success', m),
    showError: (m: string) => show('error', m),
  }
}
