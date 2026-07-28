export { PlatformClient, PlatformApiError } from "./client.js";
export type { PlatformClientOptions } from "./client.js";
export type { paths, components } from "./schema.gen.js";
export {
	SIGNATURE_NAME,
	REQUIRED_COMPONENTS,
	signHeaders,
	mergeSignedHeaders,
	contentDigestSha256,
	pathAndQuery,
	generateEd25519Pem,
} from "./httpsign.js";
export type { SignHeadersInput } from "./httpsign.js";
