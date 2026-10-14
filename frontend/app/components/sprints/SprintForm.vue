<script setup lang="ts">
import type { Sprint } from "~/types/domain.types";
import { isOverdue, overlappingSprints } from "~/utils/sprintDates";
import type {
    CreateSprintPayload,
    UpdateSprintPayload,
} from "~/services/sprints.service";

const props = defineProps<{
    projectKey: string;
    sprint?: Sprint;
}>();

const emit = defineEmits<{
    saved: [sprint: Sprint];
    cancel: [];
}>();

const sprintsStore = useSprintsStore();
const { showSuccess, showError } = useToast();

const form = reactive({
    name: props.sprint?.name ?? "",
    goal: props.sprint?.goal ?? "",
    start_date: props.sprint?.start_date?.slice(0, 10) ?? "",
    end_date: props.sprint?.end_date?.slice(0, 10) ?? "",
});
const submitting = ref(false);

// Today in YYYY-MM-DD — `<input type="date">` values compare correctly as
// plain strings in this format, no Date parsing needed.
const today = new Date().toISOString().slice(0, 10);

const nameError = ref(false);
const startDateError = ref("");
const endDateError = ref("");

// End date's picker can't go earlier than start date (or today, before a
// start date is chosen) — matches the "end after start" / "future only"
// rules enforced on submit below.
const endDateMin = computed(() => form.start_date || today);

const MAX_SPRINT_DAYS = 28;
// Latest allowed end date for a start date (YYYY-MM-DD, UTC so no timezone drift).
function latestEnd(start: string): string {
    const d = new Date(`${start}T00:00:00Z`);
    d.setUTCDate(d.getUTCDate() + MAX_SPRINT_DAYS);
    return d.toISOString().slice(0, 10);
}
function prettyDate(iso: string): string {
    return new Date(`${iso}T00:00:00Z`).toLocaleDateString("en-GB", {
        day: "numeric", month: "short", year: "numeric", timeZone: "UTC",
    });
}
const endDateMax = computed(() => (form.start_date ? latestEnd(form.start_date) : undefined));

// What may be edited depends on the sprint's state: planning = everything,
// active = the end date only, completed = nothing (the server enforces the same).
const status = props.sprint?.status;
const locked = {
    name: status === "active" || status === "completed",
    goal: status === "active" || status === "completed",
    start: status === "active" || status === "completed",
    end: status === "completed",
};
// Warnings never block saving: overlapping another open sprint, or an
// in-progress sprint whose end date has passed.
const overlaps = computed(() =>
    overlappingSprints(form.start_date, form.end_date, sprintsStore.sprints ?? [], props.sprint?.id),
);
const overdueNote = computed(() =>
    props.sprint && isOverdue({ status: props.sprint.status, end_date: form.end_date || null }),
);
const lockNote =
    status === "active"
        ? "This sprint has started, so only the end date can be changed."
        : status === "completed"
          ? "This sprint is completed and cannot be changed."
          : "";

// A sprint that already started keeps its original start/end dates even though
// they are now in the past; "can't be in the past" applies only to a date the
// user actually changes (or to any date when creating).
const origStart = props.sprint?.start_date?.slice(0, 10) ?? "";
const origEnd = props.sprint?.end_date?.slice(0, 10) ?? "";
const startUnchanged = () => !!props.sprint && form.start_date === origStart;
const endUnchanged = () => !!props.sprint && form.end_date === origEnd;
const startDateMin = computed(() => (startUnchanged() ? undefined : today));

function validate(): boolean {
    nameError.value = !form.name.trim();

    startDateError.value = !form.start_date
        ? "Start date is required"
        : form.start_date < today && !startUnchanged()
          ? "Start date can't be in the past"
          : "";

    endDateError.value = !form.end_date
        ? "End date is required"
        : form.end_date < today && !endUnchanged()
          ? "End date can't be in the past"
          : form.start_date && form.end_date <= form.start_date
            ? "End date must be after start date"
            : form.start_date && form.end_date > latestEnd(form.start_date)
              ? `A sprint can't be longer than ${MAX_SPRINT_DAYS} days (4 weeks). For this start date the latest end date is ${prettyDate(latestEnd(form.start_date))}.`
              : "";

    return !nameError.value && !startDateError.value && !endDateError.value;
}

