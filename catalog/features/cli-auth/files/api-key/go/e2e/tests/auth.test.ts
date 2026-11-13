{{- $stored := eq (option "store-key") "true" -}}
{{- if $stored -}}
import { mkdirSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { getExecutablePath, openCli, runCli, scratchDir, type Result } from '../testUtils'
{{- else -}}
import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir, type Result } from '../testUtils'
{{- end }}

{{ if $stored -}}
// without these variables, a test reads the config and the API key of the person who runs it, and writes to their
// OS credential store
{{- else -}}
// without these variables, a test reads the config and the API key of the person who runs it
{{- end }}
function isolated(extra: Record<string, string> = {}): Record<string, string> {
  return {
    XDG_CONFIG_HOME: scratchDir(),
    {{ upper (snake .Name) }}_API_KEY: '',
{{- if $stored }}
    {{ upper (snake .Name) }}_API_URL: 'https://api.example.test',
    {{ upper (snake .Name) }}_CREDENTIAL_STORE: 'file',
{{- end }}
    ...extra,
  }
}

function configDir(env: Record<string, string>): string {
  return join(env.XDG_CONFIG_HOME, '{{ kebab .Name }}')
}

function writeConfig(env: Record<string, string>, text: string): string {
  mkdirSync(configDir(env), { recursive: true })
  const path = join(configDir(env), 'config.yaml')
  writeFileSync(path, text)
  return path
}

function output(result: Result): string {
  return result.stdout + result.stderr
}

function status(env: Record<string, string>): Promise<Result> {
  return runCli(scratchDir(), ['auth', 'status'], env)
}

describe('{{ kebab .Name }} auth status', () => {
  it('names the environment variable that gives the API key, and does not print the API key', async () => {
    const result = await status(isolated({ {{ upper (snake .Name) }}_API_KEY: 'secret-value' }))

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('The API key comes from {{ upper (snake .Name) }}_API_KEY.\n')
  })

  it('runs the credentialCommand in the user config', async () => {
    const env = isolated()
    const path = writeConfig(env, 'credentialCommand: echo from-command\n')

    const result = await status(env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe(`The API key comes from the credentialCommand in ${path}.\n`)
  })

  it('shows what a failed credentialCommand writes to stderr', async () => {
    const env = isolated()
    const path = writeConfig(env, "credentialCommand: echo 'the vault is locked' >&2; exit 1\n")

    const result = await status(env)

    expect(result.status).toBe(1)
    expect(result.stderr).toContain('the vault is locked')
    expect(output(result)).toContain(`run the credentialCommand in ${path}`)
  })

  it('fails and says how to give the API key when there is none', async () => {
    const result = await status(isolated())

    expect(result.status).toBe(1)
    expect(output(result)).toContain('{{ upper (snake .Name) }}_API_KEY')
    expect(output(result)).toContain('credentialCommand')
{{- if $stored }}
    expect(output(result)).toContain('run "{{ kebab .Name }} auth login"')
{{- end }}
  })
})
{{- if $stored }}

function login(env: Record<string, string>, key: string): Promise<Result> {
  return runCli(scratchDir(), ['auth', 'login'], env, getExecutablePath(), key)
}

describe('{{ kebab .Name }} auth', () => {
  it('has the commands status, login and logout, and auth login has no flag for a key', async () => {
    const commands = await runCli(scratchDir(), ['auth', '--help'], isolated())
    const login = await runCli(scratchDir(), ['auth', 'login', '--help'], isolated())

    const section = (help: string, heading: string) =>
      (help.split(`${heading}:\n`)[1] ?? '')
        .split('\n\n')[0]
        .split('\n')
        .map((line) => line.trim().split(/\s{2,}/)[0])
        .filter((name) => name !== '')
    expect(section(commands.stdout, 'COMMANDS')).toEqual(['status', 'login', 'logout'])
    expect(section(login.stdout, 'OPTIONS')).toEqual(['--help, -h'])
  })
})

describe('{{ kebab .Name }} auth login', () => {
  it('stores the API key that it reads from stdin in a file that only the user can read, and auth status names that file', async () => {
    const env = isolated()
    const file = join(configDir(env), 'credentials.json')

    const result = await login(env, 'stored-key')

    expect(result.status).toBe(0)
    expect(result.stdout).toBe(`Stored the API key for api.example.test in ${file}.\n`)
    expect(statSync(file).mode & 0o777).toBe(0o600)
    expect((await status(env)).stdout).toBe(`The API key comes from ${file}.\n`)
  })

  it('in a terminal, asks for the API key, does not show it, and stores it', async () => {
    const env = isolated()
    const session = await openCli(scratchDir(), ['auth', 'login'], env)

    await session.waitForText('Paste the API key:')
    await session.type('typed-key-of-the-test')
    await session.press('enter')
    await session.waitForText('EXIT:0')

    expect(await session.text()).toContain('Stored the API key for api.example.test')
    expect(await session.text()).not.toContain('typed-key-of-the-test')
    expect((await status(env)).stdout).toBe(`The API key comes from ${join(configDir(env), 'credentials.json')}.\n`)
  })

  it('stores no empty API key', async () => {
    const env = isolated()

    const result = await login(env, '  ')

    expect(result.status).toBe(1)
    expect(output(result)).toContain('no API key')
    expect((await status(env)).status).toBe(1)
  })
})

describe('{{ kebab .Name }} auth status with a stored API key', () => {
  it('lets the environment variable win over the stored API key', async () => {
    const env = isolated()
    await login(env, 'stored-key')

    const result = await status({ ...env, {{ upper (snake .Name) }}_API_KEY: 'from-environment' })

    expect(result.stdout).toBe('The API key comes from {{ upper (snake .Name) }}_API_KEY.\n')
  })

  it('does not give the stored API key to another API host', async () => {
    const env = isolated()
    await login(env, 'stored-key')

    const result = await status({ ...env, {{ upper (snake .Name) }}_API_URL: 'https://other.example.test' })

    expect(result.status).toBe(1)
  })
})

describe('{{ kebab .Name }} auth logout', () => {
  it('removes the stored API key', async () => {
    const env = isolated()
    await login(env, 'stored-key')

    const result = await runCli(scratchDir(), ['auth', 'logout'], env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('Removed the API key for api.example.test.\n')
    expect((await status(env)).status).toBe(1)
  })

  it('says so when no API key is stored', async () => {
    const result = await runCli(scratchDir(), ['auth', 'logout'], isolated())

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('No API key for api.example.test is stored.\n')
  })
})
{{- end }}
