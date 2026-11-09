import { readdirSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir } from '../testUtils'

describe('feature-compose show', () => {
  it('prints the catalog as JSON when stdout is not a terminal', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['show'])

    expect(result.status, result.stderr).toBe(0)
    const catalog = JSON.parse(result.stdout)
    const starter = catalog.starters['go-event-sourcing-service']
    expect(starter.summary).toBe('A Go service with an event store, an HTTP API, and a catchup process.')
    expect(starter.folders.go.after_render[0]).toBe('go mod tidy')
    expect(catalog.features['cli-auth'].needs).toEqual(['cli', 'api-auth'])
    expect(Object.keys(catalog.features.cli.extension_points)).toEqual(['cli.commands'])
    expect(readdirSync(dir)).toEqual([])
  })

  it('show <starter> --output json prints that starter and the features that target it', async () => {
    const dir = scratchDir()

    const all = JSON.parse((await runCli(dir, ['show'])).stdout)
    const result = await runCli(dir, ['show', 'go-event-sourcing-service', '--output', 'json'])

    expect(result.status, result.stderr).toBe(0)
    expect(JSON.parse(result.stdout)).toEqual(all)
  })

  it('an unknown starter stops show, and the message lists the starters', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['show', 'no-such-starter'])

    expect(result.status).not.toBe(0)
    expect(result.stdout).toBe('')
    expect(result.stderr).toContain('unknown starter "no-such-starter": the catalog has go-event-sourcing-service')
    expect(readdirSync(dir)).toEqual([])
  })
})
