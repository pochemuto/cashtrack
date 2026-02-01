import { init, register, getLocaleFromNavigator } from 'svelte-i18n';

register('en', () => import('../locales/en.json'));

export function setupI18n() {
    init({
        fallbackLocale: 'en',
        initialLocale: getLocaleFromNavigator() || 'en'
    });
}
