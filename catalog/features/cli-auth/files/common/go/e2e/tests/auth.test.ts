import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir, type Result } from '../testUtils'

// without these variables, a test reads the config and the API key of the person who runs it
function isolated(extra: Record<string, string> = {}): Record<string, string> {
  return {
    XDG_CONFIG_HOME: scratchDir(),
    {{ upper (snake .Name) }}_API_KEY: '',
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
  })
})
