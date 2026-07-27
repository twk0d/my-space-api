import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

export interface SpotifyConfig {
  clientId: string;
  clientSecret: string;
  refreshToken: string;
  market: string;
}

export interface AppConfig {
  addr: string;
  adminToken: string;
  allowOrigin: string;
  dataPath: string;
  spotify: SpotifyConfig;
}

export function loadDotEnv(): void {
  const path = findDotEnv();
  if (!path) {
    return;
  }

  const lines = readFileSync(path, 'utf8').split(/\r?\n/);
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) {
      continue;
    }

    const separator = line.indexOf('=');
    if (separator < 0) {
      continue;
    }

    const key = line.slice(0, separator).trim();
    const value = line
      .slice(separator + 1)
      .trim()
      .replace(/^['"]|['"]$/g, '');

    if (key && process.env[key] === undefined) {
      process.env[key] = value;
    }
  }
}

export function getConfig(): AppConfig {
  loadDotEnv();

  return {
    addr: envWithDefault('API_ADDR', ':8080'),
    adminToken: process.env.API_ADMIN_TOKEN ?? '',
    allowOrigin: envWithDefault('API_ALLOW_ORIGIN', 'http://localhost:3000'),
    dataPath: envWithDefault('API_DATA_PATH', '.data/api.json'),
    spotify: {
      clientId: process.env.SPOTIFY_CLIENT_ID ?? '',
      clientSecret: process.env.SPOTIFY_CLIENT_SECRET ?? '',
      refreshToken: process.env.SPOTIFY_REFRESH_TOKEN ?? '',
      market: envWithDefault('SPOTIFY_MARKET', 'BR'),
    },
  };
}

export function portFromAddr(addr: string): number {
  const value = addr.trim();
  const portText = value.startsWith(':') ? value.slice(1) : value.split(':').at(-1);
  const port = Number.parseInt(portText ?? '', 10);

  return Number.isFinite(port) && port > 0 ? port : 8080;
}

function envWithDefault(name: string, fallback: string): string {
  return process.env[name] || fallback;
}

function findDotEnv(): string {
  let dir = process.cwd();

  for (;;) {
    const path = join(dir, '.env');
    if (existsSync(path)) {
      return path;
    }

    const parent = dirname(dir);
    if (parent === dir) {
      return '';
    }
    dir = parent;
  }
}
