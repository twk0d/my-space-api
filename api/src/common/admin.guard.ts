import { CanActivate, ExecutionContext, Injectable, ServiceUnavailableException, UnauthorizedException } from '@nestjs/common';
import { Request } from 'express';
import { getConfig } from '../config';

@Injectable()
export class AdminGuard implements CanActivate {
  private readonly adminToken = getConfig().adminToken;

  canActivate(context: ExecutionContext): boolean {
    if (!this.adminToken) {
      throw new ServiceUnavailableException('admin token is not configured');
    }

    const request = context.switchToHttp().getRequest<Request>();
    const authorization = request.header('authorization') ?? '';
    const token = authorization.startsWith('Bearer ') ? authorization.slice('Bearer '.length) : authorization;

    if (token !== this.adminToken) {
      throw new UnauthorizedException('invalid admin token');
    }

    return true;
  }
}
