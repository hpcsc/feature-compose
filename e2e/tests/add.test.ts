import { describe, expect, it } from 'vitest'
import { runCli, scratchDir } from '../testUtils'

describe('feature-compose add', () => {
  it('adds a starter and a feature that targets it and the starter of init, and check --build passes on each of their repos', async () => {
    const dir = scratchDir()
    await runCli(dir, ['init', 'my-catalog'])

    const starter = await runCli(dir, ['add', 'starter', 'service', '--catalog', 'my-catalog'])
    const feature = await runCli(dir, ['add', 'feature', 'metrics', '--target', 'example', '--target', 'service', '--catalog', 'my-catalog'])
    const checked = await runCli(dir, ['check', '--catalog', 'my-catalog', '--build'])

    expect(starter.status, starter.stderr).toBe(0)
    expect(feature.status, feature.stderr).toBe(0)
    expect(feature.stdout).toContain('  features/metrics/feature.yml\n')
    expect(checked.status, checked.stdout + checked.stderr).toBe(0)
    expect(checked.stdout).toContain('ok    service --feature metrics')
  })

  it('stops on a target that is not a starter of the catalog, and names the starters', async () => {
    const dir = scratchDir()
    await runCli(dir, ['init', 'my-catalog'])

    const result = await runCli(dir, ['add', 'feature', 'metrics', '--target', 'nope', '--catalog', 'my-catalog'])

    expect(result.status).toBe(1)
    expect(result.stderr).toBe('unknown starter "nope": the catalog has example\n')
  })
})
