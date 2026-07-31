// @ts-check
import { defineConfig } from 'astro/config';
import mermaid from 'astro-mermaid';
import starlight from '@astrojs/starlight';
import starlightSidebarTopics from 'starlight-sidebar-topics';

// https://astro.build/config
export default defineConfig({
	site: 'https://afi.fatihcure.com',
	redirects: {
		'/getting-started/quick-start': '/guides/quick-start/',
		'/getting-started/local-dev': '/development/local-dev/',
		'/getting-started/web-ui': '/guides/web-ui/',
		'/getting-started/web-ui/policies': '/guides/web-ui/policies/',
		'/getting-started/web-ui/mcp-a2a': '/guides/web-ui/mcp-a2a/',
		'/getting-started/sso': '/guides/sso/',
		'/getting-started/signup-password-reset': '/guides/signup-password-reset/',
		'/getting-started/verify': '/guides/verify/',
		'/hooks/usage': '/development/hooks/usage/',
		'/hooks/wasm': '/development/hooks/wasm/',
	},
	integrations: [
		// Must run before Starlight so ```mermaid blocks are not treated as code.
		mermaid({
			theme: 'dark',
			autoTheme: true,
		}),
		starlight({
			title: 'AFI Docs',
			description:
				'Self-hostable, cloud-native LLM gateway — control plane, data plane, and platform docs.',
			logo: {
				src: './src/assets/logo-mark.svg',
				alt: 'AFI',
				replacesTitle: true,
			},
			favicon: '/favicon.ico',
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/curefatih/afi',
				},
			],
			customCss: ['./src/styles/custom.css'],
			components: {
				Header: './src/components/Header.astro',
				Sidebar: './src/components/Sidebar.astro',
			},
			head: [
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap',
					},
				},
			],
			editLink: {
				baseUrl: 'https://github.com/curefatih/afi/edit/main/docs/',
			},
			plugins: [
				starlightSidebarTopics(
					[
						{
							id: 'guides',
							label: 'Guides',
							link: '/guides/quick-start/',
							icon: 'rocket',
							items: [
								{ label: 'Home', slug: '' },
								{ label: 'Quick start', slug: 'guides/quick-start' },
								{
									label: 'Web UI',
									items: [
										{ label: 'Overview', slug: 'guides/web-ui' },
										{ label: 'Policies', slug: 'guides/web-ui/policies' },
										{ label: 'MCP and A2A', slug: 'guides/web-ui/mcp-a2a' },
									],
								},
								{ label: 'Single sign-on (SSO)', slug: 'guides/sso' },
								{
									label: 'Signup and password reset',
									slug: 'guides/signup-password-reset',
								},
								{ label: 'Verify', slug: 'guides/verify' },
							],
						},
						{
							id: 'api',
							label: 'API Reference',
							link: '/api/',
							icon: 'document',
							badge: { text: 'v1', variant: 'tip' },
							items: [
								{ label: 'Overview', slug: 'api' },
								{ label: 'Dialects', slug: 'api/dialects' },
								{ label: 'Platform', slug: 'api/platform' },
								{ label: 'Gateway', slug: 'api/gateway' },
							],
						},
						{
							id: 'development',
							label: 'Development',
							link: '/development/local-dev/',
							icon: 'seti:config',
							items: [
								{ label: 'Local development', slug: 'development/local-dev' },
								{ label: 'Architecture', slug: 'development/architecture' },
								{ label: 'Repository layout', slug: 'development/repository-layout' },
								{ label: 'Providers', slug: 'development/providers' },
								{ label: 'Plugins (hooks)', slug: 'development/hooks/usage' },
								{ label: 'WASM hooks', slug: 'development/hooks/wasm' },
								{ label: 'Platform events', slug: 'development/platform-events' },
								{ label: 'Config reference', slug: 'development/config-reference' },
								{ label: 'Testing', slug: 'development/testing' },
							],
						},
						{
							id: 'deployment',
							label: 'Deployment',
							link: '/deployment/',
							icon: 'cloud-download',
							items: [
								{ label: 'Overview', slug: 'deployment' },
								{ label: 'Customization', slug: 'deployment/customization' },
								{ label: 'Docker Compose', slug: 'deployment/docker' },
								{ label: 'Binaries', slug: 'deployment/binary' },
								{ label: 'Observability', slug: 'deployment/observability' },
							],
						},
					],
					{
						exclude: ['404', '404.md'],
					},
				),
			],
		}),
	],
});
