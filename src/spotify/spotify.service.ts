import { Injectable, Logger } from '@nestjs/common';
import { getConfig, SpotifyConfig } from '../config';
import { MusicTrack } from '../data/types';

interface SpotifyCurrentlyResponse {
  is_playing: boolean;
  progress_ms: number;
  item: SpotifyTrack;
}

interface SpotifyTopTracksResponse {
  items: SpotifyTrack[];
}

interface SpotifyTrack {
  name: string;
  duration_ms: number;
  artists: Array<{ name: string }>;
  album: {
    name: string;
    images: Array<{ url: string }>;
  };
  external_urls: {
    spotify: string;
  };
}

@Injectable()
export class SpotifyService {
  private readonly logger = new Logger(SpotifyService.name);
  private readonly config: SpotifyConfig = getConfig().spotify;

  isConfigured(): boolean {
    return Boolean(
      this.config.clientId.trim() &&
        this.config.clientSecret.trim() &&
        this.config.refreshToken.trim(),
    );
  }

  async currentlyPlaying(): Promise<MusicTrack | undefined> {
    if (!this.isConfigured()) {
      return undefined;
    }

    const token = await this.accessToken();
    const query = new URLSearchParams({ additional_types: 'track' });
    if (this.config.market) {
      query.set('market', this.config.market);
    }

    const response = await fetchWithTimeout(
      `https://api.spotify.com/v1/me/player/currently-playing?${query.toString()}`,
      {
        headers: { authorization: `Bearer ${token}` },
      },
    );

    if (response.status === 204) {
      return undefined;
    }
    if (response.status !== 200) {
      throw new Error(`spotify currently playing status ${response.status}`);
    }

    const payload = (await response.json()) as SpotifyCurrentlyResponse;
    if (!payload.is_playing || !payload.item?.name) {
      return undefined;
    }

    const track = toMusicTrack(payload.item);
    track.isPlaying = true;
    track.progressMs = payload.progress_ms;
    track.progressLabel = formatDuration(payload.progress_ms);
    track.progressPercent = progressPercent(payload.progress_ms, track.durationMs ?? 0);

    return track;
  }

  async topTracks(limit: number): Promise<MusicTrack[]> {
    if (!this.isConfigured()) {
      return [];
    }

    const safeLimit = limit <= 0 || limit > 10 ? 5 : limit;
    const token = await this.accessToken();
    const query = new URLSearchParams({
      time_range: 'long_term',
      limit: String(safeLimit),
    });
    if (this.config.market) {
      query.set('market', this.config.market);
    }

    const response = await fetchWithTimeout(
      `https://api.spotify.com/v1/me/top/tracks?${query.toString()}`,
      {
        headers: { authorization: `Bearer ${token}` },
      },
    );

    if (response.status !== 200) {
      throw new Error(`spotify top tracks status ${response.status}`);
    }

    const payload = (await response.json()) as SpotifyTopTracksResponse;
    return (payload.items ?? []).map(toMusicTrack);
  }

  logFallback(error: unknown, message: string): void {
    this.logger.error(message, error instanceof Error ? error.stack : String(error));
  }

  private async accessToken(): Promise<string> {
    const body = new URLSearchParams({
      grant_type: 'refresh_token',
      refresh_token: this.config.refreshToken,
    });
    const credentials = Buffer.from(`${this.config.clientId}:${this.config.clientSecret}`).toString('base64');

    const response = await fetchWithTimeout('https://accounts.spotify.com/api/token', {
      method: 'POST',
      body,
      headers: {
        authorization: `Basic ${credentials}`,
        'content-type': 'application/x-www-form-urlencoded',
      },
    });

    if (response.status !== 200) {
      throw new Error(`spotify token status ${response.status}`);
    }

    const payload = (await response.json()) as { access_token?: string };
    if (!payload.access_token) {
      throw new Error('spotify token response did not include access_token');
    }

    return payload.access_token;
  }
}

export function formatTrackLabel(track: MusicTrack): string {
  if (track.artists.length === 0) {
    return track.name;
  }

  return `${track.name} - ${track.artists.join(', ')}`;
}

export function formatDuration(durationMs: number): string {
  if (durationMs <= 0) {
    return '';
  }

  const totalSeconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

export function progressPercent(progressMs: number, durationMs: number): number {
  if (progressMs <= 0 || durationMs <= 0) {
    return 0;
  }

  return Math.min(Math.floor((progressMs * 100) / durationMs), 100);
}

function toMusicTrack(track: SpotifyTrack): MusicTrack {
  return {
    name: track.name,
    artists: (track.artists ?? []).map((artist) => artist.name).filter(Boolean),
    album: track.album?.name,
    durationMs: track.duration_ms,
    durationLabel: formatDuration(track.duration_ms),
    imageUrl: track.album?.images?.[0]?.url,
    externalUrl: track.external_urls?.spotify,
  };
}

async function fetchWithTimeout(input: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8000);

  try {
    return await fetch(input, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timeout);
  }
}
