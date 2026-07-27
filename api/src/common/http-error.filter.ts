import { ArgumentsHost, Catch, ExceptionFilter, HttpException, HttpStatus } from '@nestjs/common';
import { Response } from 'express';

@Catch()
export class HttpErrorFilter implements ExceptionFilter {
  catch(exception: unknown, host: ArgumentsHost): void {
    const response = host.switchToHttp().getResponse<Response>();

    if (exception instanceof HttpException) {
      const status = exception.getStatus();
      const body = exception.getResponse();
      const message =
        typeof body === 'string'
          ? body
          : Array.isArray((body as { message?: unknown }).message)
            ? ((body as { message: string[] }).message[0] ?? exception.message)
            : ((body as { message?: string }).message ?? exception.message);

      response.status(status).json({ error: message });
      return;
    }

    response.status(HttpStatus.INTERNAL_SERVER_ERROR).json({ error: 'internal server error' });
  }
}
