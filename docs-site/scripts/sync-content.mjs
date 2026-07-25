import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..');
const srcDocs = path.resolve(root, '../docs');
const destDocs = path.resolve(root, 'src/content/docs');
const destPublic = path.resolve(root, 'public');
const destAssets = path.resolve(root, 'src/assets');

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
 * @param {string} content
 * @param {string} relPath
 */
function transform(content, relPath) {
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

	// Starlight routes omit the .md extension.
	body = body.replace(/\]\(([^)\s]+)\.md(#[^)]*)?\)/g, ']($1$2)');

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
	for (const rel of files) {
		const src = path.join(srcDocs, rel);
		const dest = path.join(destDocs, rel);
		fs.mkdirSync(path.dirname(dest), { recursive: true });
		const transformed = transform(fs.readFileSync(src, 'utf8'), rel);
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
