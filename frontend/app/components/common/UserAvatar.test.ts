// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test, vi } from 'vitest'
import UserAvatar from './UserAvatar.vue'

// Mock the composable so tests are independent of its implementation
vi.mock('~/composables/useAvatarColor', () => ({
  useAvatarColor: () => 'oklch(0.55 0.14 192)',
}))

describe('UserAvatar', () => {
  test('renders <img> when avatarUrl is set', () => {
    const wrapper = mount(UserAvatar, {
      props: { avatarUrl: 'https://example.com/avatar.jpg', name: 'Test User', userId: 1 },
    })
    expect(wrapper.find('img').exists()).toBe(true)
    expect(wrapper.find('img').attributes('src')).toBe('https://example.com/avatar.jpg')
  })

  test('renders initials div without avatarUrl', () => {
    const wrapper = mount(UserAvatar, {
      props: { name: 'Test User', userId: 1 },
    })
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('.avatar-initials').exists()).toBe(true)
  })

  test('falls back to initials on img onerror', async () => {
    const wrapper = mount(UserAvatar, {
      props: { avatarUrl: '/broken.jpg', name: 'Test User', userId: 1 },
    })
    expect(wrapper.find('img').exists()).toBe(true)
    await wrapper.find('img').trigger('error')
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('.avatar-initials').exists()).toBe(true)
  })

  test('derives "TU" from "Test User"', () => {
    const wrapper = mount(UserAvatar, {
      props: { name: 'Test User', userId: 1 },
    })
    expect(wrapper.find('.avatar-initials').text()).toBe('TU')
  })

  test('derives "MA" from "Madonna"', () => {
    const wrapper = mount(UserAvatar, {
      props: { name: 'Madonna', userId: 1 },
    })
    expect(wrapper.find('.avatar-initials').text()).toBe('MA')
  })

  test('derives "AB" from empty name using email "ab@example.com"', () => {
    const wrapper = mount(UserAvatar, {
      props: { name: '', email: 'ab@example.com', userId: 1 },
    })
    expect(wrapper.find('.avatar-initials').text()).toBe('AB')
  })

  test('applies size prop class "sm"', () => {
    const wrapper = mount(UserAvatar, {
      props: { name: 'Test User', userId: 1, size: 'sm' },
    })
    expect(wrapper.find('[class*="avatar-sm"]').exists() || wrapper.html().includes('avatar-sm')).toBe(true)
  })

  test('applies size prop class "lg"', () => {
    const wrapper = mount(UserAvatar, {
      props: { name: 'Test User', userId: 1, size: 'lg' },
    })
    expect(wrapper.find('[class*="avatar-lg"]').exists() || wrapper.html().includes('avatar-lg')).toBe(true)
  })
})
