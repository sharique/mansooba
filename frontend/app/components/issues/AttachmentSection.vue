<template>
  <div class="mt-6">
    <div class="flex items-center justify-between mb-3">
      <h3 class="font-semibold text-sm text-base-content/70 uppercase tracking-wide">
        Attachments<span v-if="store.attachments.length" class="text-base-content/40"> ({{ store.attachments.length }})</span>
      </h3>
      <button class="btn btn-ghost btn-xs gap-1" :disabled="store.uploading" @click="triggerFilePicker">
        <span v-if="store.uploading" class="loading loading-spinner loading-xs" />
        <Icon v-else name="mdi:paperclip" class="w-4 h-4" />
        Attach files
      </button>
      <input
        ref="fileInput"
        type="file"
        multiple
        class="hidden"
        @change="onFilesSelected"
      >
    </div>

    <!-- Attachment list -->
    <div v-if="store.attachments.length" class="space-y-2">
      <div
        v-for="a in store.attachments"
        :key="a.id"
        class="flex items-center justify-between gap-2 rounded-lg border border-base-300 px-3 py-2 bg-base-100"
      >
        <button
          class="flex items-center gap-2 min-w-0 text-left flex-1 hover:underline"
          :aria-label="`Download ${a.filename}`"
          @click="download(a.id)"
        >
          <Icon name="mdi:file-outline" class="w-4 h-4 shrink-0 text-base-content/50" />
          <span class="text-sm truncate">{{ a.filename }}</span>
          <span class="text-xs text-base-content/40 shrink-0">{{ formatFileSize(a.size_bytes) }}</span>
        </button>
        <div class="flex items-center gap-2 shrink-0">
          <span class="text-xs text-base-content/40">{{ a.uploader_name }} · {{ formatDateTime(a.created_at) }}</span>
          <button
            v-if="a.uploader_id === currentUserId"
            class="btn btn-ghost btn-xs text-error"
            :aria-label="`Delete ${a.filename}`"
            @click="remove(a.id)"
          >
            <Icon name="mdi:close" class="w-3 h-3" />
          </button>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <p v-else class="text-sm text-base-content/40 py-2">No attachments.</p>
  </div>
</template>

<script setup lang="ts">
import { useAttachmentsStore } from '~/stores/attachments.store'

const props = defineProps<{ issueId: number; currentUserId: number }>()
const store = useAttachmentsStore()
const { showSuccess, showError } = useToast()
const { formatDateTime } = useTimeFormatter()

const fileInput = ref<HTMLInputElement | null>(null)

function triggerFilePicker() {
  fileInput.value?.click()
}

async function onFilesSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  input.value = '' // allow re-selecting the same file(s) next time
  if (!files.length) return

  await store.uploadFiles(props.issueId, files)

  if (store.error) {
    showError(store.error)
    return
  }
  if (store.lastRejected.length) {
    showError(`${store.lastRejected.length} file(s) rejected: ${store.lastRejected.map(r => r.reason).join('; ')}`)
  }
  const uploadedCount = files.length - store.lastRejected.length
  if (uploadedCount > 0) {
    showSuccess(uploadedCount === 1 ? 'File attached' : `${uploadedCount} files attached`)
  }
}

async function download(attachmentId: number) {
  await store.downloadAttachment(props.issueId, attachmentId)
  if (store.error) showError(store.error)
}

async function remove(attachmentId: number) {
  try {
    await store.deleteAttachment(props.issueId, attachmentId)
    showSuccess('Attachment removed')
  } catch {
    showError('Failed to remove attachment')
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

onMounted(() => store.fetchAttachments(props.issueId))
</script>
