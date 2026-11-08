import { type ChildProcess, spawn } from 'node:child_process'
import { existsSync, readdirSync, readFileSync, realpathSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import type { AddressInfo, Socket } from 'node:net'
import { join } from 'node:path'
import { describe, expect, it, onTestFinished } from 'vitest'
import { getExecutablePath, runCli, runProgram, scratchDir } from '../testUtils'

describe('feature-compose new', () => {
  it('an unknown starter stops new, and the message lists the starters in the binary', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'no-such-starter', 'x'], { SCAFFOLD_DIR: '' })

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('go-event-sourcing-service')
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
