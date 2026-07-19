import type { Handle } from '@sveltejs/kit';
import { selectLocale } from '@__APP_SLUG__/i18n';
import { webConfig } from '$lib/server/config';

export const handle: Handle = async ({ event, resolve }) => {
  const { apiURL, oidcIssuer } = webConfig;
  const accepted = (event.request.headers.get('accept-language') ?? '')
    .split(',')
    .map((entry, index) => {
      const [tag, ...parameters] = entry.split(';').map((part) => part.trim());
      const qualityParameter = parameters.find((parameter) => parameter.startsWith('q='));
      const qualityText = qualityParameter?.slice(2);
      const quality = qualityText === undefined
        ? 1
        : /^(?:0(?:\.\d{0,3})?|1(?:\.0{0,3})?)$/.test(qualityText)
          ? Number(qualityText)
          : 0;
      return { tag, quality, index };
    })
    .filter(({ tag, quality }) => tag !== '*' && quality > 0)
    .sort((left, right) => right.quality - left.quality || left.index - right.index)
    .map(({ tag }) => tag);
  event.locals.locale = selectLocale(accepted);
  const response = await resolve(event, {
    transformPageChunk: ({ html }) => html.replace('lang="en"', `lang="${event.locals.locale}"`)
  });
  response.headers.append('Vary', 'Accept-Language');
  const connectOrigins = new Set<string>(["'self'"]);
  for (const configured of [apiURL, oidcIssuer]) connectOrigins.add(new URL(configured).origin);
  const generatedCSP = response.headers.get('Content-Security-Policy');
  response.headers.set(
    'Content-Security-Policy',
    generatedCSP?.replace(/connect-src [^;]+/, ['connect-src', ...connectOrigins].join(' ')) ?? "default-src 'none'"
  );
  response.headers.set('Referrer-Policy', 'no-referrer');
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
  response.headers.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
  return response;
};
