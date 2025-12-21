import {sveltekit} from '@sveltejs/kit/vite';
import tailwindcss from "@tailwindcss/vite";
import devtoolsJson from 'vite-plugin-devtools-json';
import {defineConfig, loadEnv} from 'vite';


export default defineConfig(({mode}) => {
    if (mode) {
        // code inside here will be tree-shaken in production builds
        console.log(`Vide mode: ${mode}`)
    }

    const env = loadEnv(mode, process.cwd());
    const API_URL = `${env.VITE_API_URL}`;
    return {
        plugins: [sveltekit(), tailwindcss(), devtoolsJson()],
        server: {
            proxy: {
                '/api': {
                    target: API_URL,
                    changeOrigin: true,
                    rewrite: path => path.replace(/^\/api/, '')
                }
            }
        }
    };
});
