import { type ChildProcess, spawn } from 'node:child_process'
import { createHash } from 'node:crypto'
import { existsSync, mkdirSync, readdirSync, readFileSync, realpathSync, writeFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import { type AddressInfo, createServer as createNetServer, type Socket } from 'node:net'
import { join } from 'node:path'
import { describe, expect, it, onTestFinished } from 'vitest'
import { configHome, getExecutablePath, runCli, runProgram, scratchDir } from '../testUtils'

describe('feature-compose new', () => {
  it('an unknown starter stops new, and the message lists the starters of the catalog folder that the config file names', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'no-such-starter', 'x'])

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('go-event-sourcing-service')
    expect(readdirSync(dir)).toEqual([])
  })

  it('with no config file, new stops, names the path of the config file, and makes no folder', async () => {
    const dir = scratchDir()
    const emptyConfigHome = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'x'], { XDG_CONFIG_HOME: emptyConfigHome })

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain(join(emptyConfigHome, 'feature-compose', 'config.yml'))
    expect(readdirSync(dir)).toEqual([])
  })

  it('a value that does not match the pattern of its option stops new, names the option and the pattern, and makes no folder', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'x', '--option', 'go-version=1.20'])

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('go-version')
    expect(result.stdout + result.stderr).toContain('^1\\.(2[7-9]|[3-9][0-9])(\\.[0-9]+)?$')
    expect(readdirSync(dir)).toEqual([])
  })

  it(
    'makes a repo from go-event-sourcing-service that is formatted and passes its unit tests',
    async () => {
      const dir = scratchDir()
      // the new repo has its own mise.toml, and mise refuses a config that nobody trusted. mise compares
      // real paths, and on macOS /var is a link to /private/var.
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-service')
      expect(readFileSync(join(repo, 'go.mod'), 'utf8')).toContain('module github.com/hpcsc/my-service\n')
      expect(existsSync(join(repo, 'go.sum'))).toBe(true)
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const list = await runProgram(repo, 'task', ['--list'], env)
      expect(list.status, list.stdout + list.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      const api = await startedApi(repo, env)
      expect((await openAccount(api)).status).toBe(400)
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo with api-auth that passes its unit tests, and whose API answers 401 to a request without a known key',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }
      const key = 'the-key-of-the-e2e-test'

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'api-auth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-service')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(
        expect.arrayContaining(['internal/auth', 'internal/api/use_cases', 'internal/use_cases/account', 'internal/domain/projection']),
      )
      const api = await startedApi(repo, { ...env, API_KEY_HASHES: createHash('sha256').update(key).digest('hex') })
      expect((await fetch(`${api}/`)).status).toBe(200)
      for (const authorization of [undefined, 'Bearer an-unknown-key']) {
        const refused = await openAccount(api, authorization)
        const label = authorization ?? 'no Authorization header'
        expect(refused.status, label).toBe(401)
        expect(refused.headers.get('www-authenticate'), label).toBe('Bearer')
      }
      expect((await openAccount(api, `Bearer ${key}`)).status).toBe(400)
    },
    15 * 60 * 1000,
  )

  it('a value that the option method of api-auth does not have stops new, names method and its values, and makes no folder', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'x', '--feature', 'api-auth:method=oauth'])

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('the option method is not one of its values: api-key')
    expect(readdirSync(dir)).toEqual([])
  })

  it(
    'makes a repo with cli-auth, which also selects cli and api-auth, passes its unit tests, shows where the API key comes from, and calls its API with the key',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      expect(made.stdout).toBe('api-auth: needed by cli-auth\ncli: needed by cli-auth\nMade my-service from the starter go-event-sourcing-service.\n')
      const repo = join(dir, 'my-service')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', '-o', 'bin/my-service', './cmd/my-service'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(
        expect.arrayContaining(['internal/credential', 'internal/apiclient', 'internal/auth', 'internal/cli', 'internal/use_cases/account', 'internal/domain/projection']),
      )
      const cli = join(repo, 'bin', 'my-service')
      const help = await runProgram(repo, cli, [], env)
      expect(commandsIn(help.stdout)).toEqual(['version', 'update', 'auth', 'help, h'])
      const key = 'the-key-of-the-e2e-test'
      const configHomes = [scratchDir(), scratchDir()]
      const fromVariable = await runProgram(repo, cli, ['auth', 'status'], { ...env, XDG_CONFIG_HOME: configHomes[0], MY_SERVICE_API_KEY: key })
      expect(fromVariable).toMatchObject({ status: 0, stdout: 'The API key comes from MY_SERVICE_API_KEY.\n' })
      expect(fromVariable.stdout + fromVariable.stderr).not.toContain(key)
      const noSource = await runProgram(repo, cli, ['auth', 'status'], { ...env, XDG_CONFIG_HOME: configHomes[1], MY_SERVICE_API_KEY: '' })
      expect(noSource.status).not.toBe(0)
      expect(noSource.stdout + noSource.stderr).toContain('MY_SERVICE_API_KEY')
      expect(noSource.stdout + noSource.stderr).toContain('credentialCommand')
      expect(configHomes.map((home) => readdirSync(home))).toEqual([[], []])
      const statusHelp = await runProgram(repo, cli, ['auth', 'status', '--help'], env)
      expect(statusHelp.stdout).toContain('--help')
      expect(statusHelp.stdout).not.toMatch(/--\S*key/i)

      const api = await startedApi(repo, { ...env, API_KEY_HASHES: createHash('sha256').update(key).digest('hex') })
      const call = await builtCall(repo, env)
      const withKey = await runProgram(repo, call, [api], { ...env, XDG_CONFIG_HOME: scratchDir(), MY_SERVICE_API_KEY: key })
      expect(withKey, 'POST /accounts with {} gets the answer 400 of the handler').toMatchObject({ status: 0, stdout: '400\n' })
      const wrongKey = await runProgram(repo, call, [api], { ...env, XDG_CONFIG_HOME: scratchDir(), MY_SERVICE_API_KEY: 'an-unknown-key' })
      expect(wrongKey.status).not.toBe(0)
      expect(wrongKey.stdout).toContain('the API refused the API key from MY_SERVICE_API_KEY: set MY_SERVICE_API_KEY, or set credentialCommand in')
      const noKey = await runProgram(repo, call, [api], { ...env, XDG_CONFIG_HOME: scratchDir(), MY_SERVICE_API_KEY: '' })
      expect(noKey.status).not.toBe(0)
      expect(noKey.stdout).toContain('no API key: set MY_SERVICE_API_KEY, or set credentialCommand in')
    },
    15 * 60 * 1000,
  )

  it('--dry-run prints the repo, its starter, and each file with its owner, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('Repo: my-service\nStarter: go-event-sourcing-service\n')
    expect(result.stdout).toContain('\n  go.mod: the starter\n')
    expect(result.stdout).toContain('\n  cmd/my-service/main.go: the feature cli\n')
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run lists each feature of cli-auth with its source, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('Features:\n  api-auth: needed by cli-auth\n  cli: needed by cli-auth\n  cli-auth: command line\n\n')
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run shows each option with its value and its source, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--option', 'go-version=1.28', '--feature', 'cli-auth', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('Options:\n  starter.go-version: 1.28 (command line)\n  starter.module: github.com/hpcsc/my-service (default)\n  api-auth.method: api-key (default)\n\n')
    expect(readdirSync(dir)).toEqual([])
  })

  for (const [name, args] of [
    ['an unknown feature', ['--feature', 'no-such-feature']],
    ['a go-version that does not match its pattern', ['--option', 'go-version=1.20']],
  ]) {
    it(`${name} stops --dry-run with the message that stops new, and makes no folder`, async () => {
      const dir = scratchDir()

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', ...args])
      const previewed = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', ...args, '--dry-run'])

      expect(made.status).not.toBe(0)
      expect(previewed.status).not.toBe(0)
      expect(previewed.stderr).toBe(made.stderr)
      expect(previewed.stdout).toBe('')
      expect(readdirSync(dir)).toEqual([])
    })
  }

  it('an unknown feature stops new, and the message lists the features that target the starter', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'x', '--feature', 'no-such-feature'])

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('the features that target the starter go-event-sourcing-service are api-auth, cli, cli-auth\n')
    expect(readdirSync(dir)).toEqual([])
  })

  it(
    'makes a repo with cli that builds, passes the unit tests of the service and of the CLI, and runs the CLI',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-service')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(
        expect.arrayContaining(['internal/cli', 'internal/cli/release', 'internal/cli/version', 'internal/cli/progress', 'internal/use_cases/account', 'internal/domain/projection']),
      )
      const cli = await runProgram(repo, 'go', ['run', './cmd/my-service'], env)
      expect(cli.status, cli.stdout + cli.stderr).toBe(0)
      expect(commandsIn(cli.stdout)).toEqual(['version', 'update', 'help, h'])
    },
    15 * 60 * 1000,
  )

  it('a SIGTERM during the folder commands stops the commands too, and new ends with an error', async () => {
    const dir = scratchDir()
    const proxy = await silentGoProxy()
    const tool = spawn(getExecutablePath(), ['new', 'go-event-sourcing-service', 'my-service'], {
      cwd: dir,
      env: {
        ...process.env,
        GOPROXY: proxy.url,
        GOSUMDB: 'off',
        GOMODCACHE: join(dir, 'modcache'),
        GOFLAGS: '-modcacherw',
        MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir),
        XDG_CONFIG_HOME: configHome(),
      },
    })
    let printed = ''
    tool.stdout?.on('data', (chunk) => (printed += chunk))
    tool.stderr?.on('data', (chunk) => (printed += chunk))

    await until(() => proxy.openConnections() > 0)
    tool.kill('SIGTERM')

    expect(await exitStatus(tool), printed).toBe(1)
    await until(() => proxy.openConnections() === 0)
  })
})

