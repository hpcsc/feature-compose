import { type ChildProcess, spawn } from 'node:child_process'
import { createHash, generateKeyPairSync, sign } from 'node:crypto'
import { existsSync, mkdirSync, readdirSync, readFileSync, realpathSync, writeFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import { type AddressInfo, createServer as createNetServer, type Socket } from 'node:net'
import { dirname, join } from 'node:path'
import { describe, expect, it, onTestFinished } from 'vitest'
import { buildTag, configHome, getExecutablePath, openCli, runCli, runProgram, scratchDir } from '../testUtils'

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
    'makes a repo from go-event-sourcing-service that is formatted, passes its unit tests, and has no task of cli',
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
      const list = await runProgram(repo, 'task', ['--list-all'], env)
      expect(list.status, list.stdout + list.stderr).toBe(0)
      expect(tasksIn(list.stdout)).toContain('run')
      for (const name of ['build', 'cli', 'release:local']) {
        expect(tasksIn(list.stdout)).not.toContain(name)
      }
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      const api = await startedApi(repo, env)
      expect((await fetch(`${api}/`)).status).toBe(200)
      expect((await openAccount(api)).status).toBe(400)
      const atRootPath = await fetch(`${new URL(api).origin}/accounts`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' })
      expect(atRootPath.status).toBe(404)
      const defaultOrigins = { 'http://localhost:8080': 'http://localhost:8080', 'http://127.0.0.1:8080': 'http://127.0.0.1:8080', 'https://app.example.com': null }
      expect(await allowedOriginsOf(api, Object.keys(defaultOrigins)), 'no ALLOWED_ORIGINS').toEqual(defaultOrigins)
      const withEmptyOrigins = await startedApi(repo, { ...env, ALLOWED_ORIGINS: '' })
      expect(await allowedOriginsOf(withEmptyOrigins, Object.keys(defaultOrigins)), 'an empty ALLOWED_ORIGINS').toEqual(defaultOrigins)
      const withTwoOrigins = await startedApi(repo, { ...env, ALLOWED_ORIGINS: 'https://app.example.com, http://localhost:3000' })
      expect(await allowedOriginsOf(withTwoOrigins, ['https://app.example.com', 'http://localhost:3000', 'http://localhost:8080'])).toEqual({
        'https://app.example.com': 'https://app.example.com',
        'http://localhost:3000': 'http://localhost:3000',
        'http://localhost:8080': null,
      })
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo from go that is formatted, builds, passes the unit tests of its example package, has each task that its CI workflow and its hook call, and has no task of cli',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go', 'my-tool', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-tool')
      expect(readFileSync(join(repo, 'go.mod'), 'utf8')).toContain('module github.com/hpcsc/my-tool\n')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(['internal/greeting'])
      const list = await runProgram(repo, 'task', ['--list-all'], env)
      expect(list.status, list.stdout + list.stderr).toBe(0)
      const called = ['.github/workflows/ci.yml', 'scripts/libs/git-hook-prepush.sh'].flatMap((path) => tasksCalledBy(repo, path))
      expect(called).toContain('test:unit')
      expect(tasksIn(list.stdout)).toEqual(expect.arrayContaining(called))
      for (const name of ['build', 'cli', 'release:local', 'test:e2e', 'test:e2e:local']) {
        expect(tasksIn(list.stdout)).not.toContain(name)
      }
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo from chi-api that is formatted, builds, passes the unit tests of each package with tests, has api:run and each task that its CI workflow and its hook call and no task of cli, and whose API serves its routes under /api and checks no credential',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'chi-api', 'my-api', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-api')
      expect(readFileSync(join(repo, 'go.mod'), 'utf8')).toContain('module github.com/hpcsc/my-api\n')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      const testedPackages = foldersWithTests(repo, 'internal')
      expect(testedPackages).toEqual(expect.arrayContaining(['internal/use_cases', 'internal/use_cases/user']))
      expect(passedPackages(unit.stdout).sort()).toEqual(testedPackages.sort())
      const list = await runProgram(repo, 'task', ['--list-all'], env)
      expect(list.status, list.stdout + list.stderr).toBe(0)
      const called = ['.github/workflows/ci.yml', 'scripts/libs/git-hook-prepush.sh'].flatMap((path) => tasksCalledBy(repo, path))
      expect(called).toContain('test:unit')
      expect(tasksIn(list.stdout)).toEqual(expect.arrayContaining(['api:run', ...called]))
      for (const name of ['build', 'cli', 'release:local', 'test:e2e', 'test:e2e:local']) {
        expect(tasksIn(list.stdout)).not.toContain(name)
      }
      const api = await startedApi(repo, env)
      expect((await fetch(`${api}/`)).status).toBe(200)
      const withNoCredential = await createUser(api)
      expect(withNoCredential.status).toBe(400)
      expect(await withNoCredential.json()).toMatchObject({ successful: false })
      expect((await createUser(new URL(api).origin)).status).toBe(404)
    },
    15 * 60 * 1000,
  )

  it(
    'makes a CLI repo from go with cli whose go.mod gives the modules of the CLI the versions of cli, also with no proxy, that has each task that its CI workflow calls, builds, passes the unit tests of the starter and of the CLI, builds and runs the CLI with its tasks, passes its end-to-end tests with task test:e2e:local, whose snapshot from task release:local has the archives and the checksums that update and the installer download, and whose git status shows no file of the snapshot or of the packages of e2e',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go', 'my-tool', '--feature', 'cli', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-tool')
      const requires: Array<{ module: string; version: string }> = JSON.parse((await runCli(dir, ['show'])).stdout).features.cli.add_to['gomod.requires']
      expect(requires.map((item) => item.module)).toContain('github.com/urfave/cli/v3')
      const goMod = readFileSync(join(repo, 'go.mod'), 'utf8')
      for (const { module, version } of requires) {
        expect(goMod, module).toMatch(new RegExp(`^\\s*(?:require )?${module.replaceAll('.', '\\.')} ${version.replaceAll('.', '\\.')}$`, 'm'))
      }
      const withNoProxy = scratchDir()
      const madeWithNoProxy = await runCli(withNoProxy, ['new', 'go', 'my-tool', '--feature', 'cli', '--no-prompt'], {
        MISE_TRUSTED_CONFIG_PATHS: realpathSync(withNoProxy),
        GOPROXY: 'off',
        GOSUMDB: 'off',
      })
      expect(madeWithNoProxy.status, madeWithNoProxy.stdout + madeWithNoProxy.stderr).toBe(0)
      expect(readFileSync(join(withNoProxy, 'my-tool', 'go.mod'), 'utf8')).toBe(goMod)
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      const testedCLIPackages = foldersWithTests(repo, 'internal/cli')
      expect(testedCLIPackages).toContain('internal/cli')
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(['internal/greeting', ...testedCLIPackages]))
      expect(await commandsOfTheBuiltCLI(repo, 'my-tool', env)).toEqual(['version', 'update', 'help, h'])
      expect(await commandsOfTheCLIFromTheSource(repo, env)).toEqual(['version', 'update', 'help, h'])
      const list = await runProgram(repo, 'task', ['--list-all'], env)
      const named = [...readFileSync(join(repo, 'docs', 'cli.md'), 'utf8').matchAll(/(?:^|`)task ([a-z0-9:_-]+)/gm)].map((match) => match[1])
      expect(named).toEqual(expect.arrayContaining(['build', 'cli', 'release:local', 'test:e2e', 'test:e2e:local']))
      expect(tasksIn(list.stdout)).toEqual(expect.arrayContaining(named))
      const calledByCI = tasksCalledBy(repo, '.github/workflows/ci.yml')
      expect(calledByCI).toEqual(expect.arrayContaining(['test:unit', 'test:e2e']))
      expect(tasksIn(list.stdout)).toEqual(expect.arrayContaining(calledByCI))
      const e2eTests = readdirSync(join(repo, 'e2e', 'tests'))
      expect(e2eTests).toContain('cli.test.ts')
      const e2e = await runProgram(repo, 'task', ['test:e2e:local'], { ...env, NO_COLOR: '1' })
      expect(e2e.status, e2e.stdout + e2e.stderr).toBe(0)
      for (const file of e2eTests) {
        expect(e2e.stdout, file).toMatch(new RegExp(`✓ tests/${file.replaceAll('.', '\\.')} `))
      }

      const snapshot = await runProgram(repo, 'task', ['release:local'], env)

      expect(snapshot.status, snapshot.stdout + snapshot.stderr).toBe(0)
      const config = readFileSync(join(repo, '.goreleaser.yaml'), 'utf8')
      const archives = listIn(config, 'goos').flatMap((os) => listIn(config, 'goarch').map((arch) => `my-tool-${os}-${arch}.tar.gz`))
      const hostArchive = `my-tool-${process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}.tar.gz`
      expect(archives).toContain(hostArchive)
      const checksums = checksumsIn(readFileSync(join(repo, 'dist', 'checksums.txt'), 'utf8'))
      expect(checksums).toMatchObject(Object.fromEntries(archives.map((archive) => [archive, sha256Of(join(repo, 'dist', archive))])))
      const { tag } = JSON.parse(readFileSync(join(repo, 'dist', 'metadata.json'), 'utf8'))
      const extracted = join(dir, 'extracted')
      mkdirSync(extracted)
      const untar = await runProgram(repo, 'tar', ['-xzf', join('dist', hostArchive), '-C', extracted])
      expect(untar.status, untar.stderr).toBe(0)
      const version = await runProgram(extracted, join(extracted, 'my-tool'), ['version'])
      expect(version).toMatchObject({ status: 0, stdout: `${tag}\n` })

      expect(existsSync(join(repo, 'e2e', 'node_modules'))).toBe(true)
      const onlyTheConfigOfTheRepo = { GIT_CONFIG_GLOBAL: '/dev/null', GIT_CONFIG_NOSYSTEM: '1', XDG_CONFIG_HOME: scratchDir() }
      const init = await runProgram(repo, 'git', ['init', '--quiet'], onlyTheConfigOfTheRepo)
      expect(init.status, init.stderr).toBe(0)
      const status = await runProgram(repo, 'git', ['status', '--porcelain', '--untracked-files=all'], onlyTheConfigOfTheRepo)
      expect(status.status, status.stderr).toBe(0)
      const untracked = status.stdout.split('\n').map((line) => line.slice(3))
      expect(untracked).toContain('go.mod')
      expect(untracked.filter((path) => path.startsWith('dist/') || path.startsWith('e2e/node_modules/'))).toEqual([])
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
      expect(await errorAnswersOf(api, `Bearer ${key}`)).toMatchObject({
        refused: { status: 401, contentType: 'application/json', keys: ['messages', 'successful'], successful: false, messages: [expect.stringContaining('sign in')] },
        rejectedByTheHandler: { status: 400, keys: ['messages', 'successful'], successful: false },
      })
      expect(await corsAnswersFrom(api, 'http://localhost:8080'), 'an allowed origin').toEqual({
        preflight: { status: 204, allowOrigin: 'http://localhost:8080', allowHeaders: 'authorization, content-type' },
        refused: { status: 401, allowOrigin: 'http://localhost:8080' },
      })
      expect(await corsAnswersFrom(api, 'https://another.example.com'), 'another origin').toEqual({
        preflight: { status: 204, allowOrigin: null, allowHeaders: null },
        refused: { status: 401, allowOrigin: null },
      })
      const withoutApiAuth = scratchDir()
      const plain = await runCli(withoutApiAuth, ['new', 'go-event-sourcing-service', 'my-service', '--no-prompt'], {
        MISE_TRUSTED_CONFIG_PATHS: realpathSync(withoutApiAuth),
      })
      expect(plain.status, plain.stdout + plain.stderr).toBe(0)
      expect(readFileSync(join(repo, 'go.mod'), 'utf8')).toBe(readFileSync(join(withoutApiAuth, 'my-service', 'go.mod'), 'utf8'))
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo with api-auth:method=oauth that passes its unit tests, and whose API accepts only an access token of its issuer for its audience, also after a restart with another audience',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'api-auth:method=oauth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-service')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(['internal/auth', 'internal/api/use_cases']))
      const issuer = await startedTestIssuer()
      const api = await startedApi(repo, { ...env, OAUTH_ISSUER_URL: issuer.url, OAUTH_AUDIENCE: 'my-api' })
      expect((await fetch(`${api}/`)).status).toBe(200)
      const refusals: Array<[string, string | undefined]> = [
        ['no Authorization header', undefined],
        ['a token for another audience', `Bearer ${issuer.token({ aud: 'another-api' })}`],
        ['a token whose exp is before now', `Bearer ${issuer.token({ aud: 'my-api', exp: Math.floor(Date.now() / 1000) - 60 })}`],
      ]
      for (const [label, authorization] of refusals) {
        const refused = await openAccount(api, authorization)
        expect(refused.status, label).toBe(401)
        expect(refused.headers.get('www-authenticate'), label).toBe('Bearer')
      }
      const token = issuer.token({ aud: 'my-api' })
      expect((await openAccount(api, `Bearer ${token}`)).status).toBe(400)
      expect(await errorAnswersOf(api, `Bearer ${token}`)).toMatchObject({
        refused: { status: 401, contentType: 'application/json', keys: ['messages', 'successful'], successful: false, messages: [expect.stringContaining('sign in')] },
        rejectedByTheHandler: { status: 400, keys: ['messages', 'successful'], successful: false },
      })
      const restarted = await startedApi(repo, { ...env, OAUTH_ISSUER_URL: issuer.url, OAUTH_AUDIENCE: 'another-api' })
      expect((await openAccount(restarted, `Bearer ${token}`)).status).toBe(401)
      expect((await openAccount(restarted, `Bearer ${issuer.token({ aud: 'another-api' })}`)).status).toBe(400)
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo from chi-api with api-auth that passes its unit tests, whose go.mod is the go.mod of chi-api alone, and whose API answers 401 to a request to a protected route without a known key',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }
      const key = 'the-key-of-the-e2e-test'

      const made = await runCli(dir, ['new', 'chi-api', 'my-api', '--feature', 'api-auth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-api')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(['internal/auth', 'internal/use_cases', 'internal/use_cases/user']))
      const api = await startedApi(repo, { ...env, API_KEY_HASHES: createHash('sha256').update(key).digest('hex') })
      expect((await fetch(`${api}/`)).status).toBe(200)
      for (const authorization of [undefined, 'Bearer an-unknown-key']) {
        const refused = await createUser(api, authorization)
        const label = authorization ?? 'no Authorization header'
        expect(refused.status, label).toBe(401)
        expect(refused.headers.get('www-authenticate'), label).toBe('Bearer')
      }
      expect(await answersOf(await createUser(api), await createUser(api, `Bearer ${key}`))).toMatchObject({
        refused: { status: 401, contentType: 'application/json', keys: ['messages', 'successful'], successful: false, messages: [expect.stringContaining('sign in')] },
        rejectedByTheHandler: { status: 400, keys: ['messages', 'successful'], successful: false },
      })
      const withoutApiAuth = scratchDir()
      const plain = await runCli(withoutApiAuth, ['new', 'chi-api', 'my-api', '--no-prompt'], { MISE_TRUSTED_CONFIG_PATHS: realpathSync(withoutApiAuth) })
      expect(plain.status, plain.stdout + plain.stderr).toBe(0)
      expect(readFileSync(join(repo, 'go.mod'), 'utf8')).toBe(readFileSync(join(withoutApiAuth, 'my-api', 'go.mod'), 'utf8'))
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo from chi-api with api-auth:method=oauth that passes its unit tests, and whose API accepts only an access token of its issuer for its audience',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'chi-api', 'my-api', '--feature', 'api-auth:method=oauth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-api')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(['internal/auth', 'internal/use_cases']))
      const issuer = await startedTestIssuer()
      const api = await startedApi(repo, { ...env, OAUTH_ISSUER_URL: issuer.url, OAUTH_AUDIENCE: 'my-api' })
      expect((await fetch(`${api}/`)).status).toBe(200)
      for (const [label, authorization] of [
        ['no Authorization header', undefined],
        ['a token for another audience', `Bearer ${issuer.token({ aud: 'another-api' })}`],
      ] as Array<[string, string | undefined]>) {
        const refused = await createUser(api, authorization)
        expect(refused.status, label).toBe(401)
        expect(refused.headers.get('www-authenticate'), label).toBe('Bearer')
      }
      const token = issuer.token({ aud: 'my-api' })
      expect(await answersOf(await createUser(api), await createUser(api, `Bearer ${token}`))).toMatchObject({
        refused: { status: 401, contentType: 'application/json', keys: ['messages', 'successful'], successful: false, messages: [expect.stringContaining('sign in')] },
        rejectedByTheHandler: { status: 400, keys: ['messages', 'successful'], successful: false },
      })
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo with process-manager that builds, passes its unit tests, has the tasks of task run, serves the payment and webhook routes, and whose fakes call the webhook of a request',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'process-manager', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-service')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      for (const args of [['build', './...'], ['vet', '-tags', 'integration', './...']]) {
        const result = await runProgram(repo, 'go', args, env)
        expect(result.status, `go ${args.join(' ')}: ${result.stdout}${result.stderr}`).toBe(0)
      }
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      const withUnitTests = ['internal/use_cases', 'internal/api/use_cases', 'cmd/catchup'].flatMap((folder) => foldersWithUnitTests(repo, folder))
      expect(withUnitTests).toEqual(expect.arrayContaining(['internal/use_cases', 'internal/api/use_cases', 'cmd/catchup', 'internal/use_cases/payment/checks', 'internal/use_cases/payment/submission', 'internal/use_cases/payment/outside']))
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(withUnitTests))
      const tasks = await runProgram(repo, 'task', ['--list-all'], env)
      expect(tasksIn(tasks.stdout)).toEqual(expect.arrayContaining(['run', 'fake_services:run', 'api:pay', 'api:get-payment']))
      const api = await startedApi(repo, env)
      for (const route of ['/payments', '/webhooks/fraud-checks']) {
        expect((await fetch(`${api}${route}`, { method: 'POST', body: '{}' })).status, route).toBe(400)
      }
      expect((await fetch(`${api.replace(/\/api$/, '')}/payments`, { method: 'POST', body: '{}' })).status).toBe(404)
      const webhooks = await startedWebhookServer()
      const fakes = await startedFakes(repo, env)
      const answer = await fetch(`${fakes}/fraud/checks`, {
        method: 'POST',
        headers: { 'Idempotency-Key': 'key-of-the-e2e-test' },
        body: JSON.stringify({ paymentId: 'payment-1', amount: 300, creditorIban: 'DE89370400440532013000', webhookUrl: `${webhooks.url}/fraud-checks` }),
      })
      expect(answer.status).toBe(202)
      await until(() => webhooks.received.length > 0)
      expect(webhooks.received).toEqual([{ path: '/fraud-checks', key: 'key-of-the-e2e-test', body: { paymentId: 'payment-1', passed: true } }])
    },
    15 * 60 * 1000,
  )

  it('a value that the option method of api-auth does not have stops new, names method and its values, and makes no folder', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'x', '--feature', 'api-auth:method=basic'])

    expect(result.status).not.toBe(0)
    expect(result.stdout + result.stderr).toContain('the option method is not one of its values: api-key, oauth')
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run names the need of a selected feature as the source of the value that the need sets', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'service', 'my-service', '--feature', 'client', '--dry-run'], { XDG_CONFIG_HOME: needValueHome() })

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('  auth.method: api-key (need of client)\n')
    expect(readdirSync(dir)).toEqual([])
  })

  for (const flag of ['--no-prompt', '--dry-run']) {
    it(`a flag with another value than the need of a feature sets stops new ${flag}, names both sources, and makes no folder`, async () => {
      const dir = scratchDir()

      const result = await runCli(dir, ['new', 'service', 'my-service', '--feature', 'client', '--feature', 'auth:method=oauth', flag], { XDG_CONFIG_HOME: needValueHome() })

      expect(result.status).not.toBe(0)
      expect(result.stdout + result.stderr).toContain('"oauth" from the command line and "api-key" from the need of client')
      expect(readdirSync(dir)).toEqual([])
    })
  }

  it(
    'makes a repo with cli-auth, which also selects cli and api-auth, whose go.mod requires no module of the method oauth, passes its unit tests, shows where the API key comes from, and calls its API with the key',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      expect(made.stdout).toBe('api-auth: needed by cli-auth\ncli: needed by cli-auth\nMade my-service from the starter go-event-sourcing-service.\n')
      const repo = join(dir, 'my-service')
      expect(readFileSync(join(repo, '.feature-compose.yml'), 'utf8')).toBe(
        `version: ${buildTag()}\nstarter: go-event-sourcing-service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\n` +
          'features:\n  api-auth:\n    method: api-key\n  cli: {}\n  cli-auth:\n    store-key: "false"\nrequested: [cli-auth]\n',
      )
      const requires: Array<{ module: string }> = JSON.parse((await runCli(dir, ['show'])).stdout).features['cli-auth'].add_to['gomod.requires']
      expect(requires.map((item) => item.module).sort()).toEqual(['github.com/cli/browser', 'github.com/zalando/go-keyring', 'golang.org/x/oauth2', 'golang.org/x/term'])
      const goMod = readFileSync(join(repo, 'go.mod'), 'utf8')
      for (const { module } of requires) {
        expect(goMod, `go mod tidy removes ${module}, which only the files of the method oauth and of the stored key import`).not.toContain(module)
      }
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
      expect(withKey, 'POST /api/accounts with {} gets the answer 400 of the handler').toMatchObject({ status: 0, stdout: '400\n' })
      const wrongKey = await runProgram(repo, call, [api], { ...env, XDG_CONFIG_HOME: scratchDir(), MY_SERVICE_API_KEY: 'an-unknown-key' })
      expect(wrongKey.status).not.toBe(0)
      expect(wrongKey.stdout).toContain('the API refused the API key from MY_SERVICE_API_KEY: set MY_SERVICE_API_KEY, or set credentialCommand in')
      const noKey = await runProgram(repo, call, [api], { ...env, XDG_CONFIG_HOME: scratchDir(), MY_SERVICE_API_KEY: '' })
      expect(noKey.status).not.toBe(0)
      expect(noKey.stdout).toContain('no API key: set MY_SERVICE_API_KEY, or set credentialCommand in')
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo with cli-auth:store-key=true that passes its unit and end-to-end tests, whose CLI stores the API key with auth login and sends it to its API, while the variable still comes first, and removes it with auth logout',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth:store-key=true', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-service')
      expect(readFileSync(join(repo, '.feature-compose.yml'), 'utf8')).toContain('  cli-auth:\n    store-key: "true"\n')
      const requires: Array<{ module: string; version: string }> = JSON.parse((await runCli(dir, ['show'])).stdout).features['cli-auth'].add_to['gomod.requires']
      const goMod = readFileSync(join(repo, 'go.mod'), 'utf8')
      for (const { module, version } of requires) {
        const required = new RegExp(`^\\s*(?:require )?${module.replaceAll('.', '\\.')} ${version.replaceAll('.', '\\.')}$`, 'm')
        if (['github.com/zalando/go-keyring', 'golang.org/x/term'].includes(module)) {
          expect(goMod, module).toMatch(required)
        } else {
          expect(goMod, module).not.toContain(module)
        }
      }
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', '-o', 'bin/my-service', './cmd/my-service'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(['internal/credential', 'internal/apiclient', 'internal/auth', 'internal/cli']))
      const e2e = await runProgram(repo, 'task', ['test:e2e:local'], { ...env, NO_COLOR: '1' })
      expect(e2e.status, e2e.stdout + e2e.stderr).toBe(0)
      expect(e2e.stdout).toMatch(/✓ tests\/auth\.test\.ts /)

      const key = 'the-key-of-the-e2e-test'
      const api = await startedApi(repo, { ...env, API_KEY_HASHES: createHash('sha256').update(key).digest('hex') })
      const cli = join(repo, 'bin', 'my-service')
      const call = await builtCall(repo, env)
      const settings = { ...env, XDG_CONFIG_HOME: scratchDir(), MY_SERVICE_API_KEY: '', MY_SERVICE_API_URL: api, MY_SERVICE_CREDENTIAL_STORE: 'file' }

      const login = await runProgram(repo, 'sh', ['-c', `printf '%s' '${key}' | '${cli}' auth login`], settings)

      expect(login.status, login.stdout + login.stderr).toBe(0)
      const withStoredKey = await runProgram(repo, call, [api], settings)
      expect(withStoredKey, 'POST /api/accounts with {} gets the answer 400 of the handler, so the API accepted the stored key').toMatchObject({ status: 0, stdout: '400\n' })
      const withVariable = await runProgram(repo, call, [api], { ...settings, MY_SERVICE_API_KEY: 'an-unknown-key' })
      expect(withVariable.status).not.toBe(0)
      expect(withVariable.stdout).toContain('the API refused the API key from MY_SERVICE_API_KEY')
      const logout = await runProgram(repo, cli, ['auth', 'logout'], settings)
      expect(logout.status, logout.stdout + logout.stderr).toBe(0)
      const afterLogout = await runProgram(repo, call, [api], settings)
      expect(afterLogout.status).not.toBe(0)
      expect(afterLogout.stdout).toContain('run "my-service auth login"')
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo with cli-auth and api-auth:method=oauth that passes its unit and end-to-end tests, whose CLI logs in at the issuer for the audience of the API, and whose API accepts the token that the CLI gets, also after a refresh',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--feature', 'api-auth:method=oauth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      expect(made.stdout).toBe('cli: needed by cli-auth\nMade my-service from the starter go-event-sourcing-service.\n')
      const repo = join(dir, 'my-service')
      expect(readFileSync(join(repo, '.feature-compose.yml'), 'utf8')).toContain('features:\n  api-auth:\n    method: oauth\n  cli: {}\n  cli-auth:\n    store-key: "false"\n')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', '-o', 'bin/my-service', './cmd/my-service'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(expect.arrayContaining(['internal/credential', 'internal/apiclient', 'internal/auth', 'internal/cli']))
      const requires: Array<{ module: string; version: string }> = JSON.parse((await runCli(dir, ['show'])).stdout).features['cli-auth'].add_to['gomod.requires']
      expect(requires.map((item) => item.module).sort()).toEqual(['github.com/cli/browser', 'github.com/zalando/go-keyring', 'golang.org/x/oauth2', 'golang.org/x/term'])
      const goMod = readFileSync(join(repo, 'go.mod'), 'utf8')
      for (const { module, version } of requires.filter((item) => item.module !== 'golang.org/x/term')) {
        expect(goMod, module).toMatch(new RegExp(`^\\s*(?:require )?${module.replaceAll('.', '\\.')} ${version.replaceAll('.', '\\.')}$`, 'm'))
      }
      expect(goMod, 'go mod tidy removes golang.org/x/term, which only the files of the stored key import').not.toContain('golang.org/x/term')
      const e2e = await runProgram(repo, 'task', ['test:e2e:local'], { ...env, NO_COLOR: '1' })
      expect(e2e.status, e2e.stdout + e2e.stderr).toBe(0)
      expect(e2e.stdout).toMatch(/✓ tests\/auth\.test\.ts /)

      const issuer = await startedTestIssuer()
      const api = await startedApi(repo, { ...env, OAUTH_ISSUER_URL: issuer.url, OAUTH_AUDIENCE: 'my-api' })
      const cli = join(repo, 'bin', 'my-service')
      const call = await builtCall(repo, env)
      const settings = (audience: string) => ({
        ...env,
        XDG_CONFIG_HOME: scratchDir(),
        MY_SERVICE_TOKEN: '',
        MY_SERVICE_OAUTH_ISSUER_URL: issuer.url,
        MY_SERVICE_OAUTH_CLIENT_ID: 'my-cli',
        MY_SERVICE_OAUTH_AUDIENCE: audience,
        MY_SERVICE_API_URL: api,
        MY_SERVICE_CREDENTIAL_STORE: 'file',
      })
      const loggedIn = settings('my-api')

      const login = await runProgram(repo, cli, ['auth', 'login', '--device'], loggedIn)

      expect(login.status, login.stdout + login.stderr).toBe(0)
      expect(issuer.audiences).toEqual(['my-api'])
      const accepted = await runProgram(repo, call, [api], loggedIn)
      expect(accepted, 'POST /api/accounts with {} gets the answer 400 of the handler, so the API accepted the token').toMatchObject({ status: 0, stdout: '400\n' })

      const expiring = settings('my-api')
      issuer.expiresIn = 1
      expect((await runProgram(repo, cli, ['auth', 'login', '--device'], expiring)).status).toBe(0)
      issuer.expiresIn = 3600
      const refreshed = await runProgram(repo, call, [api], expiring)
      expect(refreshed, 'the call refreshes the expired access token, and the API accepts the new one').toMatchObject({ status: 0, stdout: '400\n' })
      expect(issuer.refreshes).toBe(1)

      const notRefreshable = settings('my-api')
      issuer.expiresIn = 1
      expect((await runProgram(repo, cli, ['auth', 'login', '--device'], notRefreshable)).status).toBe(0)
      issuer.refusesRefresh = true
      const notRefreshed = await runProgram(repo, call, [api], notRefreshable)
      expect(notRefreshed.status).not.toBe(0)
      expect(notRefreshed.stdout).toContain('refresh the tokens for 127.0.0.1:')
      expect(notRefreshed.stdout).toContain('run "my-service auth login"')
      issuer.refusesRefresh = false
      issuer.expiresIn = 3600

      const otherAudience = settings('another-api')
      expect((await runProgram(repo, cli, ['auth', 'login', '--device'], otherAudience)).status).toBe(0)
      expect(issuer.audiences.at(-1)).toBe('another-api')
      const refusedByTheAPI = await runProgram(repo, call, [api], otherAudience)
      expect(refusedByTheAPI.status).not.toBe(0)
      expect(refusedByTheAPI.stdout).toContain('the API refused the token from')
      expect(refusedByTheAPI.stdout).toContain('run "my-service auth login"')
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
    expect(result.stdout).toContain('\n  .feature-compose.yml: feature-compose\n')
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run lists each feature of cli-auth with its source and the files of the default method api-key, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('Features:\n  api-auth: needed by cli-auth\n  cli: needed by cli-auth\n  cli-auth: command line\n\n')
    expect(result.stdout).toContain('  cli-auth.store-key: false (default)\n')
    expect(result.stdout).toContain('  internal/credential/credential.go: the feature cli-auth\n')
    expect(result.stdout).not.toContain('internal/credential/login.go')
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run of cli-auth with api-auth:method=oauth names the method from the command line and the files of oauth of cli-auth, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--feature', 'api-auth:method=oauth', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    for (const part of [
      '  api-auth: command line, needed by cli-auth\n',
      '  cli: needed by cli-auth\n',
      '  cli-auth: command line\n',
      '  api-auth.method: oauth (command line)\n',
      '  cli-auth.store-key: false (default)\n',
      '  internal/credential/login.go: the feature cli-auth\n',
      '  internal/credential/store.go: the feature cli-auth\n',
    ]) {
      expect(result.stdout).toContain(part)
    }
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run shows each option with its value and its source, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--option', 'go-version=1.28', '--feature', 'cli-auth', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain(
      'Options:\n  starter.go-version: 1.28 (command line)\n  starter.module: github.com/hpcsc/my-service (default)\n  api-auth.method: api-key (default)\n  cli-auth.store-key: false (default)\n\n',
    )
    expect(readdirSync(dir)).toEqual([])
  })

  it('--dry-run of cli-auth:store-key=true names store-key from the command line and the default method api-key, and writes nothing', async () => {
    const dir = scratchDir()

    const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth:store-key=true', '--dry-run'])

    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('  api-auth.method: api-key (default)\n  cli-auth.store-key: true (command line)\n')
    expect(readdirSync(dir)).toEqual([])
  })

  for (const flag of ['--no-prompt', '--dry-run']) {
    it(`cli-auth:store-key=true with api-auth:method=oauth stops new ${flag}, says that store-key applies only to the method api-key, and makes no folder`, async () => {
      const dir = scratchDir()

      const result = await runCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth:store-key=true', '--feature', 'api-auth:method=oauth', flag])

      expect(result.status).not.toBe(0)
      expect(result.stdout + result.stderr).toContain(
        'the option store-key of the feature cli-auth applies only when api-auth.method == "api-key", and api-auth.method is "oauth", but the command line gives it "true"',
      )
      expect(readdirSync(dir)).toEqual([])
    })
  }

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
    expect(result.stdout + result.stderr).toContain('the features that target the starter go-event-sourcing-service are api-auth, cli, cli-auth, process-manager\n')
    expect(readdirSync(dir)).toEqual([])
  })

  it('each feature of the catalog that does not target go stops new go, the message says so, and new makes no folder, and an unknown feature keeps its own message', async () => {
    const dir = scratchDir()
    const catalog = JSON.parse((await runCli(dir, ['show'])).stdout)
    const notTargetingGo = Object.entries(catalog.features)
      .filter(([, feature]) => !(feature as { targets: string[] }).targets.includes('go'))
      .map(([name]) => name)
    expect(notTargetingGo).toContain('api-auth')

    for (const feature of notTargetingGo) {
      const result = await runCli(dir, ['new', 'go', 'my-tool', '--feature', feature])

      expect(result.status, feature).not.toBe(0)
      expect(result.stdout + result.stderr, feature).toContain(`the feature ${feature} does not target the starter go: the features that target the starter go are cli\n`)
    }
    const unknown = await runCli(dir, ['new', 'go', 'my-tool', '--feature', 'no-such-feature'])
    expect(unknown.status).not.toBe(0)
    expect(unknown.stdout + unknown.stderr).toContain('unknown feature "no-such-feature": the features that target the starter go are cli\n')
    expect(readdirSync(dir)).toEqual([])
  })

  it(
    'makes a repo with cli that builds, passes the unit tests of the service and of the CLI, builds and runs the CLI with its tasks, whose task run still starts the service, and that has each task that its CI workflow calls',
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
      expect(await commandsOfTheBuiltCLI(repo, 'my-service', env)).toEqual(['version', 'update', 'help, h'])
      expect(await commandsOfTheCLIFromTheSource(repo, env)).toEqual(['version', 'update', 'help, h'])
      const run = await runProgram(repo, 'task', ['--dry', 'run'], env)
      expect(run.status, run.stdout + run.stderr).toBe(0)
      expect(run.stdout + run.stderr).toContain('tmuxp load -a .tmuxp.yaml')
      const list = await runProgram(repo, 'task', ['--list-all'], env)
      const calledByCI = tasksCalledBy(repo, '.github/workflows/ci.yml')
      expect(calledByCI).toEqual(expect.arrayContaining(['test:unit', 'test:e2e']))
      expect(tasksIn(list.stdout)).toEqual(expect.arrayContaining(calledByCI))
    },
    15 * 60 * 1000,
  )

  it(
    'makes a repo from chi-api with cli that builds, passes the unit tests of the API and of the CLI, builds and runs the CLI with its tasks, whose task api:run still starts the API, and that has each task that its CI workflow calls',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }

      const made = await runCli(dir, ['new', 'chi-api', 'my-api', '--feature', 'cli', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      const repo = join(dir, 'my-api')
      const unformatted = await runProgram(repo, 'gofmt', ['-l', '.'], env)
      expect(unformatted).toMatchObject({ status: 0, stdout: '' })
      const build = await runProgram(repo, 'go', ['build', './...'], env)
      expect(build.status, build.stdout + build.stderr).toBe(0)
      const unit = await runProgram(repo, 'task', ['test:unit'], env)
      expect(unit.status, unit.stdout + unit.stderr).toBe(0)
      expect(passedPackages(unit.stdout)).toEqual(
        expect.arrayContaining(['internal/cli', 'internal/cli/release', 'internal/cli/version', 'internal/cli/progress', 'internal/use_cases', 'internal/use_cases/user']),
      )
      expect(await commandsOfTheBuiltCLI(repo, 'my-api', env)).toEqual(['version', 'update', 'help, h'])
      expect(await commandsOfTheCLIFromTheSource(repo, env)).toEqual(['version', 'update', 'help, h'])
      const run = await runProgram(repo, 'task', ['--dry', 'api:run'], env)
      expect(run.status, run.stdout + run.stderr).toBe(0)
      expect(run.stdout + run.stderr).toContain('go run ./cmd/api')
      const list = await runProgram(repo, 'task', ['--list-all'], env)
      const calledByCI = tasksCalledBy(repo, '.github/workflows/ci.yml')
      expect(calledByCI).toEqual(expect.arrayContaining(['test:unit', 'test:e2e']))
      expect(tasksIn(list.stdout)).toEqual(expect.arrayContaining(calledByCI))
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

describe('feature-compose new in a terminal', () => {
  const flags = ['--feature', 'cli-auth', '--feature', 'api-auth:method=api-key', '--option', 'go-version=1.28', '--option', 'module=github.com/acme/my-service']
  const pressed = async (session: Awaited<ReturnType<typeof openCli>>, keys: string[]) => {
    for (const key of keys) {
      await session.press(key as Parameters<typeof session.press>[0])
    }
  }

  it('the form marks the values of the arguments and the flags as fixed and the needs as needed by, w shows the preview, ctrl+d scrolls it, esc goes back, and q cancels and writes nothing', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', ...flags])

    await session.waitForText('needed by cli-auth', { timeout: 15_000 })
    const tree = await session.text()
    for (const part of ['name     my-service  fixed', 'starter  go-event-sourcing-service  fixed', 'go-version  1.28', '[x] cli-auth  fixed', '[x] api-auth  fixed, needed by cli-auth', '[x] cli  needed by cli-auth']) {
      expect(tree).toContain(part)
    }
    expect(tree).toMatch(/go-version {2}1\.28 {2}fixed/)
    await session.press('w')
    await session.waitForText('Write the repo?')
    const preview = await session.text()
    for (const part of ['Repo: my-service', 'cli-auth: command line', 'api-auth: command line, needed by cli-auth', 'cli: needed by cli-auth', 'starter.go-version: 1.28 (command line)']) {
      expect(preview).toContain(part)
    }
    for (let i = 0; i < 20 && !(await session.text()).includes('go.mod: the starter'); i++) {
      await session.press(['ctrl', 'd'])
    }
    expect(await session.text()).toContain('go.mod: the starter')
    await session.press('escape')
    await session.waitForText('name     my-service  fixed')
    expect(await session.text()).not.toContain('Write the repo?')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(await session.text()).toContain('you canceled new, so it wrote no file')
    expect(readdirSync(dir)).toEqual([])
  })

  it('space at cli-auth selects its needs, space at a need keeps it and names cli-auth, space at cli-auth clears the three, and a feature that space selects shows answered in the preview', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service'])

    await session.waitForText('[ ] cli-auth', { timeout: 15_000 })
    await pressed(session, ['j', 'j', 'j', 'j', 'j', 'j', 'space'])
    await session.waitForText('[x] cli  needed by cli-auth')
    expect(await session.text()).toContain('[x] api-auth  needed by cli-auth')
    await pressed(session, ['k', 'k', 'k', 'space'])
    await session.waitForText('api-auth stays selected: it is needed by cli-auth.')
    expect(await session.text()).toContain('[x] api-auth  needed by cli-auth')
    await pressed(session, ['j', 'j', 'j', 'space'])
    await session.waitForText('[ ] api-auth')
    const cleared = await session.text()
    expect(cleared).toContain('[ ] cli ')
    expect(cleared).toContain('[ ] cli-auth')
    await pressed(session, ['k', 'space', 'w'])
    await session.waitForText('Write the repo?')
    expect(await session.text()).toContain('cli: answered')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it('with a feature whose need sets a value, the form marks the option as fixed with the source need of the feature, and enter at it starts no edit and names the feature', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'service', 'my-service', '--feature', 'client'], { XDG_CONFIG_HOME: needValueHome() })

    await session.waitForText('method  api-key  fixed', { timeout: 15_000 })
    await pressed(session, ['j', 'j', 'j'])
    await session.waitForText('source   need of client')
    await session.press('enter')
    await session.waitForText('The option method is fixed')
    expect(detailsPaneOf(await session.text())).toContain('The option method is fixed: the need of client sets it.')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it('with a flag of another value than the need of a feature sets, space at the feature leaves it unselected, and the details name both sources', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'service', 'my-service', '--feature', 'auth:method=oauth'], { XDG_CONFIG_HOME: needValueHome() })

    await session.waitForText('method  oauth  fixed', { timeout: 15_000 })
    await pressed(session, ['j', 'j', 'j', 'j', 'space'])
    await session.waitForText('two values')
    const refused = await session.text()
    expect(refused).toContain('[ ] client')
    expect(detailsPaneOf(refused)).toContain('"oauth" from the command line and "api-key" from the need of client')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it('with --feature api-auth:method=oauth, space at cli-auth selects it and cli', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'api-auth:method=oauth'])

    await session.waitForText('method  oauth  fixed', { timeout: 15_000 })
    await pressed(session, ['j', 'j', 'j', 'j', 'j', 'j', 'j', 'space'])
    await session.waitForText('[x] cli  needed by cli-auth')
    const selected = await session.text()
    expect(selected).toContain('[x] cli-auth')
    expect(selected).not.toContain('two values')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it('with --feature cli-auth, the tree shows store-key under cli-auth with api-key, hides it after the answer oauth, whose preview gives store-key its default, and shows it again after api-key', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth'])

    await session.waitForText('store-key  false', { timeout: 15_000 })
    await pressed(session, ['j', 'j', 'j', 'j', 'j', 'enter'])
    await session.waitForText('▌api-key')
    await pressed(session, ['j', 'enter'])
    await session.waitForText('method  oauth')
    expect(await session.text()).not.toContain('store-key')
    await session.press('w')
    await session.waitForText('Write the repo?')
    expect(await session.text()).toContain('cli-auth.store-key: false (default)')
    await session.press('escape')
    await session.waitForText('method  oauth')
    await pressed(session, ['enter', 'k', 'enter'])
    await session.waitForText('store-key  false')
    expect(await session.text()).toContain('method  api-key')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it('with --feature cli-auth, the method of api-auth is not fixed, enter lists api-key and oauth, and after oauth the preview names the answer and the files of oauth of cli-auth', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth'])

    await session.waitForText('needed by cli-auth', { timeout: 15_000 })
    expect(await session.text()).toMatch(/method {2}api-key +││/)
    await pressed(session, ['j', 'j', 'j', 'j', 'j'])
    await session.waitForText('source   default')
    await session.press('enter')
    await session.waitForText('▌api-key')
    expect(detailsPaneOf(await session.text())).toContain('▌api-key oauth')
    await session.press('j')
    await session.waitForText('▌oauth')
    await session.press('enter')
    await session.waitForText('method  oauth')
    await session.press('w')
    await session.waitForText('Write the repo?')
    expect(await session.text()).toContain('api-auth.method: oauth (answered)')
    for (let i = 0; i < 20 && !(await session.text()).includes('internal/credential/login.go: the feature cli-auth'); i++) {
      await session.press(['ctrl', 'd'])
    }
    expect(await session.text()).toContain('internal/credential/login.go: the feature cli-auth')
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it('enter at go-version edits it, the form refuses a text that does not match its pattern and keeps one that matches, and the preview shows it as answered next to the defaults', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth'])

    await session.waitForText('go-version  1.27.1', { timeout: 15_000 })
    await pressed(session, ['j', 'j', 'enter'])
    await session.waitForText('> 1.27.1')
    await pressed(session, Array(6).fill('backspace'))
    await session.type('1.20')
    await session.waitForText('^1\\.(2[7-9]|[3-9][0-9])(\\.[0-9]+)?$')
    await session.press('enter')
    expect(await session.text()).toContain('> 1.20')
    await session.press('backspace')
    await session.type('8')
    await session.press('enter')
    await session.waitForText('go-version  1.28')
    await session.press('w')
    await session.waitForText('Write the repo?')
    const preview = await session.text()
    for (const part of ['starter.go-version: 1.28 (answered)', 'starter.module: github.com/hpcsc/my-service (default)', 'api-auth.method: api-key (default)']) {
      expect(preview).toContain(part)
    }
    await session.press('q')
    await session.waitForText('EXIT:1')
    expect(readdirSync(dir)).toEqual([])
  })

  it(
    'with no argument, w selects the missing name, the form takes the name and the starter, the default of module follows the name, y writes the repo, and the printed command with --no-prompt makes the same files',
    async () => {
      const dir = scratchDir()
      const env = { MISE_TRUSTED_CONFIG_PATHS: realpathSync(dir) }
      mkdirSync(join(dir, 'tui'))
      mkdirSync(join(dir, 'flags'))
      const session = await openCli(join(dir, 'tui'), ['new'], env)

      await session.waitForText('starter    missing', { timeout: 15_000 })
      await session.press('w')
      await session.waitForText('the name "" of the repo')
      await session.press('enter')
      await session.type('my-service')
      await session.press('enter')
      await session.waitForText('name     my-service')
      await pressed(session, ['j', 'enter'])
      await session.waitForText('A Go HTTP API with chi, whose routes are under /api.')
      await session.press('j')
      await session.waitForText('A Go module with an example package, its unit tests, a pre-push hook, and a CI workflow.')
      await session.press('j')
      await session.waitForText('A Go service with an event store, an HTTP API, and a catchup process.')
      await session.press('enter')
      await session.waitForText('go-version  1.27.1')
      await pressed(session, ['j', 'j'])
      await session.waitForText('github.com/hpcsc/my-service')
      await pressed(session, ['j', 'j', 'j', 'space', 'w'])
      await session.waitForText('Write the repo?')
      await session.press('y')
      await session.waitForText('EXIT:0', { timeout: 14 * 60 * 1000 })
      expect(await session.text()).toContain('feature-compose new go-event-sourcing-service my-service --feature cli-auth')

      const made = await runCli(join(dir, 'flags'), ['new', 'go-event-sourcing-service', 'my-service', '--feature', 'cli-auth', '--no-prompt'], env)

      expect(made.status, made.stdout + made.stderr).toBe(0)
      expect(readFileSync(join(dir, 'tui', 'my-service', '.feature-compose.yml'), 'utf8')).toContain('\nrequested: [cli-auth]\n')
      expect(filesOf(join(dir, 'flags', 'my-service'))).toEqual(filesOf(join(dir, 'tui', 'my-service')))
    },
    15 * 60 * 1000,
  )

  for (const [name, args] of [
    ['stdin is not a terminal', ['<', '/dev/null']],
    ['stdout is not a terminal', ['|', 'cat']],
    ['--no-prompt asks for it', ['--no-prompt']],
  ]) {
    it(`opens no form when ${name}`, async () => {
      const dir = scratchDir()
      mkdirSync(join(dir, 'taken'))
      writeFileSync(join(dir, 'taken', 'notes.txt'), 'notes\n')
      const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--output-dir', 'taken', ...args])

      await session.waitForText(/EXIT:\d+/, { timeout: 15_000 })
      const screen = await session.text()
      expect(screen).toContain('the output folder taken is not empty')
      expect(screen).not.toContain('j/k move')
    })
  }

  it('--dry-run in a terminal prints the preview and opens no form', async () => {
    const dir = scratchDir()
    const session = await openCli(dir, ['new', 'go-event-sourcing-service', 'my-service', '--dry-run'])

    await session.waitForText('EXIT:0', { timeout: 15_000 })
    const screen = await session.text()
    expect(screen).toContain(': the starter')
    expect(screen).not.toContain('j/k move')
    expect(readdirSync(dir)).toEqual([])
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

function filesOf(dir: string): Record<string, string> {
  const files: Record<string, string> = {}
  for (const entry of readdirSync(dir, { recursive: true, withFileTypes: true })) {
    if (entry.isFile()) {
      const path = join(entry.parentPath, entry.name)
      files[path.slice(dir.length + 1)] = readFileSync(path, 'utf8')
    }
  }
  return files
}

function needValueHome(): string {
  const catalogDir = scratchDir()
  const files: Record<string, string> = {
    'starters/service/starter.yml': 'folders:\n  go:\n    path: .\n',
    'starters/service/files/go/main.go': 'package main\n',
    'features/auth/feature.yml': 'summary: Auth.\ntargets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n',
    'features/client/feature.yml': 'summary: Client.\ntargets: [service]\nneeds:\n  auth:\n    method: api-key\n',
  }
  for (const [path, contents] of Object.entries(files)) {
    mkdirSync(dirname(join(catalogDir, path)), { recursive: true })
    writeFileSync(join(catalogDir, path), contents)
  }
  const home = scratchDir()
  mkdirSync(join(home, 'feature-compose'))
  writeFileSync(join(home, 'feature-compose', 'config.yml'), `catalog: ${catalogDir}\n`)
  return home
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

// startedApi builds the API of a written repo, starts it on a free port, and gives its base URL, which ends with
// /api. The database settings point to no database: the pool of the API opens no connection before the first
// query, and the requests of the tests stop before a query.
async function startedApi(repo: string, env: Record<string, string>): Promise<string> {
  const build = await runProgram(repo, 'go', ['build', '-o', join(repo, 'api-of-the-test'), './cmd/api'], env)
  expect(build.status, build.stdout + build.stderr).toBe(0)
  const port = await freePort()
  const inherited = { ...process.env }
  delete inherited.ALLOWED_ORIGINS
  const api = spawn(join(repo, 'api-of-the-test'), [], {
    cwd: repo,
    env: {
      ...inherited,
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
  const url = `http://127.0.0.1:${port}/api`
  for (let waited = 0; ; waited += 100) {
    if (api.exitCode !== null) throw new Error(`the API stopped with the exit status ${api.exitCode}: ${printed}`)
    if (await answers(url)) return url
    if (waited > 20000) throw new Error(`the API did not answer within 20 seconds: ${printed}`)
    await new Promise((done) => setTimeout(done, 100))
  }
}

// startedFakes builds the fakes of the outside systems of a written repo, and starts them on a free port with no
// delay before a webhook. The fakes have no route at the root path, so the wait takes any answer as the start.
async function startedFakes(repo: string, env: Record<string, string>): Promise<string> {
  const build = await runProgram(repo, 'go', ['build', '-o', join(repo, 'fakes-of-the-test'), './cmd/fake_services'], env)
  expect(build.status, build.stdout + build.stderr).toBe(0)
  const port = await freePort()
  const fakes = spawn(join(repo, 'fakes-of-the-test'), [], { cwd: repo, env: { ...process.env, ...env, PORT: String(port), WEBHOOK_DELAY: '0s' } })
  let printed = ''
  fakes.stdout?.on('data', (chunk) => (printed += chunk))
  fakes.stderr?.on('data', (chunk) => (printed += chunk))
  onTestFinished(() => {
    fakes.kill('SIGKILL')
  })
  const url = `http://127.0.0.1:${port}`
  for (let waited = 0; ; waited += 100) {
    if (fakes.exitCode !== null) throw new Error(`the fakes stopped with the exit status ${fakes.exitCode}: ${printed}`)
    try {
      await fetch(`${url}/network/payments/none`)
      return url
    } catch {
      if (waited > 20000) throw new Error(`the fakes did not answer within 20 seconds: ${printed}`)
      await new Promise((done) => setTimeout(done, 100))
    }
  }
}

async function startedWebhookServer(): Promise<{ url: string; received: { path: string; key: string; body: unknown }[] }> {
  const received: { path: string; key: string; body: unknown }[] = []
  const server = createServer((request, response) => {
    let body = ''
    request.on('data', (chunk) => (body += chunk))
    request.on('end', () => {
      received.push({ path: request.url ?? '', key: String(request.headers['idempotency-key']), body: JSON.parse(body) })
      response.writeHead(200)
      response.end()
    })
  })
  await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
  onTestFinished(() => {
    server.closeAllConnections()
    server.close()
  })
  return { url: `http://127.0.0.1:${(server.address() as AddressInfo).port}`, received }
}

interface TestIssuer {
  url: string
  token: (claims: Record<string, unknown>) => string
  expiresIn: number
  refusesRefresh: boolean
  audiences: string[]
  refreshes: number
}

async function startedTestIssuer(): Promise<TestIssuer> {
  const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 })
  const encoded = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url')
  const issuer: TestIssuer = {
    url: '',
    token: (claims) => {
      const signed = `${encoded({ alg: 'RS256', typ: 'JWT', kid: 'e2e' })}.${encoded({ iss: issuer.url, exp: Math.floor(Date.now() / 1000) + 3600, ...claims })}`
      return `${signed}.${sign('RSA-SHA256', Buffer.from(signed), privateKey).toString('base64url')}`
    },
    expiresIn: 3600,
    refusesRefresh: false,
    audiences: [],
    refreshes: 0,
  }
  const audienceOf = new Map<string, string>()
  let issued = 0
  const tokens = (audience: string) => {
    issued += 1
    audienceOf.set(`refresh-${issued}`, audience)
    return {
      access_token: issuer.token({ aud: audience, exp: Math.floor(Date.now() / 1000) + issuer.expiresIn }),
      refresh_token: `refresh-${issued}`,
      token_type: 'Bearer',
      expires_in: issuer.expiresIn,
    }
  }
  const server = createServer(async (request, response) => {
    let text = ''
    for await (const chunk of request) text += chunk
    const form = new URLSearchParams(text)
    const json = (status: number, value: unknown) => {
      response.writeHead(status, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify(value))
    }
    switch (request.url) {
      case '/.well-known/openid-configuration':
        return json(200, { issuer: issuer.url, jwks_uri: `${issuer.url}/signing-keys`, device_authorization_endpoint: `${issuer.url}/device`, token_endpoint: `${issuer.url}/token` })
      case '/signing-keys':
        return json(200, { keys: [{ ...publicKey.export({ format: 'jwk' }), kid: 'e2e', use: 'sig', alg: 'RS256' }] })
      case '/device': {
        issuer.audiences.push(form.get('audience') ?? '')
        const deviceCode = `device-${issuer.audiences.length}`
        audienceOf.set(deviceCode, form.get('audience') ?? '')
        return json(200, { device_code: deviceCode, user_code: 'ABCD-EFGH', verification_uri: `${issuer.url}/activate`, expires_in: 600, interval: 1 })
      }
      case '/token': {
        if (form.get('grant_type') === 'urn:ietf:params:oauth:grant-type:device_code') {
          const audience = audienceOf.get(form.get('device_code') ?? '')
          return audience === undefined ? json(400, { error: 'invalid_grant' }) : json(200, tokens(audience))
        }
        const audience = audienceOf.get(form.get('refresh_token') ?? '')
        if (form.get('grant_type') !== 'refresh_token' || issuer.refusesRefresh || audience === undefined) return json(400, { error: 'invalid_grant' })
        issuer.refreshes += 1
        return json(200, tokens(audience))
      }
    }
    json(404, {})
  })
  await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
  issuer.url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`
  onTestFinished(() => {
    server.closeAllConnections()
    server.close()
  })
  return issuer
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

// POST /api/accounts with the body {} gets the answer 400 of the handler, because the owner is missing, before any
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

function openAccount(api: string, authorization?: string, origin?: string): Promise<Response> {
  return fetch(`${api}/accounts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...(authorization ? { Authorization: authorization } : {}), ...(origin ? { Origin: origin } : {}) },
    body: '{}',
  })
}

