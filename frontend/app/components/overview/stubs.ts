// Test helper: NuxtLink and the time formatter are provided by Nuxt at runtime.
export const NuxtLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}
