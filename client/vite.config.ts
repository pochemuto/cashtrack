import {sveltekit} from '@sveltejs/kit/vite';
import {defineConfig, loadEnv} from 'vite';


export default defineConfig(({mode}) => {
    const env = loadEnv(mode, process.cwd());
    const API_URL = `${env.VITE_API_URL}`;
    return {
        plugins: [sveltekit()],
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
