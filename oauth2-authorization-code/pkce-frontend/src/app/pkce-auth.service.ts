import { HttpClient } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { OAuthErrorEvent, OAuthService } from 'angular-oauth2-oidc';
import { firstValueFrom } from 'rxjs';

import { environment } from '../environments/environment';
import { pkceAuthConfig } from './pkce-oauth.config';

type DemoUser = 'alice' | 'bob';

interface BrowserTokenSet {
  access_token: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
  scope: string;
  id_token: string;
}

interface UserInfo {
  sub: string;
  name: string;
  email: string;
  preferred_username: string;
  roles: string[];
}

type ResourceApiPayload = Record<string, unknown>;

@Injectable({ providedIn: 'root' })
export class PkceBrowserFlowService {
  private readonly http = inject(HttpClient);
  private readonly oauthService = inject(OAuthService);
  private readonly refreshLeadTimeMs = 30_000;
  private initializationPromise: Promise<void> | null = null;
  private readonly authSnapshotVersion = signal(0);

  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly userInfo = signal<UserInfo | null>(null);
  readonly resourceApiPayload = signal<ResourceApiPayload | null>(null);

  readonly authenticated = computed(() => {
    this.authSnapshotVersion();
    return this.oauthService.hasValidAccessToken();
  });
  readonly tokenSet = computed<BrowserTokenSet | null>(() => {
    this.authSnapshotVersion();

    const accessToken = this.oauthService.getAccessToken();
    const idToken = this.oauthService.getIdToken();
    if (!accessToken || !idToken) {
      return null;
    }

    return {
      access_token: accessToken,
      token_type: 'Bearer',
      expires_in: calculateExpiresIn(this.oauthService.getAccessTokenExpiration()),
      refresh_token: this.oauthService.getRefreshToken() || undefined,
      scope: formatGrantedScopes(this.oauthService.getGrantedScopes()),
      id_token: idToken,
    };
  });
  readonly idTokenClaims = computed(() => {
    this.authSnapshotVersion();
    return asRecord(this.oauthService.getIdentityClaims());
  });
  readonly accessTokenClaims = computed(() =>
    decodeJwtPayload(this.tokenSet()?.access_token ?? null),
  );
  readonly hasRefreshToken = computed(() => {
    this.authSnapshotVersion();
    return Boolean(this.oauthService.getRefreshToken());
  });

  constructor() {
    this.oauthService.configure(pkceAuthConfig);
    this.oauthService.setStorage(sessionStorage);

    this.oauthService.events.subscribe((event) => {
      this.syncAuthSnapshot();

      if (event.type === 'token_received') {
        this.error.set(null);
        void this.loadUserInfo();
        return;
      }

      if (event.type === 'logout') {
        this.userInfo.set(null);
        this.resourceApiPayload.set(null);
        return;
      }

      if (event instanceof OAuthErrorEvent) {
        this.error.set(readErrorMessage(event.reason, 'OAuth processing failed.'));
      }
    });

    void this.initializeAuth();
  }

  async initializeAuth(): Promise<void> {
    if (!this.initializationPromise) {
      this.initializationPromise = this.initializeAuthInternal();
    }
    await this.initializationPromise;
  }

  async beginLogin(user: DemoUser): Promise<void> {
    await this.initializeAuth();
    this.busy.set(true);
    this.error.set(null);
    this.resourceApiPayload.set(null);

    try {
      this.oauthService.initLoginFlow('', { login_hint: user });
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Failed to start the PKCE flow.'));
      this.busy.set(false);
    }
  }

  async loadUserInfo(): Promise<void> {
    const hasAccessToken = await this.ensureFreshAccessToken();
    if (!hasAccessToken) {
      return;
    }

    try {
      const userInfo = normalizeUserInfo(await this.oauthService.loadUserProfile());
      this.userInfo.set(userInfo);
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Fetching userinfo failed.'));
    }
  }

  async callResourceApi(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);

    const hasAccessToken = await this.ensureFreshAccessToken();
    if (!hasAccessToken) {
      this.error.set('No access token is available in browser storage.');
      this.busy.set(false);
      return;
    }

    try {
      const payload = await firstValueFrom(
        this.http.get<ResourceApiPayload>(environment.resourceApiUrl),
      );
      this.resourceApiPayload.set(payload);
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Calling the resource API failed.'));
    } finally {
      this.busy.set(false);
    }
  }

  async logout(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);

    try {
      if (this.oauthService.getAccessToken()) {
        await this.oauthService.revokeTokenAndLogout(true);
      } else {
        this.oauthService.logOut(true);
      }
      this.userInfo.set(null);
      this.resourceApiPayload.set(null);
      this.syncAuthSnapshot();
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Logging out of the PKCE flow failed.'));
    } finally {
      this.busy.set(false);
    }
  }

  private async initializeAuthInternal(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);

    try {
      await this.oauthService.loadDiscoveryDocumentAndTryLogin();
      this.oauthService.setupAutomaticSilentRefresh();

      if (this.oauthService.hasValidAccessToken()) {
        await this.loadUserInfo();
      }
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Initializing OAuth login failed.'));
    } finally {
      this.syncAuthSnapshot();
      this.busy.set(false);
    }
  }

  private async ensureFreshAccessToken(): Promise<boolean> {
    if (!this.oauthService.getAccessToken()) {
      return false;
    }

    if (!this.isAccessTokenExpiringSoon()) {
      return this.oauthService.hasValidAccessToken();
    }

    if (!this.oauthService.getRefreshToken()) {
      return this.oauthService.hasValidAccessToken();
    }

    try {
      await this.oauthService.refreshToken();
      this.syncAuthSnapshot();
      return this.oauthService.hasValidAccessToken();
    } catch (error: unknown) {
      this.error.set(readErrorMessage(error, 'Refreshing the token failed.'));
      return false;
    }
  }

  private isAccessTokenExpiringSoon(): boolean {
    const accessTokenExp = this.oauthService.getAccessTokenExpiration();
    if (!accessTokenExp) {
      return false;
    }

    return accessTokenExp <= Date.now() + this.refreshLeadTimeMs;
  }

  private syncAuthSnapshot(): void {
    this.authSnapshotVersion.update((version) => version + 1);
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

function decodeJwtPayload(token: string | null): Record<string, unknown> | null {
  if (!token) {
    return null;
  }

  const parts = token.split('.');
  if (parts.length !== 3) {
    return null;
  }

  const normalized = parts[1].replace(/-/g, '+').replace(/_/g, '/');
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=');

  try {
    return JSON.parse(window.atob(padded)) as Record<string, unknown>;
  } catch {
    return null;
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function calculateExpiresIn(expiresAt: number | null): number {
  if (!expiresAt) {
    return 0;
  }
  return Math.max(Math.floor((expiresAt - Date.now()) / 1000), 0);
}

function formatGrantedScopes(scopes: unknown): string {
  if (Array.isArray(scopes)) {
    return scopes.filter((scope): scope is string => typeof scope === 'string').join(' ');
  }

  if (typeof scopes === 'string') {
    return scopes;
  }

  return '';
}

function normalizeUserInfo(value: unknown): UserInfo {
  const record = asRecord(value) ?? {};

  return {
    sub: typeof record['sub'] === 'string' ? record['sub'] : '',
    name: typeof record['name'] === 'string' ? record['name'] : '',
    email: typeof record['email'] === 'string' ? record['email'] : '',
    preferred_username:
      typeof record['preferred_username'] === 'string' ? record['preferred_username'] : '',
    roles: Array.isArray(record['roles'])
      ? record['roles'].filter((role): role is string => typeof role === 'string')
      : [],
  };
}
