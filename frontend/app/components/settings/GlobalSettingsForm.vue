<template>
    <form class="space-y-4" @submit.prevent="save">
        <div class="form-control">
            <label class="label"
                ><span class="label-text">Organization Name</span></label
            >
            <input
                v-model="form.organization_name"
                type="text"
                class="input input-bordered"
                maxlength="100"
                required
            />
            <span
                v-if="errors.organization_name"
                class="label-text-alt text-error mt-1"
            >
                {{ errors.organization_name }}
            </span>
        </div>

        <div class="form-control">
            <label class="label"
                ><span class="label-text">Date Format</span></label
            >
            <select v-model="form.date_format" class="select select-bordered">
                <option value="YYYY-MM-DD">YYYY-MM-DD (e.g. 2026-07-07)</option>
                <option value="DD/MM/YYYY">DD/MM/YYYY (e.g. 07/07/2026)</option>
                <option value="MM/DD/YYYY">MM/DD/YYYY (e.g. 07/07/2026)</option>
                <option value="D-MMM-YYYY">D-MMM-YYYY (e.g. 7-Jul-2026)</option>
            </select>
        </div>

        <div class="form-control">
            <label class="label"
                ><span class="label-text">Time Format</span></label
            >
            <select v-model="form.time_format" class="select select-bordered">
                <option value="24h">24-hour</option>
                <option value="12h">12-hour</option>
            </select>
        </div>

        <div class="form-control">
            <label class="label"
                ><span class="label-text">Locale (BCP-47)</span></label
            >
            <input
                v-model="form.locale"
                type="text"
                class="input input-bordered"
                placeholder="e.g. en-US"
                pattern="[a-zA-Z]{2,3}(-[a-zA-Z]{2,3})?"
            />
            <span v-if="errors.locale" class="label-text-alt text-error mt-1">
                {{ errors.locale }}
            </span>
        </div>

        <div class="form-control">
            <label class="label"
                ><span class="label-text">Week Starts On</span></label
            >
            <select
                v-model="form.week_start_day"
                class="select select-bordered"
            >
                <option value="monday">Monday</option>
                <option value="sunday">Sunday</option>
            </select>
        </div>

        <div class="flex justify-end pt-2">
            <button type="submit" class="btn btn-primary" :disabled="saving">
                {{ saving ? "Saving…" : "Save Settings" }}
            </button>
        </div>
    </form>
</template>

<script setup lang="ts">
import { useGlobalSettingsStore } from "~/stores/global-settings.store";

const store = useGlobalSettingsStore();
const { showSuccess, showError } = useToast();

const form = reactive({
    organization_name: store.organization_name,
    date_format: store.date_format,
    time_format: store.time_format,
    locale: store.locale,
    week_start_day: store.week_start_day,
});

const errors = reactive<Record<string, string>>({});
const saving = ref(false);

watch(
    () => store.loaded,
    (loaded) => {
        if (loaded) {
            form.organization_name = store.organization_name;
            form.date_format = store.date_format;
            form.time_format = store.time_format;
            form.locale = store.locale;
            form.week_start_day = store.week_start_day;
        }
    },
    { immediate: true },
);

function validate(): boolean {
    Object.keys(errors).forEach((k) => delete errors[k]);
    if (!form.organization_name.trim()) {
        errors.organization_name = "Organization name is required.";
        return false;
    }
    if (form.organization_name.length > 100) {
        errors.organization_name = "Must be 100 characters or fewer";
        return false;
    }
    const bcp47 = /^[a-zA-Z]{2,3}(-[a-zA-Z]{2,3})?$/;
    if (!bcp47.test(form.locale)) {
        errors.locale = "Must be a valid BCP-47 locale tag (e.g. en-US)";
        return false;
    }
    return true;
}

async function save() {
    if (!validate()) return;
    saving.value = true;
    try {
        await store.patch({
            organization_name: form.organization_name,
            date_format: form.date_format,
            time_format: form.time_format,
            locale: form.locale,
            week_start_day: form.week_start_day,
        });
        showSuccess("Settings saved");
    } catch {
        showError("Failed to save settings");
    } finally {
        saving.value = false;
    }
}
</script>
