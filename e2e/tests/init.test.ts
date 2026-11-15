import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { runCli, runProgram, scratchDir } from '../testUtils'

describe('feature-compose init', () => {
  it('writes a catalog that check --build passes, and new makes a repo from it whose script prints the greeting of the option', async () => {
    const dir = scratchDir()

    const made = await runCli(dir, ['init', 'my-catalog'])
    const checked = await runCli(dir, ['check', '--catalog', 'my-catalog', '--build'])
    const repo = await runCli(dir, ['new', '--catalog', 'my-catalog', 'example', 'my-repo', '--feature', 'greeting:language=fr'])
    const greeting = await runProgram(join(dir, 'my-repo'), './scripts/greet')

    expect(made.status, made.stderr).toBe(0)
    expect(made.stdout).toContain('Made a catalog in my-catalog with the starter example and the feature greeting.')
    expect(checked.status, checked.stdout + checked.stderr).toBe(0)
    expect(checked.stdout).toContain('The repo of each combination passes its commands.')
    expect(repo.status, repo.stderr).toBe(0)
    expect(greeting.stdout).toBe('Bonjour, my-repo!\n')
  })

  it('stops when the folder already holds a file of the catalog, and names the file', async () => {
    const dir = scratchDir()
    await runCli(dir, ['init', 'my-catalog'])

    const again = await runCli(dir, ['init', 'my-catalog'])

    expect(again.status).toBe(1)
    expect(again.stderr).toBe('my-catalog already holds .github/workflows/check.yml, so init writes no file\n')
  })
})