function createUser(api: string, authorization?: string): Promise<Response> {
  return fetch(`${api}/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...(authorization ? { Authorization: authorization } : {}) },
    body: '{}',
  })
}

async function errorAnswersOf(api: string, acceptedAuthorization: string) {
  return answersOf(await openAccount(api), await openAccount(api, acceptedAuthorization))
}

async function answersOf(refused: Response, rejectedByTheHandler: Response) {
  const answerOf = async (response: Response) => {
    const body = (await response.json()) as Record<string, unknown>
    return { status: response.status, contentType: response.headers.get('content-type'), keys: Object.keys(body).sort(), successful: body.successful, messages: body.messages }
  }
  return { refused: await answerOf(refused), rejectedByTheHandler: await answerOf(rejectedByTheHandler) }
}

async function allowedOriginsOf(api: string, origins: string[]): Promise<Record<string, string | null>> {
  const allowed: Record<string, string | null> = {}
  for (const origin of origins) {
    allowed[origin] = (await openAccount(api, undefined, origin)).headers.get('access-control-allow-origin')
  }
  return allowed
}

async function corsAnswersFrom(api: string, origin: string) {
  const preflight = await fetch(`${api}/accounts`, {
    method: 'OPTIONS',
    headers: { Origin: origin, 'Access-Control-Request-Method': 'POST', 'Access-Control-Request-Headers': 'authorization, content-type' },
  })
  const refused = await openAccount(api, undefined, origin)
  return {
    preflight: {
      status: preflight.status,
      allowOrigin: preflight.headers.get('access-control-allow-origin'),
      allowHeaders: preflight.headers.get('access-control-allow-headers'),
    },
    refused: { status: refused.status, allowOrigin: refused.headers.get('access-control-allow-origin') },
  }
}

// gotestsum starts the line of a package with ✓ when its tests pass, and with ∅ when the package has no test
// that the build tags let run.
function passedPackages(output: string): string[] {
  return output
    .split('\n')
    .filter((line) => line.startsWith('✓'))
    .map((line) => line.split(/\s+/)[1])
}

function foldersWithTests(repo: string, folder: string): string[] {
  const testFiles = (readdirSync(join(repo, folder), { recursive: true }) as string[]).filter((path) => path.endsWith('_test.go'))
  return [...new Set(testFiles.map((path) => (dirname(path) === '.' ? folder : `${folder}/${dirname(path)}`)))]
}

function foldersWithUnitTests(repo: string, folder: string): string[] {
  return foldersWithTests(repo, folder).filter((withTests) =>
    readdirSync(join(repo, withTests)).some((file) => file.endsWith('_test.go') && readFileSync(join(repo, withTests, file), 'utf8').startsWith('//go:build unit')),
  )
}

function listIn(yaml: string, key: string): string[] {
  const block = new RegExp(`^( *)${key}:\\n((?:\\1 +- .+\\n)+)`, 'm').exec(yaml)
  return (block?.[2] ?? '')
    .split('\n')
    .map((line) => line.trim().replace(/^- /, ''))
    .filter((item) => item !== '')
}

function checksumsIn(checksums: string): Record<string, string> {
  return Object.fromEntries(
    checksums
      .trim()
      .split('\n')
      .map((line) => line.split(/\s+/).reverse()),
  )
}

function sha256Of(path: string): string {
  return createHash('sha256').update(readFileSync(path)).digest('hex')
}

function tasksCalledBy(repo: string, path: string): string[] {
  return [...readFileSync(join(repo, path), 'utf8').matchAll(/^\s*(?:run: )?task ([a-z0-9:-]+)/gm)].map((match) => match[1])
}

function tasksIn(list: string): string[] {
  return list
    .split('\n')
    .map((line) => /^\* (\S+?):(\s|$)/.exec(line)?.[1])
    .filter((name): name is string => name !== undefined)
}

function detailsPaneOf(screen: string): string {
  return screen
    .split('\n')
    .map((line) => (line.split('││')[1] ?? '').replace(/│\s*$/, '').trim())
    .join(' ')
    .replace(/\s+/g, ' ')
}

async function commandsOfTheBuiltCLI(repo: string, name: string, env: Record<string, string>): Promise<string[]> {
  const build = await runProgram(repo, 'task', ['build'], env)
  expect(build.status, build.stdout + build.stderr).toBe(0)
  const help = await runProgram(repo, join(repo, 'bin', name), [], env)
  expect(help.status, help.stdout + help.stderr).toBe(0)
  return commandsIn(help.stdout)
}

async function commandsOfTheCLIFromTheSource(repo: string, env: Record<string, string>): Promise<string[]> {
  const help = await runProgram(repo, 'task', ['cli', '--', '--help'], env)
  expect(help.status, help.stdout + help.stderr).toBe(0)
  return commandsIn(help.stdout)
}

function commandsIn(help: string): string[] {
  const section = help.split('COMMANDS:\n')[1]?.split('\n\n')[0] ?? ''
  return section
    .split('\n')
    .map((line) => line.trim().split(/\s{2,}/)[0])
    .filter((name) => name !== '')
}
