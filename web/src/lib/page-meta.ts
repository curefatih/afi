const APP_NAME = "AFI";

export type PageTitleOptions = {
	/** Overrides the default document description for this route. */
	description?: string;
};

/**
 * Shared route metadata for breadcrumbs (`staticData.getTitle`) and
 * document head (`head`) via TanStack Router.
 */
export function pageTitle(title: string, options: PageTitleOptions = {}) {
	const documentTitle = `${title} · ${APP_NAME}`;
	return {
		staticData: {
			getTitle: () => title,
		},
		head: () => ({
			meta: [
				{ title: documentTitle },
				{ property: "og:title", content: documentTitle },
				{ name: "twitter:title", content: documentTitle },
				...(options.description
					? [
							{ name: "description" as const, content: options.description },
							{
								property: "og:description" as const,
								content: options.description,
							},
							{
								name: "twitter:description" as const,
								content: options.description,
							},
						]
					: []),
			],
		}),
	};
}

export const defaultAppDescription =
	"AFI is a self-hostable AI gateway with a control plane for identity, policies, and snapshots.";
