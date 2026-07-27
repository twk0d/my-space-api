import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { getConfig, portFromAddr } from './config';
import { HttpErrorFilter } from './common/http-error.filter';

async function bootstrap(): Promise<void> {
  const config = getConfig();
  const app = await NestFactory.create(AppModule);

  if (config.allowOrigin) {
    app.enableCors({
      origin: config.allowOrigin,
      allowedHeaders: ['Authorization', 'Content-Type'],
      methods: ['GET', 'POST', 'PUT', 'PATCH', 'OPTIONS'],
    });
  }

  app.useGlobalFilters(new HttpErrorFilter());

  const port = portFromAddr(config.addr);
  await app.listen(port);
}

void bootstrap();
