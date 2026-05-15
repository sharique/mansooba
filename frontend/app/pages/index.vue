<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'

definePageMeta({ middleware: [] })

const authStore = useAuthStore()
await navigateTo(authStore.isAuthenticated ? '/projects' : '/login')
</script>
