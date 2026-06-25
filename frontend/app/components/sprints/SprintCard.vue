<script setup lang="ts">
import { SprintStatus } from "~/types/domain.types";
import type { Sprint, Issue } from "~/types/domain.types";

const props = defineProps<{
    sprint: Sprint;
    projectKey: string;
    canManage: boolean;
    hasActiveSprint: boolean;
    issues?: Issue[];
}>();

const emit = defineEmits<{
    start: [sprint: Sprint];
    complete: [sprint: Sprint];
    edit: [sprint: Sprint];
    delete: [sprint: Sprint];
    expand: [sprint: Sprint];
    removeIssue: [{ sprint: Sprint; issueId: number }];
}>();

const expanded = ref(false)

function toggleExpand() {
    if (!expanded.value) {
        emit('expand', props.sprint)
    }
    expanded.value = !expanded.value
}

const { formatDate } = useTimeFormatter()

const statusBadge: Record<string, string> = {
    [SprintStatus.Planning]:  "badge-neutral",
    [SprintStatus.Active]:    "badge-success",
    [SprintStatus.Completed]: "badge-ghost",
};
</script>

<template>
    <div class="card card-bordered bg-base-100 shadow-sm">
        <div class="card-body p-4">
            <div class="flex items-start justify-between gap-2">
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                        <span
                            :class="[
                                'badge badge-sm',
                                statusBadge[sprint.status],
                            ]"
                        >
                            {{ sprint.status }}
                        </span>
                        <h3 class="font-semibold truncate">
                            {{ sprint.name }}
                        </h3>
                    </div>
                    <p
                        v-if="sprint.goal"
                        class="text-sm text-base-content/60 line-clamp-2"
                    >
                        {{ sprint.goal }}
                    </p>
                </div>

                <div v-if="canManage" class="flex gap-1 shrink-0">
                    <button
                        v-if="sprint.status === SprintStatus.Planning && !hasActiveSprint"
                        class="btn btn-xs btn-success"
                        @click="emit('start', sprint)"
                    >
                        Start
                    </button>

                    <button
                        v-if="sprint.status === SprintStatus.Active"
                        class="btn btn-xs btn-warning"
                        @click="emit('complete', sprint)"
                    >
                        Complete
                    </button>

                    <button
                        v-if="sprint.status !== SprintStatus.Completed"
                        class="btn btn-xs btn-ghost"
                        @click="emit('edit', sprint)"
                    >
                        Edit
                    </button>

                    <button
                        v-if="sprint.status === SprintStatus.Planning"
                        class="btn btn-xs btn-error btn-outline"
                        @click="emit('delete', sprint)"
                    >
                        Delete
                    </button>
                </div>
            </div>

            <div
                v-if="sprint.start_date || sprint.end_date"
                class="text-xs text-base-content/50 mt-1"
            >
                {{ formatDate(sprint.start_date) }} → {{ formatDate(sprint.end_date) }}
            </div>

            <!-- Metrics: issue count + story point total from the API response -->
            <div
                v-if="sprint.issue_count !== undefined"
                class="text-xs text-base-content/50 mt-1"
            >
                {{ sprint.issue_count }} issue{{ sprint.issue_count !== 1 ? 's' : '' }}
                <template v-if="sprint.total_story_points !== undefined">
                    · {{ sprint.total_story_points }} pts
                </template>
            </div>

            <!-- Issue list (shown when expanded and issues are loaded) -->
            <div v-if="expanded && issues && issues.length > 0" class="mt-3 divide-y divide-base-200">
                <div
                    v-for="issue in issues"
                    :key="issue.id"
                    class="flex items-center gap-2 py-2 text-sm"
                >
                    <span class="badge badge-xs badge-outline">{{ issue.key }}</span>
                    <NuxtLink
                        :to="`/projects/${projectKey}/issues/${issue.id}`"
                        class="flex-1 truncate hover:text-primary hover:underline"
                    >
                        {{ issue.title }}
                    </NuxtLink>
                    <span class="text-base-content/50 text-xs shrink-0">{{ issue.story_points ?? '?' }} pts</span>
                    <button
                        v-if="canManage"
                        class="btn btn-xs btn-ghost text-error shrink-0"
                        title="Remove from sprint"
                        @click="emit('removeIssue', { sprint, issueId: issue.id })"
                    >
                        ✕
                    </button>
                </div>
            </div>
            <div v-else-if="expanded && issues && issues.length === 0" class="mt-3 text-sm text-base-content/50">
                No issues in this sprint.
            </div>

            <!-- Expand toggle -->
            <button
                class="btn btn-xs btn-ghost mt-2 w-full"
                @click="toggleExpand"
            >
                <span v-if="!expanded">
                    Show issues{{ issues ? ` (${issues.length})` : '' }}
                </span>
                <span v-else>Hide issues</span>
            </button>
        </div>
    </div>
</template>
