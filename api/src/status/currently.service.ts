import { Injectable, InternalServerErrorException } from '@nestjs/common';
import { Currently } from '../data/types';
import { JsonStoreService } from '../data/json-store.service';
import { formatTrackLabel, SpotifyService } from '../spotify/spotify.service';

@Injectable()
export class CurrentlyService {
  constructor(
    private readonly store: JsonStoreService,
    private readonly spotify: SpotifyService,
  ) {}

  async getWithSpotify(): Promise<Currently> {
    try {
      return await this.withSpotifyListening(await this.store.currently());
    } catch {
      throw new InternalServerErrorException('failed to load currently');
    }
  }

  async withSpotifyListening(currently: Currently): Promise<Currently> {
    try {
      const track = await this.spotify.currentlyPlaying();
      if (!track) {
        return currently;
      }

      return {
        ...currently,
        listening: formatTrackLabel(track),
        track,
      };
    } catch (error) {
      this.spotify.logFallback(error, 'get spotify currently playing');
      return currently;
    }
  }
}
