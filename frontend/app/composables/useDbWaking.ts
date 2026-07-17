// Shared reactive flag for the "database is waking up" indicator (spec 010,
// db-idle-autostop, FR-013). Mirrors useToast's module-level ref pattern —
// any component can show a loading state while this is true.
const isDbWaking = ref(false)

export function useDbWaking() {
  return {
    isDbWaking,
    setDbWaking: (waking: boolean) => {
      isDbWaking.value = waking
    },
  }
}