// silentGoProxy accepts the requests of go mod tidy and never answers, so go mod tidy waits until something stops it.
async function silentGoProxy(): Promise<{ url: string; openConnections: () => number }> {
  const sockets = new Set<Socket>()
  const server: Server = createServer(() => {})
  server.on('connection', (socket) => {
    sockets.add(socket)
    socket.on('close', () => sockets.delete(socket))
  })
  await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
  onTestFinished(() => {
    for (const socket of sockets) socket.destroy()
    server.close()
  })
  return { url: `http://127.0.0.1:${(server.address() as AddressInfo).port}`, openConnections: () => sockets.size }
}

function exitStatus(child: ChildProcess): Promise<number | null> {
  return new Promise((done) => child.on('close', (status) => done(status)))
}

async function until(condition: () => boolean): Promise<void> {
  for (let waited = 0; !condition(); waited += 50) {
    if (waited > 20000) throw new Error('the condition did not hold within 20 seconds')
    await new Promise((done) => setTimeout(done, 50))
  }
}

// startedApi builds the API of a written repo and starts it on a free port. The database settings point to no
// database: the pool of the API opens no connection before the first query, and the requests of the tests
// stop before a query.
async function startedApi(repo: string, env: Record<string, string>): Promise<string> {
  const build = await runProgram(repo, 'go', ['build', '-o', join(repo, 'api-of-the-test'), './cmd/api'], env)
  expect(build.status, build.stdout + build.stderr).toBe(0)
  const port = await freePort()
  const api = spawn(join(repo, 'api-of-the-test'), [], {
    cwd: repo,
    env: {
      ...process.env,
      ...env,
      PORT: String(port),
      DB_USERNAME: 'nobody',
      DB_PASSWORD: 'nothing',
      DB_HOST: '127.0.0.1',
      DB_HOST_PORT: '1',
      DB_NAME: 'none',
    },
  })
  let printed = ''
  api.stdout?.on('data', (chunk) => (printed += chunk))
  api.stderr?.on('data', (chunk) => (printed += chunk))
  onTestFinished(() => {
    api.kill('SIGKILL')
  })
  const url = `http://127.0.0.1:${port}`
  for (let waited = 0; ; waited += 100) {
    if (api.exitCode !== null) throw new Error(`the API stopped with the exit status ${api.exitCode}: ${printed}`)
    if (await answers(url)) return url
    if (waited > 20000) throw new Error(`the API did not answer within 20 seconds: ${printed}`)
    await new Promise((done) => setTimeout(done, 100))
  }
}

