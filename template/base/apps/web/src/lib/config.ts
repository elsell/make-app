import { env } from '$env/dynamic/public';
import { publicEndpointConfig, publicEnvironmentConfig, publicStringConfig } from '@__APP_SLUG__/client-core';

export const environment = publicEnvironmentConfig(env.PUBLIC_APP_ENV, 'PUBLIC_APP_ENV');
const production = environment === 'production';
export const apiURL = publicEndpointConfig(env.PUBLIC_API_URL, 'http://localhost:8080', 'PUBLIC_API_URL', production);
export const oidcIssuer = publicEndpointConfig(env.PUBLIC_OIDC_ISSUER, 'http://localhost:5556/dex', 'PUBLIC_OIDC_ISSUER', production);
export const oidcClientId = publicStringConfig(env.PUBLIC_OIDC_CLIENT_ID, '__APP_SLUG__-web', 'PUBLIC_OIDC_CLIENT_ID', production);
