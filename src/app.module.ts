import { Module } from '@nestjs/common';
import { AdminGuard } from './common/admin.guard';
import {
  AdminGuestbookController,
  GuestbookController,
  HealthController,
  SpotifyController,
  StatusController,
  VisitsController,
  WidgetsController,
} from './app.controller';
import { JsonStoreService } from './data/json-store.service';
import { SpotifyService } from './spotify/spotify.service';
import { CurrentlyService } from './status/currently.service';

@Module({
  controllers: [
    AdminGuestbookController,
    GuestbookController,
    HealthController,
    SpotifyController,
    StatusController,
    VisitsController,
    WidgetsController,
  ],
  providers: [AdminGuard, CurrentlyService, JsonStoreService, SpotifyService],
})
export class AppModule {}
