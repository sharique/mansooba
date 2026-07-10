<script setup lang="ts">
import type { Sprint } from "~/types/domain.types";
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

function validate(): boolean {
    nameError.value = !form.name.trim();

    startDateError.value = !form.start_date
        ? "Start date is required"
        : form.start_date < today
          ? "Start date can't be in the past"
          : "";

    endDateError.value = !form.end_date
        ? "End date is required"
        : form.end_date < today
          ? "End date can't be in the past"
          : form.start_date && form.end_date <= form.start_date
            ? "End date must be after start date"
            : "";

    return !nameError.value && !startDateError.value && !endDateError.value;
}

async function submit() {
    if (!validate()) return;
    submitting.value = true;
    try {
        let sprint: Sprint;
        if (props.sprint) {
            const origStart = props.sprint.start_date?.slice(0, 10) ?? "";
            const origEnd = props.sprint.end_date?.slice(0, 10) ?? "";
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

            <form @submit.prevent="submit" class="flex flex-col gap-3">
                <label class="form-control">
                    <div class="label">
                        <span class="label-text">Name <span class="text-error">*</span></span>
                    </div>
                    <input
                        v-model="form.name"
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
                            type="date"
                            :min="today"
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
                            type="date"
                            :min="endDateMin"
                            class="input input-bordered w-full"
                            :class="{ 'input-error': endDateError }"
                            required
                        />
                        <div v-if="endDateError" class="label">
                            <span class="label-text-alt text-error">{{ endDateError }}</span>
                        </div>
                    </label>
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
                        :disabled="submitting"
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
