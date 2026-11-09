import { readdirSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { openCli, runCli, scratchDir } from '../testUtils'

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

describe('feature-compose show in a terminal', () => {
  it('opens the list of the starters, filters it with /, opens the view of a starter with l, and goes back with h', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show'])

    await session.waitForText('A Go service with an event store, an HTTP API, and a catchup process.')
    await session.press('/')
    await session.type('zzz')
    await session.waitForText('zzz')
    expect(await session.text()).not.toContain('A Go service with an event store')
    await session.press('escape')
    await session.waitForText('A Go service with an event store')
    await session.press('l')
    await session.waitForText('declared by the feature cli')
    const view = await session.text()
    for (const part of ['after_render: go mod tidy, gofmt -w .', 'test: task test:unit', 'go-version  default: 1.27.1', 'api.middleware', 'cli.commands  item: code  declared by the feature cli', 'cli-auth']) {
      expect(view).toContain(part)
    }
    await session.press('h')
    await session.waitForText('Starters')
    await session.press('q')
    await session.waitForText('EXIT:0')
    expect(readdirSync(dir)).toEqual([])
  })

  for (const [name, args] of [
    ['stdin is not a terminal', ['show', '<', '/dev/null']],
    ['stdout is not a terminal', ['show', '|', 'cat']],
    ['--output json asks for it', ['show', '--output', 'json']],
  ]) {
    it(`prints JSON and starts no TUI when ${name}`, async () => {
      const dir = scratchDir()
      const session = await openCli(dir, args)

      await session.waitForText('EXIT:0')
      const screen = await session.text()
      expect(screen).toContain('"starters": {')
      expect(screen).not.toContain('Starters')
    })
  }

  it('show <starter> opens the view of that starter', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'])

    await session.waitForText('declared by the feature cli')
    await session.press('q')
    await session.waitForText('EXIT:0')
  })

  it('an unknown starter stops show before the TUI starts, and the message lists the starters', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'no-such-starter'])

    await session.waitForText(/EXIT:[1-9]/)
    const screen = await session.text()
    expect(screen).toContain('unknown starter "no-such-starter": the catalog has go-event-sourcing-service')
    expect(screen).not.toContain('Starters')
  })
})
