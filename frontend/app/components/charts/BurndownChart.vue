<script setup lang="ts">
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import type { BurndownData } from '~/types/domain.types'
import { toBurndownChartData } from '~/utils/chart'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
)

const props = defineProps<{
  data: BurndownData
}>()

const chartData = computed(() => toBurndownChartData(props.data))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'top' as const },
    title: { display: true, text: `Burndown — ${props.data.sprint_name}` },
  },
  scales: {
    y: {
      beginAtZero: true,
      title: { display: true, text: 'Story Points' },
    },
    x: {
      title: { display: true, text: 'Date' },
    },
  },
}
</script>

<template>
  <div class="relative h-64">
    <Line :data="chartData" :options="chartOptions" />
  </div>
</template>
