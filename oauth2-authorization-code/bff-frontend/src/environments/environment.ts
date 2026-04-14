export const environment = {
  appName: 'BFF Demo',
  backendBaseUrl: 'http://localhost:8082',
  loginUrl: 'http://localhost:8082/auth/login',
  logoutUrl: 'http://localhost:8082/auth/logout',
  sessionUrl: 'http://localhost:8082/api/session',
  dataUrl: 'http://localhost:8082/api/data',
  providerLogoutUrl: 'http://localhost:8080/logout',
} as const;
