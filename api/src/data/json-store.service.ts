import { Injectable, OnModuleInit } from '@nestjs/common';
import { mkdir, readFile, rename, writeFile } from 'node:fs/promises';
import { dirname } from 'node:path';
import { getConfig } from '../config';
import { Currently, Database, Signature, SignatureStatus } from './types';

@Injectable()
export class JsonStoreService implements OnModuleInit {
  private data: Database = defaultDatabase();
  private queue: Promise<unknown> = Promise.resolve();
  private readonly path = getConfig().dataPath;

  async onModuleInit(): Promise<void> {
    await this.runLocked(async () => {
      await this.loadLocked();
    });
  }

  currently(): Promise<Currently> {
    return this.runLocked(async () => ({ ...this.data.currently }));
  }

  updateCurrently(currently: Currently): Promise<void> {
    return this.runLocked(async () => {
      this.data.currently = { ...currently };
      await this.saveLocked();
    });
  }

  createSignature(name: string, message: string, now: Date): Promise<Signature> {
    return this.runLocked(async () => {
      const signature: Signature = {
        id: newId(now),
        name,
        message,
        status: SignatureStatus.Pending,
        createdAt: new Date(now),
      };

      this.data.signatures.push(signature);
      await this.saveLocked();

      return { ...signature };
    });
  }

  signatures(status: SignatureStatus, limit: number, offset: number): Promise<Signature[]> {
    return this.runLocked(async () => {
      const matches = this.data.signatures
        .filter((signature) => signature.status === status)
        .reverse();

      return paginate(matches, limit, offset).map((signature) => ({ ...signature }));
    });
  }

  updateSignatureStatus(id: string, status: SignatureStatus, now: Date): Promise<Signature | undefined> {
    return this.runLocked(async () => {
      const index = this.data.signatures.findIndex((signature) => signature.id === id);
      if (index < 0) {
        return undefined;
      }

      const signature: Signature = {
        ...this.data.signatures[index],
        status,
        approvedAt: undefined,
        rejectedAt: undefined,
      };

      const timestamp = new Date(now);
      if (status === SignatureStatus.Approved) {
        signature.approvedAt = timestamp;
      }
      if (status === SignatureStatus.Rejected) {
        signature.rejectedAt = timestamp;
      }

      this.data.signatures[index] = signature;
      await this.saveLocked();

      return { ...signature };
    });
  }

  incrementVisitors(): Promise<number> {
    return this.runLocked(async () => {
      this.data.visitorCount += 1;
      await this.saveLocked();

      return this.data.visitorCount;
    });
  }

  visitorCount(): Promise<number> {
    return this.runLocked(async () => this.data.visitorCount);
  }

  private async loadLocked(): Promise<void> {
    try {
      const contents = await readFile(this.path, 'utf8');
      this.data = reviveDatabase(JSON.parse(contents) as Database);
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'ENOENT') {
        throw error;
      }
      this.data = defaultDatabase();
      await this.saveLocked();
    }
  }

  private async saveLocked(): Promise<void> {
    await mkdir(dirname(this.path), { recursive: true });
    const temporaryPath = `${this.path}.tmp`;
    await writeFile(temporaryPath, `${JSON.stringify(this.data, null, 2)}\n`, 'utf8');
    await rename(temporaryPath, this.path);
  }

  private runLocked<T>(operation: () => Promise<T>): Promise<T> {
    const next = this.queue.then(operation, operation);
    this.queue = next.catch(() => undefined);

    return next;
  }
}

export function defaultDatabase(): Database {
  return {
    currently: {
      listening: 'Shoegaze mix.mp3',
      isOnline: true,
    },
    signatures: [],
    visitorCount: 123,
  };
}

export function paginate<T>(items: T[], limit: number, offset: number): T[] {
  const safeOffset = offset < 0 ? 0 : offset;
  const safeLimit = limit <= 0 || limit > 50 ? 10 : limit;
  if (safeOffset >= items.length) {
    return [];
  }

  return items.slice(safeOffset, safeOffset + safeLimit);
}

function newId(now: Date): string {
  const pad = (value: number, length: number) => value.toString().padStart(length, '0');

  return [
    now.getUTCFullYear(),
    pad(now.getUTCMonth() + 1, 2),
    pad(now.getUTCDate(), 2),
    pad(now.getUTCHours(), 2),
    pad(now.getUTCMinutes(), 2),
    pad(now.getUTCSeconds(), 2),
    '.',
    pad(now.getUTCMilliseconds(), 3),
    '000000',
  ].join('');
}

function reviveDatabase(data: Database): Database {
  return {
    currently: data.currently ?? defaultDatabase().currently,
    signatures: (data.signatures ?? []).map((signature) => ({
      ...signature,
      createdAt: new Date(signature.createdAt),
      approvedAt: signature.approvedAt ? new Date(signature.approvedAt) : undefined,
      rejectedAt: signature.rejectedAt ? new Date(signature.rejectedAt) : undefined,
    })),
    visitorCount: data.visitorCount ?? 0,
  };
}
