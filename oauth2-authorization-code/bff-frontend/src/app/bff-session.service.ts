import { HttpClient } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { environment } from '../environments/environment';

type DemoUser = 'alice' | 'bob';

interface BrowserSessionPayload {
  authenticated: boolean;
  user?: {
    sub: string;
    preferred_username: string;
    name: string;
    email: string;
    roles: string[];
  };
  session?: {
    browserHasTokens: boolean;
    tokenStorage: string;
    accessTokenExpiresAt: number;
    refreshTokenAvailable: boolean;
    scope: string;
  };
}

type ResourceDataPayload = Record<string, unknown>;

@Injectable({ providedIn: 'root' })
export class BffBrowserSessionService {
  private readonly http = inject(HttpClient);

  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly browserSession = signal<BrowserSessionPayload | null>(null);
  readonly resourceData = signal<ResourceDataPayload | null>(null);
  readonly authenticated = computed(() => this.browserSession()?.authenticated === true);

  async loadSession(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    try {
      const payload = await firstValueFrom(
        this.http.get<BrowserSessionPayload>(environment.sessionUrl, { withCredentials: true }),
      );
      this.browserSession.set(payload);
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Loading the BFF session failed.'));
    } finally {
      this.busy.set(false);
    }
  }

  async loadProtectedData(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    try {
      const payload = await firstValueFrom(
        this.http.get<ResourceDataPayload>(environment.dataUrl, { withCredentials: true }),
      );
      this.resourceData.set(payload);
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Loading protected data through the BFF failed.'));
    } finally {
      this.busy.set(false);
    }
  }

  beginLogin(user: DemoUser): void {
    const loginUrl = new URL(environment.loginUrl);
    loginUrl.searchParams.set('user', user);
    window.location.assign(loginUrl.toString());
  }

  async logout(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.http.post(
          environment.logoutUrl,
          {},
          {
            withCredentials: true,
          },
        ),
      );
      this.browserSession.set({ authenticated: false });
      this.resourceData.set(null);
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Logging out of the BFF session failed.'));
    } finally {
      this.busy.set(false);
    }
  }
}

function readErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === 'object' &&
    error !== null &&
    'error' in error &&
    typeof (error as { error?: unknown }).error === 'object'
  ) {
    const payload = (error as { error?: Record<string, unknown> }).error;
    const errorDescription = payload?.['error_description'];
    if (typeof errorDescription === 'string' && errorDescription.length > 0) {
      return errorDescription;
    }
    const message = payload?.['error'];
    if (typeof message === 'string' && message.length > 0) {
      return message;
    }
  }

  if (error instanceof Error) {
    return error.message;
  }

  return fallback;
}
