import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir } from '../testUtils'

function writeFile(path: string, contents: string): void {
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, contents)
}

describe('feature-compose check', () => {
  it('with --build, makes the repo of each combination, runs its commands, names each combination whose command fails, and exits with 1', async () => {
    const catalogDir = scratchDir()
    writeFile(join(catalogDir, 'starters', 'service', 'starter.yml'), 'folders:\n  app:\n    path: .\n    test: [grep -q my-repo README.md, test ! -f broken.txt]\n')
    writeFile(join(catalogDir, 'starters', 'service', 'files', 'app', 'README.md'), '# {{ .Name }}\n')
    writeFile(join(catalogDir, 'features', 'broken', 'feature.yml'), 'targets: [service]\nfiles:\n  - path: files/\n')
    writeFile(join(catalogDir, 'features', 'broken', 'files', 'app', 'broken.txt'), 'broken\n')

    const result = await runCli(scratchDir(), ['check', '--catalog', catalogDir, '--build'])

    expect(result.status).toBe(1)
    expect(result.stdout.replace(/\(\d+s\)/g, '(…)')).toBe(
      'The catalog has no mistakes.\n' +
        'The catalog has 2 combinations. Make the repo of each, and run its after_render and test commands.\n' +
        '[1/2] ok    service (…)\n' +
        '[2/2] FAIL  service --feature broken (…)\n' +
        '      the test command "test ! -f broken.txt" of the folder app failed: exit status 1\n',
    )
    expect(result.stderr).toBe('1 of 2 combinations failed:\n  service --feature broken\n')
  })

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

  it('names the feature, the option and the when of a when of an option that does not parse, and exits with 1', async () => {
    const catalogDir = scratchDir()
    writeFile(join(catalogDir, 'starters', 'service', 'starter.yml'), 'folders:\n  go:\n    path: .\n')
    writeFile(join(catalogDir, 'starters', 'service', 'files', 'go', 'main.go'), 'package main\n')
    writeFile(join(catalogDir, 'features', 'auth', 'feature.yml'), "targets: [service]\noptions:\n  store-key:\n    default: 'false'\n    when: 'method =='\n")

    const result = await runCli(scratchDir(), ['check', '--catalog', catalogDir])

    expect(result.status).toBe(1)
    expect(result.stdout).toBe('the feature auth: the when "method ==" of the option "store-key": method == needs a quoted value after it\n')
    expect(result.stderr).toBe('the catalog has 1 mistake\n')
  })

  it('names the template, the option and the feature of a template that reads a starter option that its target does not declare, and exits with 1', async () => {
    const catalogDir = scratchDir()
    writeFile(join(catalogDir, 'starters', 'service', 'starter.yml'), 'folders:\n  go:\n    path: .\n')
    writeFile(join(catalogDir, 'starters', 'service', 'files', 'go', 'main.go'), 'package main\n')
    writeFile(join(catalogDir, 'features', 'auth', 'feature.yml'), 'targets: [service]\nfiles:\n  - path: files/\n')
    writeFile(join(catalogDir, 'features', 'auth', 'files', 'go', 'auth.go'), 'module {{ option "starter.module" }}\n')

    const result = await runCli(scratchDir(), ['check', '--catalog', catalogDir])

    expect(result.status).toBe(1)
    expect(result.stdout).toBe(
      'the feature auth: the template go/auth.go of the files entry files/ reads "starter.module", which the starter service does not declare\n',
    )
    expect(result.stderr).toBe('the catalog has 1 mistake\n')
  })

  it('names the path and both owners of a path that the starter and a feature write, and exits with 1', async () => {
    const catalogDir = scratchDir()
    writeFile(join(catalogDir, 'starters', 'service', 'starter.yml'), 'folders:\n  go:\n    path: .\n')
    writeFile(join(catalogDir, 'starters', 'service', 'files', 'go', 'go.mod.tmpl'), 'module example.com/service\n')
    writeFile(join(catalogDir, 'features', 'auth', 'feature.yml'), 'targets: [service]\nfiles:\n  - path: files/\n')
    writeFile(join(catalogDir, 'features', 'auth', 'files', 'go', 'go.mod'), 'module example.com/auth\n')

    const result = await runCli(scratchDir(), ['check', '--catalog', catalogDir])

    expect(result.status).toBe(1)
    expect(result.stdout).toBe('the starter service: the starter and the feature auth both write go.mod\n')
    expect(result.stderr).toBe('the catalog has 1 mistake\n')
  })
})
