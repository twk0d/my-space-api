import { INestApplication } from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import * as request from 'supertest';
import { AppModule } from '../src/app.module';
import { HttpErrorFilter } from '../src/common/http-error.filter';
import { Signature } from '../src/data/types';

describe('API', () => {
  let app: INestApplication;

  beforeEach(async () => {
    const tempDir = await mkdtemp(join(tmpdir(), 'my-space-api-'));
    process.env.API_ADMIN_TOKEN = 'test-token';
    process.env.API_DATA_PATH = join(tempDir, 'api.json');
    delete process.env.SPOTIFY_CLIENT_ID;
    delete process.env.SPOTIFY_CLIENT_SECRET;
    delete process.env.SPOTIFY_REFRESH_TOKEN;

    const moduleRef = await Test.createTestingModule({
      imports: [AppModule],
    }).compile();

    app = moduleRef.createNestApplication();
    app.useGlobalFilters(new HttpErrorFilter());
    await app.init();
  });

  afterEach(async () => {
    await app.close();
  });

  it('returns health status', async () => {
    await request(app.getHttpServer())
      .get('/healthz')
      .expect(200)
      .expect({ status: 'ok' });
  });

  it('supports the guestbook moderation flow', async () => {
    const createResponse = await request(app.getHttpServer())
      .post('/guestbook/signatures')
      .send({
        name: 'Kao',
        message: 'Hello guestbook',
      })
      .expect(201);

    const signature = createResponse.body as Signature;
    expect(signature.status).toBe('pending');

    const latestBeforeApproval = await request(app.getHttpServer())
      .get('/guestbook/signatures/latest')
      .expect(200);
    expect(latestBeforeApproval.body.signatures).toHaveLength(0);

    await request(app.getHttpServer())
      .patch(`/admin/guestbook/signatures/${signature.id}/approve`)
      .set('Authorization', 'Bearer test-token')
      .expect(200);

    const latestAfterApproval = await request(app.getHttpServer())
      .get('/guestbook/signatures/latest?limit=3')
      .expect(200);
    expect(latestAfterApproval.body.signatures).toHaveLength(1);
    expect(latestAfterApproval.body.signatures[0].name).toBe('Kao');
  });

  it('updates currently, increments visits, and returns home widgets', async () => {
    const currentlyResponse = await request(app.getHttpServer())
      .get('/status/currently')
      .expect(200);
    expect(currentlyResponse.body.listening).toBeTruthy();
    expect(currentlyResponse.body.isOnline).toBe(true);

    await request(app.getHttpServer())
      .put('/admin/status/currently')
      .set('Authorization', 'Bearer test-token')
      .send({
        listening: 'New song.mp3',
        isOnline: true,
      })
      .expect(200);

    await request(app.getHttpServer()).post('/visits').expect(201);

    const widgetsResponse = await request(app.getHttpServer())
      .get('/widgets/home')
      .expect(200);
    expect(widgetsResponse.body.currently.listening).toBe('New song.mp3');
    expect(widgetsResponse.body.visitors).toBe(124);
  });

  it('requires the admin bearer token', async () => {
    await request(app.getHttpServer())
      .get('/admin/guestbook/signatures/pending')
      .expect(401)
      .expect({ error: 'invalid admin token' });
  });
});
