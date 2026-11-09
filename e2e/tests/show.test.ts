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
  it('opens the tree with each starter open and the details of the first starter, filters it with /, closes the starter with h, and opens it with l', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show'])

    await session.waitForText('A Go service with an event store, an HTTP API, and a catchup process.')
    const screen = await session.text()
    for (const part of ['╭─ Catalog ', '▾ go-event-sourcing-service', '├─ ▾ features', '│  ├─ api-auth', '│  └─ cli-auth', '└─ ▾ extension points', '├─ api.middleware', '└─ cli.commands', 'after_render  go mod tidy', 'test          task test:unit', 'go-version  default  1.27.1']) {
      expect(screen).toContain(part)
    }
    await session.press('/')
    await session.type('zzz')
    await session.waitForText('No name matches the filter.')
    await session.press('escape')
    await session.waitForText('▾ go-event-sourcing-service')
    await session.press('h')
    await session.waitForText('▸ go-event-sourcing-service')
    expect(await session.text()).not.toContain('api-auth')
    await session.press('l')
    await session.waitForText('└─ ▾ extension points')
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
      expect(screen).not.toContain('╭─ Catalog')
    })
  }

  it('shows the details of cli-auth when j selects it, and h selects its group', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'])

    await session.waitForText('└─ cli-auth')
    for (let i = 0; i < 4; i++) {
      await session.press('j')
    }
    await session.waitForText('╭─ cli-auth ')
    const details = await session.text()
    for (const part of ['Needs', '  api-auth', 'Extension points that it adds items to', '  cli.commands']) {
      expect(details).toContain(part)
    }
    await session.press('h')
    await session.waitForText('╭─ go-event-sourcing-service › features ')
    await session.press('q')
    await session.waitForText('EXIT:0')
    expect(readdirSync(dir)).toEqual([])
  })

  it('shows the contract of api.middleware and the features that add items to it', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'])

    await session.waitForText('└─ cli.commands')
    for (let i = 0; i < 7; i++) {
      await session.press('j')
    }
    await session.waitForText('╭─ api.middleware ')
    const details = await session.text()
    for (const part of ['An expression of type func(http.Handler) http.Handler', 'item         code, import', 'declared by  the starter', 'added to by  api-auth']) {
      expect(details).toContain(part)
    }
    await session.press('q')
    await session.waitForText('EXIT:0')
  })

  it('show <starter> opens the TUI with that starter selected', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'])

    await session.waitForText('▾ go-event-sourcing-service')
    expect(await session.text()).toContain('╭─ go-event-sourcing-service ')
    await session.press('q')
    await session.waitForText('EXIT:0')
  })

  it('an unknown starter stops show before the TUI starts, and the message lists the starters', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'no-such-starter'])

    await session.waitForText(/EXIT:[1-9]/)
    const screen = await session.text()
    expect(screen).toContain('unknown starter "no-such-starter": the catalog has go-event-sourcing-service')
    expect(screen).not.toContain('╭─ Catalog')
  })
})
