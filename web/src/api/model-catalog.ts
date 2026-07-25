import { queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type CatalogModel = {
	provider_type: string;
	id: string;
	mode: string;
};

export const modelCatalogQueryOptions = (providerType?: string) =>
	queryOptions({
		queryKey: ["model-catalog", providerType ?? ""],
		queryFn: () => {
			const q = providerType
				? `?provider_type=${encodeURIComponent(providerType)}`
				: "";
			return apiFetch<CatalogModel[]>(`/api/v1/platform/model-catalog${q}`);
		},
		staleTime: 60_000,
	});
