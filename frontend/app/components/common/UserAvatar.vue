<template>
  <div :class="['avatar-wrapper', sizeClass]">
    <img
      v-if="showImage"
      :src="resolvedAvatarUrl"
      :alt="name || email || 'Avatar'"
      :class="['avatar-img', sizeClass]"
      @error="showImage = false"
    />
    <div
      v-else
      class="avatar-initials"
      :class="sizeClass"
      :style="{ background: color }"
    >{{ initials }}</div>
  </div>
</template>

<script setup lang="ts">
import { useAvatarColor } from '~/composables/useAvatarColor'

const props = withDefaults(defineProps<{
  avatarUrl?: string
  name: string
  userId: number
  email?: string
  size?: 'sm' | 'md' | 'lg'
}>(), { size: 'md' })

const showImage = ref(!!props.avatarUrl)
const color = useAvatarColor(props.userId)

// Storage returns server-relative paths (/uploads/avatars/...). Prefix with the
// API server origin so the browser doesn't resolve against the Nuxt dev port.
const resolvedAvatarUrl = computed(() => {
  if (!props.avatarUrl) return undefined
  if (props.avatarUrl.startsWith('/')) {
    try {
      const config = useRuntimeConfig()
      const origin = new URL(config.public.apiBaseUrl as string).origin
      return origin + props.avatarUrl
    } catch {
      return props.avatarUrl
    }
  }
  return props.avatarUrl
})

const sizeClass = computed(() => `avatar-${props.size}`)

const initials = computed(() => {
  if (props.name && props.name.trim()) {
    const words = props.name.trim().split(/\s+/)
    if (words.length >= 2) {
      return (words[0][0] + words[1][0]).toUpperCase()
    }
    return props.name.trim().slice(0, 2).toUpperCase()
  }
  if (props.email) {
    const local = props.email.split('@')[0]
    return local.slice(0, 2).toUpperCase()
  }
  return '?'
})

watch(() => props.avatarUrl, (val) => {
  showImage.value = !!val
})
</script>

<style scoped>
.avatar-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-img,
.avatar-initials {
  border-radius: 50%;
  object-fit: cover;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  user-select: none;
}

.avatar-sm .avatar-img,
.avatar-sm.avatar-img,
.avatar-sm .avatar-initials,
.avatar-sm.avatar-initials,
.avatar-sm { width: 2rem; height: 2rem; font-size: 0.7rem; }

.avatar-md .avatar-img,
.avatar-md.avatar-img,
.avatar-md .avatar-initials,
.avatar-md.avatar-initials,
.avatar-md { width: 2.5rem; height: 2.5rem; font-size: 0.8rem; }

.avatar-lg .avatar-img,
.avatar-lg.avatar-img,
.avatar-lg .avatar-initials,
.avatar-lg.avatar-initials,
.avatar-lg { width: 4rem; height: 4rem; font-size: 1.2rem; }
</style>