async function answers(url: string): Promise<boolean> {
  try {
    return (await fetch(`${url}/`)).ok
  } catch {
    return false
  }
}

function freePort(): Promise<number> {
  return new Promise((done, fail) => {
    const server = createNetServer()
    server.on('error', fail)
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address() as AddressInfo
      server.close(() => done(port))
    })
  })
}

// POST /accounts with the body {} gets the answer 400 of the handler, because the owner is missing, before any
// query to the database.
async function builtCall(repo: string, env: Record<string, string>): Promise<string> {
  mkdirSync(join(repo, 'cmd', 'call-of-the-test'))
  writeFileSync(
    join(repo, 'cmd', 'call-of-the-test', 'main.go'),
    `package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hpcsc/my-service/internal/apiclient"
)

func main() {
	response, err := apiclient.New(os.Args[1]).Do(context.Background(), http.MethodPost, "/accounts", strings.NewReader("{}"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer response.Body.Close()
	fmt.Println(response.StatusCode)
}
`,
  )
  const build = await runProgram(repo, 'go', ['build', '-o', join(repo, 'call-of-the-test'), './cmd/call-of-the-test'], env)
  expect(build.status, build.stdout + build.stderr).toBe(0)
  return join(repo, 'call-of-the-test')
}

function openAccount(api: string, authorization?: string): Promise<Response> {
  return fetch(`${api}/accounts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...(authorization ? { Authorization: authorization } : {}) },
    body: '{}',
  })
}

// gotestsum starts the line of a package with ✓ when its tests pass, and with ∅ when the package has no test
// that the build tags let run.
function passedPackages(output: string): string[] {
  return output
    .split('\n')
    .filter((line) => line.startsWith('✓'))
    .map((line) => line.split(/\s+/)[1])
}

function commandsIn(help: string): string[] {
  const section = help.split('COMMANDS:\n')[1]?.split('\n\n')[0] ?? ''
  return section
    .split('\n')
    .map((line) => line.trim().split(/\s{2,}/)[0])
    .filter((name) => name !== '')
}
