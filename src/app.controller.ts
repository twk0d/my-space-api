import {
  BadRequestException,
  Body,
  Controller,
  Get,
  InternalServerErrorException,
  NotFoundException,
  Param,
  Patch,
  Post,
  Put,
  Query,
  UseGuards,
} from '@nestjs/common';
import { AdminGuard } from './common/admin.guard';
import { CurrentlyService } from './status/currently.service';
import { JsonStoreService } from './data/json-store.service';
import { Currently, MusicTrack, Signature, SignatureStatus } from './data/types';
import { SpotifyService } from './spotify/spotify.service';

@Controller()
export class HealthController {
  @Get('healthz')
  healthz(): { status: string } {
    return { status: 'ok' };
  }
}

@Controller()
export class StatusController {
  constructor(
    private readonly store: JsonStoreService,
    private readonly currentlyService: CurrentlyService,
  ) {}

  @Get('status/currently')
  getCurrently(): Promise<Currently> {
    return this.currentlyService.getWithSpotify();
  }

  @Put('admin/status/currently')
  @UseGuards(AdminGuard)
  async updateCurrently(@Body() body: Record<string, unknown>): Promise<Currently> {
    assertPlainObject(body);
    assertOnlyKeys(body, ['listening', 'isOnline', 'track']);

    const listening = typeof body.listening === 'string' ? body.listening.trim() : '';
    if (!listening) {
      throw new BadRequestException('listening is required');
    }

    const currently: Currently = {
      listening,
      isOnline: Boolean(body.isOnline),
      track: body.track as Currently['track'],
    };

    try {
      await this.store.updateCurrently(currently);
      return currently;
    } catch {
      throw new InternalServerErrorException('failed to update currently');
    }
  }
}

@Controller()
export class VisitsController {
  constructor(private readonly store: JsonStoreService) {}

  @Post('visits')
  async createVisit(): Promise<{ visitors: number }> {
    try {
      return { visitors: await this.store.incrementVisitors() };
    } catch {
      throw new InternalServerErrorException('failed to record visit');
    }
  }

  @Get('stats/visitors')
  async getVisitors(): Promise<{ visitors: number }> {
    try {
      return { visitors: await this.store.visitorCount() };
    } catch {
      throw new InternalServerErrorException('failed to load visitors');
    }
  }
}

@Controller('spotify')
export class SpotifyController {
  constructor(private readonly spotify: SpotifyService) {}

  @Get('top-tracks')
  async topTracks(@Query('limit') limit?: string): Promise<{ tracks: MusicTrack[] }> {
    try {
      return { tracks: await this.spotify.topTracks(queryInt(limit, 5)) };
    } catch (error) {
      this.spotify.logFallback(error, 'get spotify top tracks');
      return { tracks: [] };
    }
  }
}

@Controller('widgets')
export class WidgetsController {
  constructor(
    private readonly store: JsonStoreService,
    private readonly currentlyService: CurrentlyService,
  ) {}

  @Get('home')
  async home(): Promise<{
    currently: Currently;
    latestSignatures: Signature[];
    visitors: number;
  }> {
    try {
      const currently = await this.currentlyService.withSpotifyListening(await this.store.currently());
      const latestSignatures = await this.store.signatures(SignatureStatus.Approved, 3, 0);
      const visitors = await this.store.visitorCount();

      return { currently, latestSignatures, visitors };
    } catch {
      throw new InternalServerErrorException('failed to load widgets');
    }
  }
}

@Controller('guestbook/signatures')
export class GuestbookController {
  constructor(private readonly store: JsonStoreService) {}

  @Post()
  async createSignature(@Body() body: Record<string, unknown>): Promise<Signature> {
    assertPlainObject(body);
    assertOnlyKeys(body, ['name', 'message']);

    const name = typeof body.name === 'string' ? body.name.trim() : '';
    const message = typeof body.message === 'string' ? body.message.trim() : '';

    if (!name || !message) {
      throw new BadRequestException('name and message are required');
    }
    if (name.length > 48) {
      throw new BadRequestException('name is too long');
    }
    if (message.length > 600) {
      throw new BadRequestException('message is too long');
    }

    try {
      return await this.store.createSignature(name, message, new Date());
    } catch {
      throw new InternalServerErrorException('failed to create signature');
    }
  }

  @Get()
  async listApproved(
    @Query('limit') limit?: string,
    @Query('offset') offset?: string,
  ): Promise<{ signatures: Signature[] }> {
    try {
      return {
        signatures: await this.store.signatures(
          SignatureStatus.Approved,
          queryInt(limit, 10),
          queryInt(offset, 0),
        ),
      };
    } catch {
      throw new InternalServerErrorException('failed to load signatures');
    }
  }

  @Get('latest')
  async latest(@Query('limit') limit?: string): Promise<{ signatures: Signature[] }> {
    try {
      return {
        signatures: await this.store.signatures(SignatureStatus.Approved, queryInt(limit, 3), 0),
      };
    } catch {
      throw new InternalServerErrorException('failed to load signatures');
    }
  }
}

@Controller('admin/guestbook/signatures')
@UseGuards(AdminGuard)
export class AdminGuestbookController {
  constructor(private readonly store: JsonStoreService) {}

  @Get('pending')
  async listPending(
    @Query('limit') limit?: string,
    @Query('offset') offset?: string,
  ): Promise<{ signatures: Signature[] }> {
    try {
      return {
        signatures: await this.store.signatures(
          SignatureStatus.Pending,
          queryInt(limit, 20),
          queryInt(offset, 0),
        ),
      };
    } catch {
      throw new InternalServerErrorException('failed to load signatures');
    }
  }

  @Patch(':id/approve')
  approve(@Param('id') id: string): Promise<Signature> {
    return this.updateStatus(id, SignatureStatus.Approved);
  }

  @Patch(':id/reject')
  reject(@Param('id') id: string): Promise<Signature> {
    return this.updateStatus(id, SignatureStatus.Rejected);
  }

  private async updateStatus(id: string, status: SignatureStatus): Promise<Signature> {
    try {
      const signature = await this.store.updateSignatureStatus(id, status, new Date());
      if (!signature) {
        throw new NotFoundException('signature not found');
      }

      return signature;
    } catch (error) {
      if (error instanceof NotFoundException) {
        throw error;
      }
      throw new InternalServerErrorException('failed to update signature');
    }
  }
}

function queryInt(value: string | undefined, fallback: number): number {
  const parsed = Number.parseInt(value ?? '', 10);
  return Number.isNaN(parsed) ? fallback : parsed;
}

function assertOnlyKeys(body: Record<string, unknown>, keys: string[]): void {
  const allowed = new Set(keys);
  const unexpected = Object.keys(body ?? {}).find((key) => !allowed.has(key));
  if (unexpected) {
    throw new BadRequestException('invalid JSON body');
  }
}

function assertPlainObject(body: unknown): asserts body is Record<string, unknown> {
  if (!body || typeof body !== 'object' || Array.isArray(body)) {
    throw new BadRequestException('invalid JSON body');
  }
}
