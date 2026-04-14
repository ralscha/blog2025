import { AuthConfig } from 'angular-oauth2-oidc';

import { environment } from '../environments/environment';

export const pkceAuthConfig: AuthConfig = {
  issuer: environment.providerBaseUrl,
  redirectUri: environment.redirectUri,
  clientId: environment.clientId,
  responseType: 'code',
  scope: environment.scope,
  requireHttps: false,
  showDebugInformation: false,
  timeoutFactor: 0.75,
};
