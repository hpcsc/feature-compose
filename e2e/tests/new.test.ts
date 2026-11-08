import { readdirSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir } from '../testUtils'

describe('feature-compose new', () => {
  it('an unknown starter stops new, and the message lists the starters in the binary', async () => {
    const folder = scratchDir()

    const result = await runCli(folder, ['new', 'no-such-starter', 'x'], { SCAFFOLD_DIR: '' })

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('go-event-sourcing-service')
    expect(readdirSync(folder)).toEqual([])
  })

  it('a value that does not match the pattern of its option stops new, names the option and the pattern, and makes no folder', async () => {
    const folder = scratchDir()

    const result = await runCli(folder, ['new', 'go-event-sourcing-service', 'x', '--option', 'go-version=1.20'])

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('go-version')
    expect(result.stdout + result.stderr).toContain('^1\\.(2[7-9]|[3-9][0-9])(\\.[0-9]+)?$')
    expect(readdirSync(folder)).toEqual([])
  })
})
