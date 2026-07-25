// @ts-check
import { defineConfig } from 'astro/config';
import mermaid from 'astro-mermaid';
import starlight from '@astrojs/starlight';
import starlightSidebarTopics from 'starlight-sidebar-topics';

// https://astro.build/config
export default defineConfig({
	site: 'https://afi.fatihcure.com',
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
							link: '/getting-started/quick-start/',
							icon: 'rocket',
							items: [
								{ label: 'Home', slug: '' },
								{
									label: 'Getting started',
									items: [
										{ label: 'Quick start', slug: 'getting-started/quick-start' },
										{ label: 'Local development', slug: 'getting-started/local-dev' },
										{
											label: 'Web UI',
											items: [
												{ label: 'Overview', slug: 'getting-started/web-ui' },
												{ label: 'Policies', slug: 'getting-started/web-ui/policies' },
												{ label: 'MCP and A2A', slug: 'getting-started/web-ui/mcp-a2a' },
											],
										},
										{ label: 'Single sign-on (SSO)', slug: 'getting-started/sso' },
										{
											label: 'Signup and password reset',
											slug: 'getting-started/signup-password-reset',
										},
										{ label: 'Verify', slug: 'getting-started/verify' },
									],
								},
								{
									label: 'Concepts',
									items: [
										{ label: 'Plugins (hooks)', slug: 'hooks/usage' },
										{ label: 'WASM hooks', slug: 'hooks/wasm' },
									],
								},
							],
						},
						{
							id: 'api',
							label: 'API',
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
							link: '/development/architecture/',
							icon: 'seti:config',
							items: [
								{ label: 'Repository layout', slug: 'development/repository-layout' },
								{ label: 'Architecture', slug: 'development/architecture' },
								{ label: 'Providers', slug: 'development/providers' },
								{ label: 'Platform events', slug: 'development/platform-events' },
								{ label: 'Config reference', slug: 'development/config-reference' },
								{ label: 'Testing', slug: 'development/testing' },
							],
						},
						{
							id: 'deployment',
							label: 'Deploy',
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
