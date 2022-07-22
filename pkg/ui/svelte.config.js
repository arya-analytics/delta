import adapter from '@sveltejs/adapter-static';
import preprocess from 'svelte-preprocess';

const config = {
	preprocess: preprocess(),

	kit: {
		adapter: adapter({
			pages: 'dist',
			assets: 'dist',
			fallback: null,
			precompress: false,
		}),
		prerender: {
			default: true,
		}
	}
};

export default config;
