#!/usr/bin/env bash
# Generate README + web favicon/PWA + MkDocs assets from assets/brand SVGs.
# Requires: rsvg-convert (librsvg), magick (ImageMagick)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BRAND="$ROOT/assets/brand"
WEB_PUBLIC="$ROOT/web/public"
DOCS_BRAND="$ROOT/docs/assets/brand"
MARK="$BRAND/logo-mark.svg"
LOGO="$BRAND/logo.svg"

need() {
  command -v "$1" >/dev/null || {
    echo "$1 not found. Install with: $2" >&2
    exit 1
  }
}

need rsvg-convert "brew install librsvg"
need magick "brew install imagemagick"

mkdir -p "$BRAND" "$WEB_PUBLIC" "$DOCS_BRAND"

# README / docs wordmark (viewBox 500x240)
rsvg-convert -w 250 -h 120 "$LOGO" -o "$BRAND/logo.png"
rsvg-convert -w 500 -h 240 "$LOGO" -o "$BRAND/logo@2x.png"

# Square mark variants
rsvg-convert -w 512 -h 512 "$MARK" -o "$BRAND/logo-mark.png"
rsvg-convert -w 32 -h 32 "$MARK" -o "$BRAND/favicon-32.png"
rsvg-convert -w 16 -h 16 "$MARK" -o "$BRAND/favicon-16.png"
rsvg-convert -w 180 -h 180 "$MARK" -o "$BRAND/apple-touch-icon.png"
rsvg-convert -w 192 -h 192 "$MARK" -o "$BRAND/logo192.png"
rsvg-convert -w 512 -h 512 "$MARK" -o "$BRAND/logo512.png"

# Multi-resolution favicon.ico
magick "$BRAND/logo-mark.png" -define icon:auto-resize=64,48,32,16 "$BRAND/favicon.ico"

# Copy into web public
cp "$BRAND/favicon.ico" "$WEB_PUBLIC/favicon.ico"
cp "$BRAND/favicon-16.png" "$WEB_PUBLIC/favicon-16.png"
cp "$BRAND/favicon-32.png" "$WEB_PUBLIC/favicon-32.png"
cp "$BRAND/apple-touch-icon.png" "$WEB_PUBLIC/apple-touch-icon.png"
cp "$BRAND/logo192.png" "$WEB_PUBLIC/logo192.png"
cp "$BRAND/logo512.png" "$WEB_PUBLIC/logo512.png"
cp "$BRAND/logo-mark.svg" "$WEB_PUBLIC/logo.svg"

# Copy into MkDocs docs (paths referenced from mkdocs.yml)
cp "$BRAND/logo-mark.svg" "$DOCS_BRAND/logo-mark.svg"
cp "$BRAND/logo.svg" "$DOCS_BRAND/logo.svg"
cp "$BRAND/logo.png" "$DOCS_BRAND/logo.png"
cp "$BRAND/favicon.ico" "$DOCS_BRAND/favicon.ico"
cp "$BRAND/favicon-32.png" "$DOCS_BRAND/favicon-32.png"

echo "Brand assets generated in $BRAND and copied to $WEB_PUBLIC and $DOCS_BRAND"
