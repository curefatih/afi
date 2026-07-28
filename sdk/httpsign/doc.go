package httpsign

// Package httpsign signs AFI gateway HTTP requests using RFC 9421
// (HTTP Message Signatures) and RFC 9530 Content-Digest.
//
// Prefer Signature label "sig1" and cover @method, @path, @query, and
// content-digest — the gateway's required component set.
