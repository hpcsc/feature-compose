import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir } from '../testUtils'

function writeFile(path: string, contents: string): void {
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, contents)
}

describe('feature-compose check', () => {
  it('finds no mistake in the catalog of this repo, and exits with 0', async () => {
    const result = await runCli(scratchDir(), ['check'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toBe('The catalog has no mistakes.\n')
  })

  it('names the starter, the feature and the extension point of an add_to that nothing declares, and exits with a status that is not zero', async () => {
    const catalogDir = scratchDir()
    writeFile(join(catalogDir, 'starters', 'service', 'starter.yml'), 'folders:\n  go:\n    path: .\n')
    writeFile(join(catalogDir, 'starters', 'service', 'files', 'go', 'main.go'), 'package main\n')
    writeFile(join(catalogDir, 'features', 'tools', 'feature.yml'), 'targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n')
    const home = scratchDir()
    writeFile(join(home, 'feature-compose', 'config.yml'), `catalog: ${catalogDir}\n`)

    const result = await runCli(scratchDir(), ['check'], { XDG_CONFIG_HOME: home })

    expect(result.status).toBe(1)
    expect(result.stdout).toBe(
      'the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare\n',
    )
    expect(result.stderr).toBe('the catalog has 1 mistake\n')
  })

  it('names the feature, the files entry and the when of a when that does not parse, and exits with 1', async () => {
    const catalogDir = scratchDir()
    writeFile(join(catalogDir, 'starters', 'service', 'starter.yml'), 'folders:\n  go:\n    path: .\n')
    writeFile(join(catalogDir, 'starters', 'service', 'files', 'go', 'main.go'), 'package main\n')
    writeFile(join(catalogDir, 'features', 'auth', 'feature.yml'), "targets: [service]\nfiles:\n  - path: files/\n    when: 'method =='\n")
    writeFile(join(catalogDir, 'features', 'auth', 'files', 'go', 'auth.go'), 'package auth\n')

    const result = await runCli(scratchDir(), ['check', '--catalog', catalogDir])

    expect(result.status).toBe(1)
    expect(result.stdout).toBe(
      'the feature auth: the when "method ==" of the files entry "files/": method == needs a quoted value after it\n',
    )
    expect(result.stderr).toBe('the catalog has 1 mistake\n')
  })
})
