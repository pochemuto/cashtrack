import { init, register, getLocaleFromNavigator } from 'svelte-i18n';

register('en', () => import('../locales/en.json'));
register('ru', () => import('../locales/ru.json'));

export function setupI18n() {
    if (typeof window !== 'undefined') {
        const savedLocale = localStorage.getItem('locale');
        init({
            fallbackLocale: 'en',
            initialLocale: savedLocale || getLocaleFromNavigator() || 'en'
        });
    } else {
        init({
            fallbackLocale: 'en',
            initialLocale: 'en'
        });
    }
}
