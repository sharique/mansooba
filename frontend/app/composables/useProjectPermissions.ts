/**
 * What the signed-in user may do in the open project. Reads only what the
 * server sent (`permissions`, `myRole`); the role table is never copied here.
 * Missing permissions mean nothing is allowed.
 */
const DENIED: Record<string, string> = {
  'task.write': 'edit tasks',
  'task.delete_any': "delete other people's tasks",
  'comment.write': 'comment',
  'comment.delete_any': "delete other people's comments",
  'label.write': 'change labels',
  'label.delete': 'delete labels',
  'attachment.write': 'attach files',
  'attachment.delete_any': "delete other people's files",
  'sprint.manage': 'manage sprints',
  'project.edit': 'edit the project',
  'member.manage': 'manage people',
  'project.delete': 'delete the project',
}

export function useProjectPermissions() {
  const projectsStore = useProjectsStore()

  const myRole = computed<string | null>(() => projectsStore.current?.myRole ?? null)
  const permissions = computed<string[]>(() => projectsStore.current?.permissions ?? [])

  const can = (capability: string): boolean => permissions.value.includes(capability)

  const isReadOnly = computed(() => myRole.value === 'viewer' || permissions.value.length === 0)

  /** One-line reason for a disabled control, or undefined when allowed, e.g. "Viewers can't edit tasks". */
  const reason = (capability: string): string | undefined => {
    if (can(capability)) return undefined
    const role = myRole.value
    const who = role ? `${role.charAt(0).toUpperCase()}${role.slice(1)}s` : 'You'
    return `${who} can't ${DENIED[capability] ?? 'do this'}`
  }

  return { myRole, permissions, can, isReadOnly, reason }
}
