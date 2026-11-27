import { readdirSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { openCli, runCli, scratchDir } from '../testUtils'

describe('feature-compose show', () => {
  it('prints the catalog as JSON when stdout is not a terminal', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['show'])

    expect(result.status, result.stderr).toBe(0)
    const catalog = JSON.parse(result.stdout)
    const starter = catalog.starters['hpcsc/go-event-sourcing-service']
    expect(starter.summary).toBe('A Go service with an event store, an HTTP API, and a catchup process.')
    expect(starter.folders.go.after_render[0]).toBe('go mod tidy')
    expect(catalog.features['hpcsc/cli-auth'].needs).toEqual({ 'hpcsc/cli': {}, 'hpcsc/api-auth': {} })
    expect(catalog.features['hpcsc/cli-auth'].options).toEqual({ 'store-key': { default: 'false', pattern: '', values: ['true', 'false'], when: 'api-auth.method == "api-key"' } })
    expect(Object.keys(catalog.features['hpcsc/cli'].extension_points)).toEqual(['cli.commands'])
    expect(readdirSync(dir)).toEqual([])
  })

  it('show <starter> --output json prints that starter and the features that target it', async () => {
    const dir = scratchDir()

    const all = JSON.parse((await runCli(dir, ['show'])).stdout)
    const result = await runCli(dir, ['show', 'go-event-sourcing-service', '--output', 'json'])

    expect(result.status, result.stderr).toBe(0)
    expect(JSON.parse(result.stdout)).toEqual({
      starters: { 'hpcsc/go-event-sourcing-service': all.starters['hpcsc/go-event-sourcing-service'] },
      features: Object.fromEntries(
        Object.entries(all.features).filter(([, feature]) => (feature as { targets: string[] }).targets.includes('hpcsc/go-event-sourcing-service')),
      ),
    })
  })

  it('an unknown starter stops show, and the message lists the starters', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['show', 'no-such-starter'])

    expect(result.status).not.toBe(0)
    expect(result.stdout).toBe('')
    expect(result.stderr).toContain('unknown starter "no-such-starter": the catalog has hpcsc/chi-api, hpcsc/go, hpcsc/go-event-sourcing-service')
    expect(readdirSync(dir)).toEqual([])
  })
})

describe('feature-compose show in a terminal', () => {
  const rowsOfTheWholeTree = 73

  it('opens the tree with each starter open and the details of the first starter, chi-api, and filters it with /', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show'], {}, rowsOfTheWholeTree)

    await session.waitForText('A Go HTTP API with chi, whose routes are under /api.', { timeout: 15_000 })
    const screen = await session.text()
    for (const part of ['╭─ Catalog ', '1 catalog · 4 starters', '▾ hpcsc', '╭─ chi-api ─', '▾ go', '▾ go-event-sourcing-service', '├─ ▾ features', '│  ├─ api-auth', '│  ├─ cli-auth', '│  ├─ process-manager', '│  └─ push', '└─ ▾ extension points', '├─ api.middleware', '├─ cli.commands', '└─ use_cases.handlers', 'after_render  go mod tidy', 'test          task test:unit', 'go-version  default  1.27.1']) {
      expect(screen).toContain(part)
    }
    await session.press('/')
    await session.type('zzz')
    await session.waitForText('No name matches the filter.')
    await session.press('escape')
    await session.waitForText('▾ go-event-sourcing-service')
    await session.press('q')
    await session.waitForText('EXIT:0')
    expect(readdirSync(dir)).toEqual([])
  })

  it('closes the selected starter with h, and opens it with l', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'])

    await session.waitForText('▾ go-event-sourcing-service', { timeout: 15_000 })
    await session.press('h')
    await session.waitForText('▸ go-event-sourcing-service')
    expect(await session.text()).not.toContain('process-manager')
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

      await session.waitForText('EXIT:0', { timeout: 15_000 })
      const screen = await session.text()
      expect(screen).toContain('"starters": {')
      expect(screen).not.toContain('╭─ Catalog')
    })
  }

  it('shows the details of cli-auth when j selects it, and h selects its group', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'])

    await session.waitForText('└─ push', { timeout: 15_000 })
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
    const session = await openCli(dir, ['show', 'go-event-sourcing-service'], {}, rowsOfTheWholeTree)

    await session.waitForText('└─ use_cases.handlers', { timeout: 15_000 })
    for (let i = 0; i < 11; i++) {
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

    await session.waitForText('▾ go-event-sourcing-service', { timeout: 15_000 })
    expect(await session.text()).toContain('╭─ go-event-sourcing-service ')
    await session.press('q')
    await session.waitForText('EXIT:0')
  })

  it('an unknown starter stops show before the TUI starts, and the message lists the starters', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['show', 'no-such-starter'])

    await session.waitForText(/EXIT:[1-9]/, { timeout: 15_000 })
    const screen = await session.text()
    expect(screen).toContain('unknown starter "no-such-starter": the catalog has hpcsc/chi-api, hpcsc/go, hpcsc/go-event-sourcing-service')
    expect(screen).not.toContain('╭─ Catalog')
  })
})
