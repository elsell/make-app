import assert from 'node:assert/strict';
import test from 'node:test';
import { createTranslator, selectLocale } from './index';

test('selects supported base locales and falls back safely', () => {
  assert.equal(selectLocale(['es-MX']), 'es');
  assert.equal(selectLocale(['zz-ZZ']), 'en');
  assert.equal(selectLocale([]), 'en');
});

test('translates interpolation and plurals', () => {
  const translator = createTranslator(['es-MX']);
  assert.equal(translator.t('auth.signedInAs', { name: 'Ada' }), 'Sesión iniciada como Ada');
  assert.equal(translator.t('examples.count', { count: 1 }), '1 ejemplo');
  assert.equal(translator.t('examples.count', { count: 2 }), '2 ejemplos');
});

test('formats numbers and dates with the selected locale', () => {
  const translator = createTranslator(['es']);
  const date = new Date('2026-01-02T12:00:00Z');
  const dateOptions = { dateStyle: 'medium', timeZone: 'UTC' } as const;
  assert.equal(translator.number(1234.5), new Intl.NumberFormat('es').format(1234.5));
  assert.equal(translator.date(date, dateOptions), new Intl.DateTimeFormat('es', dateOptions).format(date));
});
