/**
 * Single definition of "can manage" (FR-003): the project owner, or a member
 * with role `admin`. Reads the stores only; the page is responsible for
 * loading `projectsStore.current` and `projectsStore.members`.
 */
export function useCanManageProject() {
  const projectsStore = useProjectsStore()
  const authStore = useAuthStore()

  const canManage = computed(() => {
    const me = authStore.user?.id
    if (me === undefined || !projectsStore.current) return false
    if (projectsStore.current.ownerId === me) return true
    return projectsStore.members.find(m => m.user_id === me)?.role === 'admin'
  })

  return { canManage }
}
