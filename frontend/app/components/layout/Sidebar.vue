<script lang="ts">
export interface NavLink {
    label: string;
    to: string;
    icon: string;
}

export function projectNavLinks(key: string): NavLink[] {
    return [
        {
            label: "Overview",
            to: `/projects/${key}`,
            icon: "mdi:view-dashboard-outline",
        },
        {
            label: "Board",
            to: `/projects/${key}/board`,
            icon: "mdi:view-column-outline",
        },
        {
            label: "Backlog",
            to: `/projects/${key}/backlog`,
            icon: "mdi:format-list-bulleted",
        },
        // Sprints live on the backlog page in this codebase; update to a dedicated route when one exists.
        {
            label: "Sprints",
            to: `/projects/${key}/backlog`,
            icon: "mdi:run-fast",
        },
        {
            label: "Reports",
            to: `/projects/${key}/reports`,
            icon: "mdi:chart-line",
        },
        {
            label: "Settings",
            to: `/projects/${key}/settings`,
            icon: "mdi:cog-outline",
        },
    ];
}
</script>

<script setup lang="ts">
import { useProjectsStore } from "~/stores/projects.store";
import { useAuthStore } from "~/stores/auth.store";

const route = useRoute();
const projectsStore = useProjectsStore();
const authStore = useAuthStore();

const primary: NavLink[] = [
    { label: "My Desk", to: "/", icon: "mdi:monitor-dashboard" },
    { label: "Projects", to: "/projects", icon: "mdi:folder-multiple-outline" },
    { label: "Reports", to: "/reports", icon: "mdi:chart-box-outline" },
];

const systemLinks: NavLink[] = [
    {
        label: "System Settings",
        to: "/system/settings",
        icon: "mdi:cog-outline",
    },
    {
        label: "User Management",
        to: "/system/users",
        icon: "mdi:account-group-outline",
    },
    {
        label: "Create User",
        to: "/system/createuser",
        icon: "mdi:account-plus-outline",
    },
];

const currentKey = computed(() =>
    typeof route.params.key === "string" ? route.params.key : null,
);

const projectLinks = computed(() =>
    currentKey.value ? projectNavLinks(currentKey.value) : [],
);

const recentProjects = computed(() => projectsStore.projects.slice(0, 5));

// Build the set of active project links using only the FIRST link per unique URL,
// so duplicate-URL entries (Backlog + Sprints both at /backlog) don't both highlight.
const activeProjectLinks = computed<Set<NavLink>>(() => {
    const seen = new Set<string>();
    const active = new Set<NavLink>();
    for (const link of projectLinks.value) {
        if (link.to === route.path && !seen.has(link.to)) {
            active.add(link);
        }
        seen.add(link.to);
    }
    return active;
});

function isPrimaryActive(to: string): boolean {
    return route.path === to;
}

// Warm the projects list in the background so "Recent projects" shows up
// even when the user hasn't visited /projects yet in this session.
onMounted(async () => {
    if (projectsStore.projects.length === 0) {
        try {
            await projectsStore.fetchAll();
        } catch {
            /* decorative section — fail silently */
        }
    }
});
</script>

<template>
    <aside class="w-60 bg-neutral text-neutral-content flex flex-col h-full">
        <!-- brand -->
        <NuxtLink
            to="/"
            class="flex items-center gap-2 px-4 h-14 font-bold text-lg shrink-0"
        >
            <span class="inline-flex items-center justify-center w-6 h-6 rounded-md bg-primary text-primary-content text-xs font-black select-none">M</span>
            Mansooba
        </NuxtLink>

        <nav class="flex-1 overflow-y-auto px-2 pb-4 space-y-1">
            <NuxtLink
                v-for="link in primary"
                :key="link.to"
                :to="link.to"
                class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors"
                :class="
                    isPrimaryActive(link.to)
                        ? 'bg-primary text-primary-content font-semibold'
                        : 'hover:bg-base-content/10'
                "
            >
                <Icon :name="link.icon" class="w-5 h-5 opacity-90" />
                {{ link.label }}
            </NuxtLink>

            <!-- system section (admin only) -->
            <template v-if="authStore.isAdmin">
                <div
                    class="px-3 pt-3 pb-1 mt-2 text-[10px] uppercase tracking-wide opacity-50 border-t border-neutral-content/10"
                >
                    System
                </div>
                <NuxtLink
                    v-for="link in systemLinks"
                    :key="link.to"
                    :to="link.to"
                    class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors"
                    :class="
                        isPrimaryActive(link.to)
                            ? 'bg-primary text-primary-content font-semibold'
                            : 'hover:bg-base-content/10'
                    "
                >
                    <Icon :name="link.icon" class="w-5 h-5 opacity-90" />
                    {{ link.label }}
                </NuxtLink>
            </template>

            <!-- contextual project section -->
            <template v-if="projectLinks.length">
                <div
                    class="px-3 pt-3 pb-1 mt-2 text-[10px] uppercase tracking-wide opacity-50 border-t border-neutral-content/10"
                >
                    {{ currentKey }}
                </div>
                <NuxtLink
                    v-for="link in projectLinks"
                    :key="link.to + link.label"
                    :to="link.to"
                    class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors"
                    :class="
                        activeProjectLinks.has(link)
                            ? 'bg-primary text-primary-content font-semibold'
                            : 'hover:bg-base-content/10'
                    "
                >
                    <Icon :name="link.icon" class="w-5 h-5 opacity-90" />
                    {{ link.label }}
                </NuxtLink>
            </template>

            <!-- recent projects -->
            <template v-if="recentProjects.length">
                <div
                    class="px-3 pt-3 pb-1 mt-2 text-[10px] uppercase tracking-wide opacity-50 border-t border-neutral-content/10"
                >
                    Recent projects
                </div>
                <NuxtLink
                    v-for="p in recentProjects"
                    :key="p.id"
                    :to="`/projects/${p.key}`"
                    class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm hover:bg-base-content/10 transition-colors"
                >
                    <span class="w-2 h-2 rounded-full bg-accent shrink-0" />
                    <span class="truncate">{{ p.name }}</span>
                </NuxtLink>
            </template>
        </nav>
    </aside>
</template>
