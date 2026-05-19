import { defineStore } from 'pinia'
import { labelsService } from '~/services/labels.service'
import type { Label } from '~/types/domain.types'

export const useLabelsStore = defineStore('labels', () => {
  const projectLabels = ref<Label[]>([])
  const issueLabels = ref<Record<number, Label[]>>({})
  const error = ref<string | null>(null)

  async function fetchProjectLabels(projectKey: string) {
    try {
      projectLabels.value = await labelsService.list(projectKey)
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    }
  }

  async function createLabel(projectKey: string, name: string, color: string) {
    const label = await labelsService.create(projectKey, name, color)
    projectLabels.value.push(label)
  }

  async function deleteLabel(projectKey: string, labelId: number) {
    await labelsService.delete(projectKey, labelId)
    projectLabels.value = projectLabels.value.filter(l => l.id !== labelId)
  }

  async function attachLabel(issueId: number, label: Label) {
    await labelsService.attach(issueId, label.id)
    if (!issueLabels.value[issueId]) issueLabels.value[issueId] = []
    if (!issueLabels.value[issueId].find(l => l.id === label.id)) {
      issueLabels.value[issueId].push(label)
    }
  }

  async function detachLabel(issueId: number, labelId: number) {
    await labelsService.detach(issueId, labelId)
    if (issueLabels.value[issueId]) {
      issueLabels.value[issueId] = issueLabels.value[issueId].filter(l => l.id !== labelId)
    }
  }

  return { projectLabels, issueLabels, error, fetchProjectLabels, createLabel, deleteLabel, attachLabel, detachLabel }
})
