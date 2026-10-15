import { describe, expect, it } from 'vitest'
import { lowercaseProjectKeyPath } from './projectKeyPath'

describe('lowercaseProjectKeyPath', () => {
  it('lower-cases the project key segment', () => {
    expect(lowercaseProjectKeyPath('/projects/DEMO')).toBe('/projects/demo')
    expect(lowercaseProjectKeyPath('/projects/Demo/issues/3')).toBe('/projects/demo/issues/3')
    expect(lowercaseProjectKeyPath('/projects/SDEMO/sprints')).toBe('/projects/sdemo/sprints')
  })

  it('returns null when nothing needs to change', () => {
    expect(lowercaseProjectKeyPath('/projects/demo/board')).toBeNull()
    expect(lowercaseProjectKeyPath('/projects')).toBeNull()
    expect(lowercaseProjectKeyPath('/projects/')).toBeNull()
    expect(lowercaseProjectKeyPath('/login')).toBeNull()
  })

  it('leaves the rest of the path alone, including upper-case segments after the key', () => {
    expect(lowercaseProjectKeyPath('/projects/DEMO/issues/3?x=Y#Top')).toBe('/projects/demo/issues/3?x=Y#Top')
  })

  it('does not touch other routes that merely contain "projects"', () => {
    expect(lowercaseProjectKeyPath('/system/Projects/ABC')).toBeNull()
    expect(lowercaseProjectKeyPath('/projectsX/ABC')).toBeNull()
  })
})
