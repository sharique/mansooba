<script setup lang="ts">
import type { VelocityDataPoint } from '~/types/domain.types'

const props = defineProps<{
  data: VelocityDataPoint[]
}>()

// Compute the maximum story-point value across all data points so bar heights
// are proportional to the overall range (preventing division-by-zero).
const maxValue = computed(() => {
  if (props.data.length === 0) return 1
  return Math.max(...props.data.map(d => Math.max(d.committed, d.completed)), 1)
})

function barHeightPct(value: number): string {
  return `${Math.round((value / maxValue.value) * 100)}%`
}
</script>

<template>
  <!-- Empty state -->
  <div
    v-if="data.length === 0"
    class="flex items-center justify-center h-40 text-base-content/50 text-sm"
  >
    No completed sprints yet.
  </div>

  <div v-else class="w-full">
    <!-- Legend -->
    <div class="flex items-center gap-6 mb-4 text-sm">
      <span class="flex items-center gap-1.5">
        <span class="inline-block w-3 h-3 rounded-sm bg-neutral" aria-hidden="true" />
        Committed
      </span>
      <span class="flex items-center gap-1.5">
        <span class="inline-block w-3 h-3 rounded-sm bg-success" aria-hidden="true" />
        Completed
      </span>
    </div>

    <!-- Bar chart area -->
    <div class="relative">
      <!-- Y-axis label -->
      <span class="absolute -left-10 top-1/2 -translate-y-1/2 -rotate-90 text-xs text-base-content/50 whitespace-nowrap select-none">
        Story Points
      </span>

      <!-- Chart columns -->
      <div
        class="flex items-end gap-6 overflow-x-auto pb-2"
        role="img"
        aria-label="Velocity chart showing committed vs completed story points per sprint"
      >
        <div
          v-for="point in data"
          :key="point.sprint_id"
          class="flex flex-col items-center gap-1 shrink-0"
        >
          <!-- Bar group: committed + completed side by side -->
          <div class="flex items-end gap-1 h-48">
            <!-- Committed bar -->
            <div class="flex flex-col items-center gap-0.5">
              <span class="text-xs text-base-content/70 font-mono">{{ point.committed }}</span>
              <div
                class="w-8 bg-neutral rounded-t transition-all duration-300"
                :style="{ height: barHeightPct(point.committed) }"
                :title="`Committed: ${point.committed} pts`"
              />
            </div>
            <!-- Completed bar -->
            <div class="flex flex-col items-center gap-0.5">
              <span class="text-xs text-success font-mono">{{ point.completed }}</span>
              <div
                class="w-8 bg-success rounded-t transition-all duration-300"
                :style="{ height: barHeightPct(point.completed) }"
                :title="`Completed: ${point.completed} pts`"
              />
            </div>
          </div>

          <!-- Sprint name label -->
          <span
            class="text-xs text-base-content/60 max-w-[5rem] text-center truncate"
            :title="point.sprint_name"
          >
            {{ point.sprint_name }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
