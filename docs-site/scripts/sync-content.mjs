import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..');
const srcDocs = path.resolve(root, '../docs');
const destDocs = path.resolve(root, 'src/content/docs');
const destPublic = path.resolve(root, 'public');
const destAssets = path.resolve(root, 'src/assets');
const repoRoot = path.resolve(root, '..');

const GITHUB_BLOB = 'https://github.com/curefatih/afi/blob/main';
const GITHUB_TREE = 'https://github.com/curefatih/afi/tree/main';

const SKIP_NAMES = new Set(['llms.txt']);
const SKIP_DIRS = new Set(['assets']);

function walk(dir, base = dir) {
	/** @type {string[]} */
	const files = [];
	for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
		if (entry.name.startsWith('.')) continue;
		const abs = path.join(dir, entry.name);
		if (entry.isDirectory()) {
			if (SKIP_DIRS.has(entry.name)) continue;
			files.push(...walk(abs, base));
			continue;
		}
		if (SKIP_NAMES.has(entry.name)) continue;
		if (!entry.name.endsWith('.md')) continue;
		files.push(path.relative(base, abs));
	}
	return files;
}

function extractTitle(content, fallback) {
	const match = content.match(/^#\s+(.+)$/m);
	if (!match) return fallback;
	return match[1].replace(/[*_`]/g, '').trim();
}

function yamlQuote(value) {
	return JSON.stringify(value);
}

/**
 * Map a docs-relative markdown path (posix, may include .md / index.md) to a site URL.
 * @param {string} docsRel
 */
function docsPathToUrl(docsRel) {
	let clean = docsRel.replace(/\\/g, '/');
	if (clean.endsWith('.md')) clean = clean.slice(0, -3);
	if (clean.endsWith('/index')) clean = clean.slice(0, -'/index'.length);
	if (clean === 'index' || clean === '') return '/';
	return `/${clean}/`;
}

/**
 * @param {string} href
 * @param {string} fromRel  e.g. getting-started/local-dev.md
 * @param {Set<string>} docFiles
 */
function rewriteHref(href, fromRel, docFiles) {
	const trimmed = href.trim();
	if (!trimmed) return href;
	if (/^(https?:|mailto:|tel:|#)/i.test(trimmed)) return href;
	if (trimmed.startsWith('/assets/')) return href;

	const hashIndex = trimmed.indexOf('#');
	const hash = hashIndex >= 0 ? trimmed.slice(hashIndex) : '';
	const pathPart = hashIndex >= 0 ? trimmed.slice(0, hashIndex) : trimmed;

	if (!pathPart) return href; // pure hash

	// Already site-root absolute docs path.
	if (pathPart.startsWith('/') && !pathPart.startsWith('/assets/')) {
		const noSlash = pathPart.replace(/\/+$/, '') || '';
		const candidate = noSlash.replace(/^\//, '');
		if (
			docFiles.has(`${candidate}.md`) ||
			docFiles.has(`${candidate}/index.md`) ||
			candidate === ''
		) {
			const url = candidate === '' ? '/' : `/${candidate}/`;
			return `${url}${hash}`;
		}
		return href;
	}

	const fromDir = path.posix.dirname(fromRel.replace(/\\/g, '/'));

	// Resolve against the docs tree, then see if it escapes into the repo.
	const absFromDocs = path.resolve(srcDocs, fromDir === '.' ? '.' : fromDir, pathPart);
	const relFromDocs = path.relative(srcDocs, absFromDocs).replace(/\\/g, '/');

	if (relFromDocs.startsWith('..')) {
		const absFromRepo = path.resolve(path.dirname(path.join(srcDocs, fromRel)), pathPart);
		const relFromRepo = path.relative(repoRoot, absFromRepo).replace(/\\/g, '/');
		if (relFromRepo.startsWith('..')) return href;

		const isDir =
			fs.existsSync(absFromRepo) && fs.statSync(absFromRepo).isDirectory();
		return `${isDir ? GITHUB_TREE : GITHUB_BLOB}/${relFromRepo}${hash}`;
	}

	// Prefer exact .md, then index.md under a directory name.
	let docsRel = relFromDocs;
	if (docsRel.endsWith('/')) docsRel = docsRel.slice(0, -1);

	if (docFiles.has(docsRel)) {
		return `${docsPathToUrl(docsRel)}${hash}`;
	}
	if (docFiles.has(`${docsRel}.md`)) {
		return `${docsPathToUrl(`${docsRel}.md`)}${hash}`;
	}
	if (docFiles.has(`${docsRel}/index.md`)) {
		return `${docsPathToUrl(`${docsRel}/index.md`)}${hash}`;
	}

	// Fallback: treat as docs page path without verifying.
	if (docsRel.endsWith('.md') || !docsRel.includes('.')) {
		return `${docsPathToUrl(docsRel.endsWith('.md') ? docsRel : `${docsRel}.md`)}${hash}`;
	}

	return href;
}

/**
 * @param {string} content
 * @param {string} relPath
 * @param {Set<string>} docFiles
 */
function transform(content, relPath, docFiles) {
	let body = content.replace(/^\uFEFF/, '');

	if (body.startsWith('---\n')) {
		const end = body.indexOf('\n---\n', 4);
		if (end !== -1) {
			body = body.slice(end + 5);
		}
	}

	const fallback = path.basename(relPath, '.md');
	const title = extractTitle(body, fallback);

	// Starlight renders the page title from frontmatter — drop the leading H1.
	body = body.replace(/^#\s+.+\n+/, '');

	// MkDocs serves docs/assets at /assets; Starlight uses public/assets.
	body = body.replace(/\]\((?:\.\.\/)*(?:\.\/)?assets\//g, '](/assets/');
	body = body.replace(/(src|href)="(?:\.\.\/)*(?:\.\/)?assets\//g, '$1="/assets/');

	// Rewrite markdown links to absolute docs URLs or GitHub repo URLs.
	body = body.replace(/\]\(([^)\s]+)\)/g, (full, href) => {
		const next = rewriteHref(href, relPath.replace(/\\/g, '/'), docFiles);
		return `](${next})`;
	});

	const description =
		relPath === 'index.md'
			? 'AFI is a self-hostable, cloud-native LLM gateway.'
			: undefined;

	const lines = ['---', `title: ${yamlQuote(title)}`];
	if (description) lines.push(`description: ${yamlQuote(description)}`);
	lines.push('---', '', body.trim(), '');
	return lines.join('\n');
}

function copyDir(from, to) {
	fs.mkdirSync(to, { recursive: true });
	for (const entry of fs.readdirSync(from, { withFileTypes: true })) {
		const src = path.join(from, entry.name);
		const dest = path.join(to, entry.name);
		if (entry.isDirectory()) {
			copyDir(src, dest);
		} else {
			fs.copyFileSync(src, dest);
		}
	}
}

function writeNotFound() {
	const notFound = [
		'---',
		'title: "Page not found"',
		'template: splash',
		'editUrl: false',
		'---',
		'',
		'This page does not exist. Check the sidebar or go [home](/).',
		'',
	].join('\n');
	fs.writeFileSync(path.join(destDocs, '404.md'), notFound);
	console.log('synced 404.md');
}

function main() {
	if (!fs.existsSync(srcDocs)) {
		console.error(`Source docs not found: ${srcDocs}`);
		process.exit(1);
	}

	fs.rmSync(destDocs, { recursive: true, force: true });
	fs.mkdirSync(destDocs, { recursive: true });

	const files = walk(srcDocs);
	const docFiles = new Set(files.map((f) => f.replace(/\\/g, '/')));

	for (const rel of files) {
		const src = path.join(srcDocs, rel);
		const dest = path.join(destDocs, rel);
		fs.mkdirSync(path.dirname(dest), { recursive: true });
		const transformed = transform(fs.readFileSync(src, 'utf8'), rel, docFiles);
		fs.writeFileSync(dest, transformed);
		console.log(`synced ${rel}`);
	}

	writeNotFound();

	const brandSrc = path.join(srcDocs, 'assets');
	if (fs.existsSync(brandSrc)) {
		const publicAssets = path.join(destPublic, 'assets');
		fs.rmSync(publicAssets, { recursive: true, force: true });
		copyDir(brandSrc, publicAssets);
		console.log('synced public/assets');
	}

	const llms = path.join(srcDocs, 'llms.txt');
	if (fs.existsSync(llms)) {
		fs.copyFileSync(llms, path.join(destPublic, 'llms.txt'));
		console.log('synced public/llms.txt');
	}

	const logoMark = path.join(srcDocs, 'assets/brand/logo-mark.svg');
	if (fs.existsSync(logoMark)) {
		fs.mkdirSync(destAssets, { recursive: true });
		fs.copyFileSync(logoMark, path.join(destAssets, 'logo-mark.svg'));
		console.log('synced src/assets/logo-mark.svg');
	}

	const favicon = path.join(srcDocs, 'assets/brand/favicon.ico');
	if (fs.existsSync(favicon)) {
		fs.copyFileSync(favicon, path.join(destPublic, 'favicon.ico'));
	}

	console.log(`Done. ${files.length} pages synced.`);
}

main();
