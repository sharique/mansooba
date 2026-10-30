import tailwindcss from "@tailwindcss/vite";

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2025-07-15",
  devtools: { enabled: true },
  srcDir: "app",
  ssr: false,
  modules: [
    "@nuxt/eslint",
    "@nuxt/icon",
    "@pinia/nuxt",
    "pinia-plugin-persistedstate/nuxt",
  ],
  css: ["~/assets/css/main.css"],
  vite: {
    plugins: [tailwindcss()],
    // Pre-bundle the chart and drag-and-drop libraries at start-up. Otherwise the dev
    // server finds them only when the Reports or Board page first renders, re-optimises
    // and reloads mid-session, which can leave two copies of Vue loaded.
    optimizeDeps: {
      include: ["chart.js", "vue-chartjs", "vue-draggable-plus", "sortablejs"],
    },
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: "http://localhost:8080/api/v1",
    },
  },
});