async function submit() {
    if (status === "completed") return;
    if (!validate()) return;
    submitting.value = true;
    try {
        let sprint: Sprint;
        if (props.sprint) {
            const payload: UpdateSprintPayload = {};
            if (form.name !== props.sprint.name) payload.name = form.name;
            if (form.goal !== props.sprint.goal) payload.goal = form.goal;
            if (form.start_date !== origStart)
                payload.start_date = form.start_date
                    ? `${form.start_date}T00:00:00Z`
                    : null;
            if (form.end_date !== origEnd)
                payload.end_date = form.end_date
                    ? `${form.end_date}T00:00:00Z`
                    : null;
            sprint = await sprintsStore.updateSprint(
                props.projectKey,
                props.sprint.id,
                payload,
            );
            showSuccess("Sprint updated");
        } else {
            const payload: CreateSprintPayload = {
                name: form.name,
                goal: form.goal || undefined,
                start_date: form.start_date
                    ? convertToRFC3339Date(form.start_date)
                    : undefined,
                end_date: form.end_date
                    ? convertToRFC3339Date(form.end_date)
                    : undefined,
            };
            sprint = await sprintsStore.createSprint(props.projectKey, payload);
            showSuccess("Sprint created");
        }
        emit("saved", sprint);
    } catch (e: any) {
        showError(e.data?.message ?? "Failed to save sprint");
    } finally {
        submitting.value = false;
    }
}

// Converting date to rfc3339Date.
function convertToRFC3339Date(mdate: string) {
    // Create a Date object
    const date = new Date(mdate);

    // Convert to RFC3339 format (ISO 8601 is compatible with RFC3339)
    const rfc3339Date = date.toISOString();

    return rfc3339Date;
}
</script>

<template>
    <dialog class="modal modal-open">
        <div class="modal-box">
            <h3 class="font-bold text-lg mb-4">
                {{ sprint ? "Edit Sprint" : "Create Sprint" }}
            </h3>

            <form novalidate @submit.prevent="submit" class="flex flex-col gap-3">
                <p v-if="lockNote" class="text-sm text-base-content/70">{{ lockNote }}</p>
                <div v-if="overdueNote" class="alert alert-warning py-2 text-sm">
                    <span>This sprint is in progress but its end date has passed. Complete it, or move the end date.</span>
                </div>
                <label class="form-control">
                    <div class="label">
                        <span class="label-text">Name <span class="text-error">*</span></span>
                    </div>
                    <input
                        v-model="form.name"
                        :disabled="locked.name"
                        type="text"
                        placeholder="Sprint 1"
                        class="input input-bordered w-full"
                        :class="{ 'input-error': nameError }"
                        required
                    />
                    <div v-if="nameError" class="label">
                        <span class="label-text-alt text-error">Sprint name is required</span>
                    </div>
                </label>

                <label class="form-control">
                    <div class="label">
                        <span class="label-text">Goal</span>
                    </div>
                    <textarea
                        v-model="form.goal"
                        :disabled="locked.goal"
                        class="textarea textarea-bordered w-full"
                        placeholder="What does this sprint aim to achieve?"
                        rows="2"
                    />
                </label>

                <div class="grid grid-cols-2 gap-3">
                    <label class="form-control">
                        <div class="label">
                            <span class="label-text">Start date <span class="text-error">*</span></span>
                        </div>
                        <input
                            v-model="form.start_date"
                            :disabled="locked.start"
                            type="date"
                            :min="startDateMin"
                            class="input input-bordered w-full"
                            :class="{ 'input-error': startDateError }"
                            required
                        />
                        <div v-if="startDateError" class="label">
                            <span class="label-text-alt text-error">{{ startDateError }}</span>
                        </div>
                    </label>
                    <label class="form-control">
                        <div class="label">
                            <span class="label-text">End date <span class="text-error">*</span></span>
                        </div>
                        <input
                            v-model="form.end_date"
                            :disabled="locked.end"
                            type="date"
                            :min="endDateMin"
                            :max="endDateMax"
                            class="input input-bordered w-full"
                            :class="{ 'input-error': endDateError }"
                            required
                        />
                        <div v-if="endDateError" class="label">
                            <span class="label-text-alt text-error">{{ endDateError }}</span>
                        </div>
                    </label>
                </div>

                <div v-if="overlaps.length" class="alert alert-warning py-2 text-sm">
                    <span>
                        These dates overlap
                        <template v-for="(o, i) in overlaps" :key="o.id">
                            <strong>{{ o.name }}</strong> ({{ prettyDate(o.start_date!.slice(0, 10)) }} – {{ prettyDate(o.end_date!.slice(0, 10)) }})<template v-if="i < overlaps.length - 1">, </template>
                        </template>.
                        You can still save.
                    </span>
                </div>

                <div class="modal-action mt-2">
                    <button
                        type="button"
                        class="btn btn-ghost"
                        @click="emit('cancel')"
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        class="btn btn-primary"
                        :disabled="submitting || status === 'completed'"
                    >
                        <span
                            v-if="submitting"
                            class="loading loading-spinner loading-sm"
                        />
                        {{ sprint ? "Save" : "Create" }}
                    </button>
                </div>
            </form>
        </div>
        <div class="modal-backdrop" @click="emit('cancel')" />
    </dialog>
</template>
