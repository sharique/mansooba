<script setup lang="ts">
import { SprintStatus } from "~/types/domain.types";
import type { Issue, Sprint } from "~/types/domain.types";
import UserAvatar from "~/components/common/UserAvatar.vue";
import { priorityBadgeClass, typeIconName, typeIconClass } from "~/utils/issueStyles";

const props = defineProps<{
    issue: Issue;
    projectKey: string;
    canManage?: boolean;
    sprints?: Sprint[];
}>();

const emit = defineEmits<{
    "sprint-assign": [{ issueId: number; sprintId: number }];
}>();

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
            class="w-4 h-4 shrink-0 cursor-pointer flex items-center"
            :title="issue.type"
            @click="navigateTo(`/projects/${projectKey}/issues/${issue.id}`)"
        >
            <Icon :name="typeIconName(issue.type)" class="w-4 h-4" :class="typeIconClass(issue.type)" />
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
        <span :class="['badge badge-sm shrink-0', priorityBadgeClass(issue.priority)]">
            {{ issue.priority }}
        </span>

        <!-- Assignee avatar -->
        <UserAvatar
            v-if="issue.assignee_id"
            :avatarUrl="issue.assignee_avatar_url || undefined"
            :name="issue.assignee_name || ''"
            :userId="issue.assignee_id"
            size="sm"
            class="shrink-0"
            title="Assigned"
        />

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
