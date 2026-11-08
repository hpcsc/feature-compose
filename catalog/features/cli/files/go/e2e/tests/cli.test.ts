import { describe, expect, it } from 'vitest'
import { buildTag, runCli, scratchDir } from '../testUtils'

describe('the root command', () => {
  it('with no command, prints its commands and reads no input', async () => {
    const result = await runCli(scratchDir())

    expect(result.status).toBe(0)
    expect(result.stdout).toContain('print the tag {{ kebab .Name }} was built from')
    expect(result.stdout).toContain('replace {{ kebab .Name }} with the latest release')
  })
})

describe('{{ kebab .Name }} version', () => {
  it('prints the tag the binary was built with', async () => {
    const result = await runCli(scratchDir(), ['version'])

    expect(result.status).toBe(0)
    expect(result.stdout.trim()).toBe(buildTag())
  })

  it('--version prints the name and the tag', async () => {
    const result = await runCli(scratchDir(), ['--version'])

    expect(result.status).toBe(0)
    expect(result.stdout.trim()).toBe(`{{ kebab .Name }} version ${buildTag()}`)
  })
})
