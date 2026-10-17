import { defineStore } from 'pinia'
import { overviewService } from '~/services/overview.service'
import type { ActivityScope, OpenTask, OverviewList, RecentActivityItem, SprintSummary } from '~/types/domain.types'

// One section of the overview page. Each has its own loading and error state so
// a failure in one never hides the others.
interface Section<T> {
  data: T | null
  loading: boolean
  error: string | null
}

// A section starts as "loading": until its first request has finished the page must
// show a skeleton, not an empty or "no sprint" message.
const empty = <T>(): Section<T> => ({ data: null, loading: true, error: null })

function messageOf(e: unknown): string {
  const err = e as { data?: { message?: string }; message?: string }
  return err?.data?.message ?? err?.message ?? 'Something went wrong'
}

export const useOverviewStore = defineStore('overview', () => {
  const sprint = ref<Section<SprintSummary | null>>(empty())
  const tasks = ref<Section<OverviewList<OpenTask>>>(empty())
  const activity = ref<Section<OverviewList<RecentActivityItem>>>(empty())
  const scope = ref<ActivityScope>('all')

  async function load<T>(section: Section<T>, request: () => Promise<T>, stillWanted: () => boolean = () => true) {
    section.loading = true
    section.error = null
    try {
      const data = await request()
      if (stillWanted()) section.data = data
    }
    catch (e) {
      if (stillWanted()) section.error = messageOf(e)
    }
    finally {
      if (stillWanted()) section.loading = false
    }
  }

  const fetchSprint = (key: string) => load(sprint.value, () => overviewService.getSprint(key))
  const fetchTasks = (key: string) => load(tasks.value, () => overviewService.getTasks(key))

  // Switching tabs refetches only this section. A slow answer for a tab the
  // person has already left is dropped so it cannot overwrite the newer list.
  function fetchActivity(key: string, next?: ActivityScope) {
    if (next) scope.value = next
    const requested = scope.value
    return load(activity.value, () => overviewService.getActivity(key, requested), () => scope.value === requested)
  }

  function reset() {
    sprint.value = empty()
    tasks.value = empty()
    activity.value = empty()
    scope.value = 'all'
  }

  return { sprint, tasks, activity, scope, fetchSprint, fetchTasks, fetchActivity, reset }
})
