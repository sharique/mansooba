import * as vue from 'vue'

// Expose Vue's reactivity API as globals to match Nuxt's auto-import behaviour
Object.assign(globalThis, vue)
