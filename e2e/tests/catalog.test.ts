import { execFileSync } from 'node:child_process'
import { readFileSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:http'
import type { AddressInfo } from 'node:net'
import { join } from 'node:path'
import { describe, expect, it, onTestFinished } from 'vitest'
import { noGitConfigOfTheMachine, openCli, runCli, scratchDir, writeFile } from '../testUtils'

function git(dir: string, args: string[]): string {
  return execFileSync('git', ['-C', dir, ...args], { encoding: 'utf8', env: { ...process.env, ...noGitConfigOfTheMachine, XDG_CONFIG_HOME: '/dev/null' } }).trim()
}

function commitStarter(source: string, starterName: string): string {
  writeFile(join(source, 'catalog', 'starters', starterName, 'starter.yml'), `summary: The starter ${starterName}.\nfolders:\n  app:\n    path: .\n`)
  writeFile(join(source, 'catalog', 'starters', starterName, 'files', 'app', 'README.md'), '# {{ .Name }}\n')
  git(source, ['add', '--all'])
  git(source, ['-c', 'user.name=test', '-c', 'user.email=test@example.com', 'commit', '--quiet', '--message', `Add ${starterName}`])
  return git(source, ['rev-parse', 'HEAD'])
}

async function servedIndex(contents: string): Promise<string> {
  const server = createServer((_, response) => response.end(contents))
  await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
  onTestFinished(() => new Promise<void>((done) => server.close(() => done())))
  return `http://127.0.0.1:${(server.address() as AddressInfo).port}/catalogs.yml`
}

describe('feature-compose catalog', () => {
  it('lists the known catalogs of the index, and adds one of them by its name with no source', async () => {
    const source = scratchDir()
    git(source, ['init', '--quiet', '--initial-branch=main'])
    const firstCommit = commitStarter(source, 'first')
    const url = `file://${source}`
    const indexURL = await servedIndex(`catalogs:\n  team:\n    summary: The catalog of the team.\n    source: ${url}\n    path: catalog\n`)
    const dir = scratchDir()
    const env = { XDG_CONFIG_HOME: scratchDir(), XDG_DATA_HOME: scratchDir(), FEATURE_COMPOSE_INDEX_URL: indexURL, ...noGitConfigOfTheMachine }

    const known = await runCli(dir, ['catalog', 'list', '--known'], env)
    const added = await runCli(dir, ['catalog', 'add', 'team'], env)
    const made = await runCli(dir, ['new', 'team/first', 'my-repo'], env)

    expect([known.status, added.status, made.status], known.stderr + added.stderr + made.stderr).toEqual([0, 0, 0])
    expect(known.stdout).toContain(`team  ${url}  catalog  The catalog of the team.\n`)
    expect(added.stdout).toContain(`Added the catalog team from ${url} at the commit ${firstCommit.slice(0, 7)}.\n`)
    expect(made.stdout).toBe('Made my-repo from the starter team/first.\n')
  })

  it('adds a catalog from a git repo, syncs it to a new commit of the repo, and makes a repo from a starter of that commit', async () => {
    const source = scratchDir()
    git(source, ['init', '--quiet', '--initial-branch=main'])
    const firstCommit = commitStarter(source, 'first')
    const url = `file://${source}`
    const dir = scratchDir()
    const env = { XDG_CONFIG_HOME: scratchDir(), XDG_DATA_HOME: scratchDir(), ...noGitConfigOfTheMachine }

    const added = await runCli(dir, ['catalog', 'add', 'team', url, '--path', 'catalog'], env)
    const secondCommit = commitStarter(source, 'second')
    const synced = await runCli(dir, ['catalog', 'sync'], env)
    const made = await runCli(dir, ['new', 'second', 'my-repo'], env)

    expect([added.status, synced.status, made.status], added.stderr + synced.stderr + made.stderr).toEqual([0, 0, 0])
    expect(added.stdout).toContain(`Added the catalog team from ${url} at the commit ${firstCommit.slice(0, 7)}.\n`)
    expect(synced.stdout).toBe(`team: from ${firstCommit.slice(0, 7)} to ${secondCommit.slice(0, 7)}\n`)
    expect(made.stdout).toBe('Made my-repo from the starter team/second.\n')
    expect(readFileSync(join(dir, 'my-repo', '.feature-compose.yml'), 'utf8')).toContain(
      `catalog:\n  name: team\n  source: ${url}\n  path: catalog\n  commit: ${secondCommit}\nstarter: second\n`,
    )
  })

  it('stops with the message about git clone when git cannot read the source, and adds no catalog', async () => {
    const missing = `file://${join(scratchDir(), 'missing')}`
    const dir = scratchDir()
    const env = { XDG_CONFIG_HOME: scratchDir(), XDG_DATA_HOME: scratchDir(), ...noGitConfigOfTheMachine }

    const added = await runCli(dir, ['catalog', 'add', 'team', missing], env)
    const listed = await runCli(dir, ['catalog', 'list'], env)

    expect(added.status).not.toBe(0)
    expect(added.stderr).toContain(`git cannot read ${missing}: check that git clone ${missing} works\n`)
    expect(listed.stdout).toBe('The config file has no catalog. Add one with: feature-compose catalog add <name> <source>\n')
  })
})

describe('feature-compose catalog in a terminal', () => {
  it('stops at once with the message about git clone when the source asks for a credential, and does not wait for a password', async () => {
    const server = createServer((_, response) => {
      response.writeHead(401, { 'WWW-Authenticate': 'Basic realm="catalog"' })
      response.end()
    })
    await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
    onTestFinished(() => new Promise<void>((done) => server.close(() => done())))
    const url = `http://127.0.0.1:${(server.address() as AddressInfo).port}/team/catalog.git`
    const env = { XDG_CONFIG_HOME: scratchDir(), XDG_DATA_HOME: scratchDir(), GIT_ASKPASS: '', SSH_ASKPASS: '', ...noGitConfigOfTheMachine }

    const session = await openCli(scratchDir(), ['catalog', 'add', 'team', url], env)

    await session.waitForText(/EXIT:[1-9]/, { timeout: 15_000 })
    const screen = await session.text()
    expect(screen).toContain(`git cannot read ${url}: check that git clone ${url} works`)
  })

  it('gives ssh no terminal, so ssh cannot ask for a passphrase or a host key', async () => {
    const dir = scratchDir()
    const marker = join(dir, 'terminal')
    const ssh = join(dir, 'ssh')
    writeFileSync(ssh, `#!/bin/sh\nif (: < /dev/tty) 2>/dev/null; then echo yes > '${marker}'; else echo no > '${marker}'; fi\nexit 1\n`, { mode: 0o755 })
    const env = { XDG_CONFIG_HOME: scratchDir(), XDG_DATA_HOME: scratchDir(), GIT_ASKPASS: '', SSH_ASKPASS: '', GIT_SSH_COMMAND: ssh, ...noGitConfigOfTheMachine }

    const session = await openCli(scratchDir(), ['catalog', 'add', 'team', 'ssh://git@example.invalid/team/catalog.git'], env)

    await session.waitForText(/EXIT:[1-9]/, { timeout: 15_000 })
    expect(readFileSync(marker, 'utf8')).toBe('no\n')
  })

  it('with no command, opens a TUI that lists the catalogs and the known catalogs, syncs with S, and stops with q', async () => {
    const source = scratchDir()
    git(source, ['init', '--quiet', '--initial-branch=main'])
    commitStarter(source, 'first')
    const indexURL = await servedIndex('catalogs:\n  other:\n    summary: Another catalog.\n    source: https://example.com/other.git\n')
    const env = { XDG_CONFIG_HOME: scratchDir(), XDG_DATA_HOME: scratchDir(), FEATURE_COMPOSE_INDEX_URL: indexURL, ...noGitConfigOfTheMachine }
    const added = await runCli(scratchDir(), ['catalog', 'add', 'team', `file://${source}`, '--path', 'catalog'], env)
    expect(added.status, added.stderr).toBe(0)

    const session = await openCli(scratchDir(), ['catalog'], env)

    await session.waitForText('Known catalogs', { timeout: 15_000 })
    await session.waitForText('other')
    expect(await session.text()).toContain('source  file://')
    await session.press('S')
    await session.waitForText('team: no new commit, at')
    await session.press('q')
    await session.waitForText('EXIT:0')
  })
})
