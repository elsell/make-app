export default ({ config }: { config: Record<string, unknown> }) => ({
  ...config,
  extra: {
    ...((config.extra as Record<string, unknown> | undefined) ?? {}),
    application: {
      environment: process.env.__ENV_PREFIX___APP_ENV ?? 'development',
      apiURL: process.env.__ENV_PREFIX___API_URL ?? 'http://localhost:8080',
      oidcIssuer: process.env.__ENV_PREFIX___OIDC_ISSUER ?? 'http://localhost:5556/dex',
      oidcClientId: process.env.__ENV_PREFIX___MOBILE_OIDC_CLIENT_ID ?? '__APP_SLUG__-mobile',
    },
  },
});
