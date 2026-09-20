<template>
    <form class="space-y-4" @submit.prevent="save">
        <div class="form-control">
            <label class="label"><span class="label-text">Organization Name</span></label>
            <input
                v-model="form.organization_name"
                type="text"
                class="input input-bordered w-full"
                maxlength="100"
                required
            />
            <label v-if="errors.organization_name" class="label">
                <span class="label-text-alt text-error">{{ errors.organization_name }}</span>
            </label>
        </div>

        <div class="form-control">
            <label class="label"><span class="label-text">Date Format</span></label>
            <select v-model="form.date_format" class="select select-bordered w-full">
                <option value="YYYY-MM-DD">YYYY-MM-DD (e.g. 2026-07-07)</option>
                <option value="DD/MM/YYYY">DD/MM/YYYY (e.g. 07/07/2026)</option>
                <option value="MM/DD/YYYY">MM/DD/YYYY (e.g. 07/07/2026)</option>
                <option value="D-MMM-YYYY">D-MMM-YYYY (e.g. 7-Jul-2026)</option>
            </select>
        </div>

        <div class="form-control">
            <label class="label"><span class="label-text">Time Format</span></label>
            <select v-model="form.time_format" class="select select-bordered w-full">
                <option value="24h">24-hour</option>
                <option value="12h">12-hour</option>
            </select>
        </div>

        <div class="form-control">
            <label class="label"><span class="label-text">Locale (BCP-47)</span></label>
            <input
                v-model="form.locale"
                type="text"
                class="input input-bordered w-full"
                placeholder="e.g. en-US"
                pattern="[a-zA-Z]{2,3}(-[a-zA-Z]{2,3})?"
            />
            <label v-if="errors.locale" class="label">
                <span class="label-text-alt text-error">{{ errors.locale }}</span>
            </label>
        </div>

        <div class="form-control">
            <label class="label"><span class="label-text">Week Starts On</span></label>
            <select v-model="form.week_start_day" class="select select-bordered w-full">
                <option value="monday">Monday</option>
                <option value="sunday">Sunday</option>
            </select>
        </div>

        <div class="form-control">
            <label class="label"><span class="label-text">System Log Retention (days)</span></label>
            <input
                v-model="form.system_log_retention_days"
                type="number"
                min="1"
                step="1"
                class="input input-bordered w-full"
            />
            <label class="label">
                <span class="label-text-alt">
                    How long entries on the
                    <NuxtLink to="/system/logs" class="link">System Logs</NuxtLink>
                    page are kept before being purged.
                </span>
            </label>
            <label v-if="errors.system_log_retention_days" class="label">
                <span class="label-text-alt text-error">{{ errors.system_log_retention_days }}</span>
            </label>
        </div>

        <div class="form-control">
            <label class="label cursor-pointer justify-start gap-3">
                <input
                    type="checkbox"
                    data-testid="demo-banner-enabled"
                    class="checkbox"
                    :checked="form.demo_banner_enabled === 'true'"
                    @change="form.demo_banner_enabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
                />
                <span class="label-text">Show demo-instance banner</span>
            </label>
        </div>

        <div class="form-control">
            <label class="label"><span class="label-text">Demo Banner Message</span></label>
            <textarea
                v-model="form.demo_banner_message"
                data-testid="demo-banner-message"
                class="textarea textarea-bordered w-full"
                maxlength="280"
                rows="2"
            />
            <label v-if="errors.demo_banner_message" class="label">
                <span data-testid="demo-banner-message-error" class="label-text-alt text-error">{{ errors.demo_banner_message }}</span>
            </label>
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
    system_log_retention_days: store.system_log_retention_days,
    demo_banner_enabled: store.demo_banner_enabled,
    demo_banner_message: store.demo_banner_message,
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
            form.system_log_retention_days = store.system_log_retention_days;
            form.demo_banner_enabled = store.demo_banner_enabled;
            form.demo_banner_message = store.demo_banner_message;
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
    const retentionDays = Number(form.system_log_retention_days);
    if (!Number.isInteger(retentionDays) || retentionDays < 1) {
        errors.system_log_retention_days = "Must be a whole number of days, 1 or greater";
        return false;
    }
    if (form.demo_banner_enabled === "true" && !form.demo_banner_message.trim()) {
        errors.demo_banner_message = "A message is required while the banner is enabled.";
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
            system_log_retention_days: form.system_log_retention_days,
            demo_banner_enabled: form.demo_banner_enabled,
            demo_banner_message: form.demo_banner_message,
        });
        showSuccess("Settings saved");
    } catch {
        showError("Failed to save settings");
    } finally {
        saving.value = false;
    }
}
</script>
