import type { Handle } from '@sveltejs/kit';
import { selectLocale } from '@__APP_SLUG__/i18n';

export const handle: Handle = async ({ event, resolve }) => {
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
  return response;
};
