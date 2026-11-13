
import { createHash } from 'node:crypto'
import { mkdirSync, statSync, writeFileSync } from 'node:fs'
import { createServer, type IncomingMessage } from 'node:http'
import type { AddressInfo } from 'node:net'
import { join } from 'node:path'
import { describe, expect, it, onTestFinished } from 'vitest'
import { runCli, scratchDir, type Result } from '../testUtils'

// without these variables, a test reads and writes the config, the OS credential store and the OAuth settings of the person who runs it
function isolated(extra: Record<string, string> = {}): Record<string, string> {
  return {
    XDG_CONFIG_HOME: scratchDir(),
    {{ upper (snake .Name) }}_TOKEN: '',
    {{ upper (snake .Name) }}_API_URL: 'https://api.example.test',
    {{ upper (snake .Name) }}_CREDENTIAL_STORE: 'file',
    {{ upper (snake .Name) }}_OAUTH_ISSUER_URL: '',
    {{ upper (snake .Name) }}_OAUTH_CLIENT_ID: '',
    {{ upper (snake .Name) }}_OAUTH_AUDIENCE: '',
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
  it('names the environment variable that gives the token, and does not print the token', async () => {
    const result = await status(isolated({ {{ upper (snake .Name) }}_TOKEN: 'secret-value' }))

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('The token comes from {{ upper (snake .Name) }}_TOKEN.\n')
  })

  it('runs the credentialCommand in the user config', async () => {
    const env = isolated()
    const path = writeConfig(env, 'credentialCommand: echo from-command\n')

    const result = await status(env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe(`The token comes from the credentialCommand in ${path}.\n`)
  })

  it('shows what a failed credentialCommand writes to stderr', async () => {
    const env = isolated()
    const path = writeConfig(env, "credentialCommand: echo 'the vault is locked' >&2; exit 1\n")

    const result = await status(env)

    expect(result.status).toBe(1)
    expect(result.stderr).toContain('the vault is locked')
    expect(output(result)).toContain(`run the credentialCommand in ${path}`)
  })

  it('fails and says how to give the token when there is none', async () => {
    const result = await status(isolated())

    expect(result.status).toBe(1)
    expect(output(result)).toContain('{{ upper (snake .Name) }}_TOKEN')
    expect(output(result)).toContain('credentialCommand')
    expect(output(result)).toContain('"{{ kebab .Name }} auth login"')
  })
})

interface TestIssuer {
  url: string
  expiresIn: number
  verifiedLogins: number
  audiences: string[]
  revoked: string[]
}

function body(request: IncomingMessage): Promise<URLSearchParams> {
  return new Promise((done) => {
    let text = ''
    request.on('data', (chunk) => (text += chunk))
    request.on('end', () => done(new URLSearchParams(text)))
  })
}

async function testIssuer(): Promise<TestIssuer> {
  const issuer: TestIssuer = { url: '', expiresIn: 3600, verifiedLogins: 0, audiences: [], revoked: [] }
  const challenges = new Map<string, string>()
  let issued = 0
  const claims = Buffer.from(JSON.stringify({ sub: 'user-1', email: 'ada@example.com' })).toString('base64url')

  const server = createServer(async (request, response) => {
    const url = new URL(request.url ?? '/', issuer.url)
    const form = await body(request)
    const json = (status: number, value: unknown) => {
      response.statusCode = status
      response.setHeader('content-type', 'application/json')
      response.end(JSON.stringify(value))
    }
    const tokens = () => {
      issued += 1
      json(200, {
        access_token: `access-${issued}`,
        refresh_token: `refresh-${issued}`,
        token_type: 'Bearer',
        expires_in: issuer.expiresIn,
        id_token: `eyJhbGciOiJub25lIn0.${claims}.`,
      })
    }
    switch (url.pathname) {
      case '/.well-known/openid-configuration':
        return json(200, {
          issuer: issuer.url,
          authorization_endpoint: `${issuer.url}/authorize`,
          token_endpoint: `${issuer.url}/token`,
          device_authorization_endpoint: `${issuer.url}/device`,
          revocation_endpoint: `${issuer.url}/revoke`,
        })
      case '/authorize': {
        issuer.audiences.push(url.searchParams.get('audience') ?? '')
        const code = `code-${challenges.size + 1}`
        challenges.set(code, url.searchParams.get('code_challenge') ?? '')
        const redirect = new URL(url.searchParams.get('redirect_uri') ?? '')
        redirect.searchParams.set('code', code)
        redirect.searchParams.set('state', url.searchParams.get('state') ?? '')
        response.writeHead(302, { location: redirect.toString() })
        return response.end()
      }
      case '/device':
        issuer.audiences.push(form.get('audience') ?? '')
        return json(200, {
          device_code: 'device-1',
          user_code: 'ABCD-EFGH',
          verification_uri: `${issuer.url}/activate`,
          expires_in: 600,
          interval: 1,
        })
      case '/token':
        switch (form.get('grant_type')) {
          case 'authorization_code': {
            const challenge = createHash('sha256').update(form.get('code_verifier') ?? '').digest('base64url')
            if (challenges.get(form.get('code') ?? '') !== challenge) return json(400, { error: 'invalid_grant' })
            issuer.verifiedLogins += 1
            return tokens()
          }
          case 'urn:ietf:params:oauth:grant-type:device_code':
            return tokens()
          case 'refresh_token':
            if (form.get('refresh_token') !== `refresh-${issued}`) return json(400, { error: 'invalid_grant' })
            return tokens()
        }
        return json(400, { error: 'unsupported_grant_type' })
      case '/revoke':
        issuer.revoked.push(form.get('token') ?? '')
        return response.end()
    }
    response.statusCode = 404
    response.end()
  })
  await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
  onTestFinished(() => new Promise<void>((done) => server.close(() => done())))
  issuer.url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`
  return issuer
}

function withIssuer(issuer: TestIssuer, extra: Record<string, string> = {}): Record<string, string> {
  return isolated({
    {{ upper (snake .Name) }}_OAUTH_ISSUER_URL: issuer.url,
    {{ upper (snake .Name) }}_OAUTH_CLIENT_ID: 'test-client',
    {{ upper (snake .Name) }}_OAUTH_AUDIENCE: 'my-api',
    BROWSER: '',
    SSH_CONNECTION: '',
    SSH_TTY: '',
    ...extra,
  })
}

function fakeBrowser(): string {
  const script = join(scratchDir(), 'browser.mjs')
  writeFileSync(script, 'await fetch(process.argv[2])\n')
  return `node ${script}`
}

function login(env: Record<string, string>, args: string[] = ['--device']): Promise<Result> {
  return runCli(scratchDir(), ['auth', 'login', ...args], env)
}

describe('{{ kebab .Name }} auth login', () => {
  it('--device prints the code to enter, asks for a token for the audience of the API, and auth status shows the account', async () => {
    const issuer = await testIssuer()
    const env = withIssuer(issuer)
    const file = join(configDir(env), 'credentials.json')

    const result = await login(env)

    expect(result.status).toBe(0)
    expect(issuer.audiences).toEqual(['my-api'])
    expect(result.stderr).toContain('enter the code ABCD-EFGH')
    expect(result.stdout).toBe(`Logged in as ada@example.com. The tokens are in ${file}.\n`)
    expect(statSync(file).mode & 0o777).toBe(0o600)
    const shown = await status(env)
    expect(shown.stdout).toContain(`The token comes from ${file}.\nAccount: ada@example.com\n`)
  })

  it('opens the browser, the issuer gets the PKCE code verifier, and the login asks for a token for the audience of the API', async () => {
    const issuer = await testIssuer()
    const env = withIssuer(issuer, { BROWSER: fakeBrowser() })

    const result = await login(env, [])

    expect(result.status).toBe(0)
    expect(result.stdout).toContain('Logged in as ada@example.com.')
    expect(issuer.verifiedLogins).toBe(1)
    expect(issuer.audiences).toEqual(['my-api'])
  })

  it('has the flag --device, and no flag for a token or a setting', async () => {
    const help = await runCli(scratchDir(), ['auth', 'login', '--help'], isolated())

    const flags = (help.stdout.split('OPTIONS:\n')[1] ?? '').split('\n').map((line) => line.trim().split(/\s{2,}/)[0]).filter((flag) => flag.startsWith('-'))
    expect(flags).toEqual(['--device', '--help, -h'])
  })

  it('without the issuer settings, names the constants to set', async () => {
    const result = await login(isolated())

    expect(result.status).toBe(1)
    expect(output(result)).toContain('set issuerURL, clientID and audience in internal/credential/oauth_setting.go')
  })
})

describe('{{ kebab .Name }} auth token', () => {
  it('prints the access token, and gets a new one when it expires', async () => {
    const issuer = await testIssuer()
    const env = withIssuer(issuer)
    issuer.expiresIn = 1
    await login(env)
    issuer.expiresIn = 3600

    const result = await runCli(scratchDir(), ['auth', 'token'], env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('access-2\n')
  })
})

describe('{{ kebab .Name }} auth logout', () => {
  it('revokes the refresh token and removes the tokens', async () => {
    const issuer = await testIssuer()
    const env = withIssuer(issuer)
    await login(env)

    const result = await runCli(scratchDir(), ['auth', 'logout'], env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('Removed the tokens for api.example.test.\n')
    expect(issuer.revoked).toEqual(['refresh-1'])
    expect((await status(env)).status).toBe(1)
  })
})
