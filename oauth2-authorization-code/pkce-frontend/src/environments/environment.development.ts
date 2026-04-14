export const environment = {
  appName: 'PKCE Demo',
  providerBaseUrl: 'http://localhost:8080',
  authorizeUrl: 'http://localhost:8080/authorize',
  tokenUrl: 'http://localhost:8080/token',
  userinfoUrl: 'http://localhost:8080/userinfo',
  revocationUrl: 'http://localhost:8080/revoke',
  providerLogoutUrl: 'http://localhost:8080/logout',
  clientId: 'pkce-spa',
  redirectUri: 'http://localhost:4200/callback',
  scope: 'openid profile email roles offline_access api.read',
  resourceApiUrl: 'http://localhost:8082/api/profile',
} as const;
