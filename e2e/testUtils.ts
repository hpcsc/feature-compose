import { spawn } from 'node:child_process'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { launchTerminal, type Session } from 'tuistory'
import { onTestFinished } from 'vitest'

export function getExecutablePath(): string {
  const executablePath = process.env.EXECUTABLE
  if (!executablePath) {
    throw new Error('EXECUTABLE environment variable is required. Set it to the path of the feature-compose binary.')
  }
  return resolve(executablePath)
}

export function buildTag(): string {
  const tag = process.env.BUILD_TAG
  if (!tag) {
    throw new Error('BUILD_TAG environment variable is required. Set it to the tag the feature-compose binary was built with.')
  }
  return tag
}

export function writeFile(path: string, contents: string): void {
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, contents)
}

export function scratchDir(): string {
  const dir = mkdtempSync(join(tmpdir(), 'feature-compose-e2e-'))
  onTestFinished(() => rmSync(dir, { recursive: true, force: true }))
  return dir
}

export const noGitConfigOfTheMachine = { GIT_CONFIG_GLOBAL: '/dev/null', GIT_CONFIG_NOSYSTEM: '1' }

export function featureComposeCatalogDir(): string {
  const dir = process.env.CATALOG_DIR
  if (!dir) {
    throw new Error('CATALOG_DIR environment variable is required. Set it to the catalog folder of this repo.')
  }
  return resolve(dir)
}

export function configHome(): string {
  return configHomeWith({ hpcsc: featureComposeCatalogDir() })
}

export function configHomeWith(catalogDirs: Record<string, string>): string {
  const home = scratchDir()
  mkdirSync(join(home, 'feature-compose'))
  const entries = Object.entries(catalogDirs).map(([name, dir]) => `  ${name}:\n    source: ${dir}\n`)
  writeFileSync(join(home, 'feature-compose', 'config.yml'), `catalogs:\n${entries.join('')}`)
  return home
}

export interface Result {
  stdout: string
  stderr: string
  status: number | null
}

// runProgram does not block, so a fake server in this process can still answer the
// requests the CLI makes.
export function runProgram(cwd: string, program: string, args: string[] = [], env: Record<string, string> = {}): Promise<Result> {
  return new Promise((done, fail) => {
    const child = spawn(program, args, { cwd, env: { ...process.env, ...env } })
    let stdout = ''
    let stderr = ''
    child.stdout.on('data', (chunk) => (stdout += chunk))
    child.stderr.on('data', (chunk) => (stderr += chunk))
    child.on('error', fail)
    child.on('close', (status) => done({ stdout, stderr, status }))
  })
}

export function runCli(
  cwd: string,
  args: string[] = [],
  env: Record<string, string> = {},
  executable = getExecutablePath(),
): Promise<Result> {
  return runProgram(cwd, executable, args, { XDG_CONFIG_HOME: configHome(), ...env })
}

// openCli starts the CLI in a pseudo-terminal. When the CLI stops, the shell
// writes its exit status on the screen, so a test can wait for EXIT:0.
export async function openCli(cwd: string, args: string[] = [], env: Record<string, string> = {}, rows = 40): Promise<Session> {
  const assignments = Object.entries({ XDG_CONFIG_HOME: configHome(), ...env })
    .map(([name, value]) => `${name}='${value.replaceAll("'", `'\\''`)}'`)
    .join(' ')
  const session = await launchTerminal({
    command: 'sh',
    // The emulator starts in new line mode, where a line feed also goes back to
    // column 1. A real terminal does not, and a full screen program that moves
    // the cursor down with a line feed then draws in the wrong column, so
    // \e[20l turns the mode off before the CLI starts.
    args: ['-c', `printf '\\033[20l'; ${assignments} "${getExecutablePath()}" ${args.join(' ')}; echo "EXIT:$?"`],
    cwd,
    cols: 160,
    rows,
  })
  onTestFinished(() => session.close())
  return session
}
