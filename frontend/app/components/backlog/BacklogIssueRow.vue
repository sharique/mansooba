<script setup lang="ts">
import { SprintStatus } from "~/types/domain.types";
import type { Issue, Sprint } from "~/types/domain.types";

const props = defineProps<{
    issue: Issue;
    projectKey: string;
    canManage?: boolean;
    sprints?: Sprint[];
}>();

const emit = defineEmits<{
    "sprint-assign": [{ issueId: number; sprintId: number }];
}>();

const priorityBadge: Record<string, string> = {
    critical: "badge-error",
    high: "badge-warning",
    medium: "badge-info",
    low: "badge-ghost",
};

const typeIcon: Record<string, string> = {
    epic: "⚡",
    story: "📖",
    task: "✓",
    bug: "🐛",
};

const openSprints = computed(() =>
    (props.sprints ?? []).filter((s) => s.status !== SprintStatus.Completed),
);

function onSprintChange(e: Event) {
    const sprintId = Number((e.target as HTMLSelectElement).value);
    if (sprintId) emit("sprint-assign", { issueId: props.issue.id, sprintId });
}
</script>

<template>
    <div
        class="flex items-center gap-3 px-4 py-3 hover:bg-base-200 rounded-lg transition-colors"
    >
        <!-- Issue type icon -->
        <span
            class="text-base w-5 text-center shrink-0 cursor-pointer"
            :title="issue.type"
            @click="navigateTo(`/projects/${projectKey}/issues/${issue.id}`)"
        >
            {{ typeIcon[issue.type] ?? "·" }}
        </span>

        <!-- Title -->
        <span
            class="flex-1 text-sm truncate cursor-pointer"
            @click="navigateTo(`/projects/${projectKey}/issues/${issue.id}`)"
            >{{ issue.title }}</span
        >

        <!-- Story points -->
        <span
            v-if="issue.story_points != null"
            class="badge badge-outline badge-sm shrink-0"
            title="Story points"
        >
            {{ issue.story_points }}
        </span>

        <!-- Priority badge -->
        <span
            :class="[
                'badge badge-sm shrink-0',
                priorityBadge[issue.priority] ?? 'badge-ghost',
            ]"
        >
            {{ issue.priority }}
        </span>

        <!-- Assignee initials -->
        <div
            v-if="issue.assignee_id"
            class="avatar placeholder shrink-0"
            title="Assigned"
        >
            <div class="bg-neutral text-neutral-content rounded-full w-6">
                <span class="text-xs">{{
                    String(issue.assignee_id).slice(0, 2)
                }}</span>
            </div>
        </div>

        <!-- Add to sprint -->
        <select
            v-if="canManage && openSprints.length > 0"
            class="select select-bordered select-xs shrink-0 w-36"
            title="Add to sprint"
            @change="onSprintChange"
            @click.stop
        >
            <option value="">Add to sprint</option>
            <option v-for="s in openSprints" :key="s.id" :value="s.id">
                {{ s.name }}
            </option>
        </select>
    </div>
</template>
