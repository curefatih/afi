import { useReducedMotion } from "motion/react";
import { type RefObject, useEffect, useId, useRef } from "react";

type MagicalBowlProps = {
	audioRef: RefObject<HTMLAudioElement | null>;
	/** True while speech audio is available to play. */
	ready: boolean;
	/** True while the gateway is generating speech. */
	busy?: boolean;
	className?: string;
};

type AudioGraph = {
	ctx: AudioContext;
	analyser: AnalyserNode;
	source: MediaElementAudioSourceNode;
	data: Uint8Array<ArrayBuffer>;
};

const POINT_COUNT = 28;
const IDLE_LEVEL = 0.08;

function clamp(n: number, min: number, max: number) {
	return Math.min(max, Math.max(min, n));
}

function lerp(a: number, b: number, t: number) {
	return a + (b - a) * t;
}

/** Open bowl silhouette: wide rim on top, rounded vessel below. */
function bowlPoint(
	i: number,
	count: number,
	cx: number,
	cy: number,
	rx: number,
	ry: number,
	openness: number,
): { x: number; y: number } {
	const t = i / count;
	// Map 0..1 around the vessel; leave a soft gap at the top rim
	const angle = -Math.PI * 0.12 + t * Math.PI * 1.24;
	const rimPull = Math.cos(angle) ** 2;
	const localRx = rx * (1 + rimPull * openness * 0.18);
	const localRy = ry * (1 - rimPull * 0.08);
	return {
		x: cx + Math.cos(angle) * localRx,
		y: cy + Math.sin(angle) * localRy + rimPull * 6,
	};
}

function catmullRomPath(
	pts: Array<{ x: number; y: number }>,
	closed: boolean,
): string {
	if (pts.length < 2) return "";
	const n = pts.length;
	let d = `M ${pts[0].x.toFixed(2)} ${pts[0].y.toFixed(2)}`;
	const count = closed ? n : n - 1;
	for (let i = 0; i < count; i++) {
		const p0 = pts[closed ? (i - 1 + n) % n : Math.max(0, i - 1)];
		const p1 = pts[i];
		const p2 = pts[closed ? (i + 1) % n : Math.min(n - 1, i + 1)];
		const p3 = pts[closed ? (i + 2) % n : Math.min(n - 1, i + 2)];
		const c1x = p1.x + (p2.x - p0.x) / 6;
		const c1y = p1.y + (p2.y - p0.y) / 6;
		const c2x = p2.x - (p3.x - p1.x) / 6;
		const c2y = p2.y - (p3.y - p1.y) / 6;
		d += ` C ${c1x.toFixed(2)} ${c1y.toFixed(2)}, ${c2x.toFixed(2)} ${c2y.toFixed(2)}, ${p2.x.toFixed(2)} ${p2.y.toFixed(2)}`;
	}
	if (closed) d += " Z";
	return d;
}

function liquidPath(
	cx: number,
	cy: number,
	rx: number,
	ry: number,
	level: number,
	bands: Float32Array,
	time: number,
): string {
	const surfaceY = cy - ry * 0.35 + (1 - level) * ry * 0.9;
	const left = cx - rx * 0.82;
	const right = cx + rx * 0.82;
	const steps = 18;
	const pts: Array<{ x: number; y: number }> = [];
	for (let i = 0; i <= steps; i++) {
		const u = i / steps;
		const x = lerp(left, right, u);
		const wave =
			Math.sin(u * Math.PI * 3 + time * 2.4) * (4 + level * 10) +
			Math.sin(u * Math.PI * 7 - time * 3.1) * (2 + level * 6) +
			(bands[i % bands.length] ?? 0) * 22;
		pts.push({ x, y: surfaceY + wave * (0.35 + level) });
	}
	// Drop down the vessel walls to close the fill
	pts.push({ x: right, y: cy + ry * 0.72 });
	pts.push({ x: cx, y: cy + ry * 0.92 });
	pts.push({ x: left, y: cy + ry * 0.72 });
	return catmullRomPath(pts, true);
}

/**
 * Magical answering bowl whose silhouette and liquid surface
 * react to live audio levels from the linked HTMLAudioElement.
 */
export function MagicalBowl({
	audioRef,
	ready,
	busy = false,
	className,
}: MagicalBowlProps) {
	const reduceMotion = useReducedMotion();
	const uid = useId();
	const svgRef = useRef<SVGSVGElement>(null);
	const vesselRef = useRef<SVGPathElement>(null);
	const rimRef = useRef<SVGPathElement>(null);
	const liquidRef = useRef<SVGPathElement>(null);
	const glowRef = useRef<SVGEllipseElement>(null);
	const auraRef = useRef<SVGEllipseElement>(null);
	const sparkRefs = useRef<Array<SVGCircleElement | null>>([]);
	const graphRef = useRef<AudioGraph | null>(null);
	const levelsRef = useRef({
		amp: 0,
		bass: 0,
		mid: 0,
		treble: 0,
		playing: false,
	});
	const bandsRef = useRef(new Float32Array(POINT_COUNT));
	const rafRef = useRef(0);
	const busyRef = useRef(busy);
	const readyRef = useRef(ready);
	const reduceMotionRef = useRef(reduceMotion);
	busyRef.current = busy;
	readyRef.current = ready;
	reduceMotionRef.current = reduceMotion;

	// MediaElementSource can only be created once per <audio> element.
	useEffect(() => {
		const el = audioRef.current;
		if (!el) return;

		let cancelled = false;

		const ensureGraph = (): AudioGraph | null => {
			const existing = graphRef.current;
			if (existing?.source.mediaElement === el) return existing;

			try {
				const Ctx =
					window.AudioContext ||
					(window as unknown as { webkitAudioContext: typeof AudioContext })
						.webkitAudioContext;
				const ctx = new Ctx();
				const analyser = ctx.createAnalyser();
				analyser.fftSize = 256;
				analyser.smoothingTimeConstant = 0.78;
				const source = ctx.createMediaElementSource(el);
				source.connect(analyser);
				analyser.connect(ctx.destination);
				const graph: AudioGraph = {
					ctx,
					analyser,
					source,
					data: new Uint8Array(
						analyser.frequencyBinCount,
					) as Uint8Array<ArrayBuffer>,
				};
				graphRef.current = graph;
				return graph;
			} catch {
				return graphRef.current;
			}
		};

		const onPlay = () => {
			if (cancelled) return;
			const g = ensureGraph();
			if (!g) return;
			void g.ctx.resume().then(() => {
				if (!cancelled) levelsRef.current.playing = true;
			});
		};
		const onPause = () => {
			levelsRef.current.playing = false;
		};
		const onEnded = () => {
			levelsRef.current.playing = false;
		};

		el.addEventListener("play", onPlay);
		el.addEventListener("pause", onPause);
		el.addEventListener("ended", onEnded);
		if (!el.paused && !el.ended) onPlay();

		return () => {
			cancelled = true;
			el.removeEventListener("play", onPlay);
			el.removeEventListener("pause", onPause);
			el.removeEventListener("ended", onEnded);
		};
	}, [audioRef]);

	useEffect(() => {
		return () => {
			cancelAnimationFrame(rafRef.current);
			const g = graphRef.current;
			graphRef.current = null;
			if (g) void g.ctx.close();
		};
	}, []);

	useEffect(() => {
		const tick = (now: number) => {
			const time = now / 1000;
			const g = graphRef.current;
			const levels = levelsRef.current;
			const bands = bandsRef.current;
			const isBusy = busyRef.current;
			const isReady = readyRef.current;
			const preferReduced = !!reduceMotionRef.current;

			if (g && levels.playing && !preferReduced) {
				g.analyser.getByteFrequencyData(g.data);
				const data = g.data;
				const n = data.length;
				let sum = 0;
				let bass = 0;
				let mid = 0;
				let treble = 0;
				const bassEnd = Math.floor(n * 0.12);
				const midEnd = Math.floor(n * 0.45);
				for (let i = 0; i < n; i++) {
					const v = data[i] / 255;
					sum += v;
					if (i < bassEnd) bass += v;
					else if (i < midEnd) mid += v;
					else treble += v;
				}
				levels.amp = lerp(levels.amp, sum / n, 0.35);
				levels.bass = lerp(levels.bass, bass / Math.max(1, bassEnd), 0.3);
				levels.mid = lerp(levels.mid, mid / Math.max(1, midEnd - bassEnd), 0.3);
				levels.treble = lerp(
					levels.treble,
					treble / Math.max(1, n - midEnd),
					0.3,
				);
				for (let i = 0; i < POINT_COUNT; i++) {
					const idx = Math.floor((i / POINT_COUNT) * n);
					bands[i] = lerp(bands[i], data[idx] / 255, 0.4);
				}
			} else {
				const breathe = preferReduced
					? IDLE_LEVEL
					: IDLE_LEVEL +
						Math.sin(time * (isReady ? 1.6 : 1.2)) * (isReady ? 0.05 : 0.035);
				const swirl =
					isBusy && !preferReduced ? 0.22 + Math.sin(time * 4) * 0.08 : 0;
				const awaitPulse =
					isReady && !isBusy && !preferReduced
						? 0.04 + Math.sin(time * 2.2) * 0.025
						: 0;
				const target = breathe + swirl + awaitPulse;
				levels.amp = lerp(levels.amp, target, 0.08);
				levels.bass = lerp(levels.bass, target * 0.9, 0.08);
				levels.mid = lerp(levels.mid, target * 0.7, 0.08);
				levels.treble = lerp(levels.treble, target * 0.5, 0.08);
				for (let i = 0; i < POINT_COUNT; i++) {
					const w =
						target *
						(0.6 +
							0.4 *
								Math.sin(
									time * (isBusy ? 3.2 : isReady ? 1.5 : 1.1) +
										i * 0.45 +
										(isBusy ? 1 : 0),
								));
					bands[i] = lerp(bands[i], w, 0.1);
				}
			}

			const amp = levels.amp;
			const energy = clamp(amp * 2.4, 0, 1.4);
			const cx = 100;
			const cy = 108;
			const baseRx = 62 + levels.mid * 18;
			const baseRy = 48 + levels.bass * 22;
			const openness = 0.55 + levels.treble * 0.8;

			const vesselPts: Array<{ x: number; y: number }> = [];
			for (let i = 0; i < POINT_COUNT; i++) {
				const base = bowlPoint(
					i,
					POINT_COUNT,
					cx,
					cy,
					baseRx,
					baseRy,
					openness,
				);
				const band = bands[i] ?? 0;
				const angle = (i / POINT_COUNT) * Math.PI * 2;
				const push = band * (14 + energy * 20);
				const wobble =
					Math.sin(time * (levels.playing ? 6 : 2) + i * 0.7) *
					(2 + energy * 5);
				vesselPts.push({
					x: base.x + Math.cos(angle) * push * 0.35 + wobble * 0.4,
					y: base.y + Math.sin(angle * 0.5 + 1) * push * 0.85 + wobble,
				});
			}

			const rimPts: Array<{ x: number; y: number }> = [];
			const rimY = cy - baseRy * 0.55;
			const rimRx = baseRx * (1.05 + openness * 0.12);
			for (let i = 0; i < 16; i++) {
				const u = i / 16;
				const a = u * Math.PI * 2;
				const rip =
					(bands[i % POINT_COUNT] ?? 0) * 10 +
					Math.sin(time * 3 + u * Math.PI * 4) * (1.5 + energy * 3);
				rimPts.push({
					x: cx + Math.cos(a) * (rimRx + rip * 0.35),
					y: rimY + Math.sin(a) * (9 + rip * 0.25) + levels.bass * 4,
				});
			}

			if (vesselRef.current) {
				vesselRef.current.setAttribute("d", catmullRomPath(vesselPts, true));
			}
			if (rimRef.current) {
				rimRef.current.setAttribute("d", catmullRomPath(rimPts, true));
			}
			if (liquidRef.current) {
				liquidRef.current.setAttribute(
					"d",
					liquidPath(
						cx,
						cy,
						baseRx * 0.88,
						baseRy * 0.88,
						clamp(0.35 + energy * 0.55, 0.2, 1),
						bands,
						time,
					),
				);
			}
			if (glowRef.current) {
				const grx = 48 + energy * 36;
				const gry = 28 + energy * 22;
				glowRef.current.setAttribute("rx", grx.toFixed(2));
				glowRef.current.setAttribute("ry", gry.toFixed(2));
				glowRef.current.setAttribute(
					"opacity",
					(0.18 + energy * 0.45 + (isBusy ? 0.12 : 0)).toFixed(3),
				);
			}
			if (auraRef.current) {
				auraRef.current.setAttribute("rx", (70 + energy * 40).toFixed(2));
				auraRef.current.setAttribute("ry", (42 + energy * 28).toFixed(2));
				auraRef.current.setAttribute(
					"opacity",
					(0.08 + energy * 0.28).toFixed(3),
				);
			}

			for (let i = 0; i < sparkRefs.current.length; i++) {
				const spark = sparkRefs.current[i];
				if (!spark) continue;
				const orbit = time * (0.6 + i * 0.12) + i;
				const radius = 54 + (i % 3) * 10 + energy * 18;
				const sx = cx + Math.cos(orbit) * radius * (0.7 + (i % 2) * 0.25);
				const sy =
					cy - 8 + Math.sin(orbit * 1.3) * radius * 0.35 - levels.treble * 20;
				const size = 1.2 + (bands[i % POINT_COUNT] ?? 0) * 3.5 + energy;
				spark.setAttribute("cx", sx.toFixed(2));
				spark.setAttribute("cy", sy.toFixed(2));
				spark.setAttribute("r", size.toFixed(2));
				spark.setAttribute(
					"opacity",
					(levels.playing || isBusy
						? 0.25 + (bands[i % POINT_COUNT] ?? 0) * 0.7
						: 0.08 + Math.sin(time + i) * 0.04
					).toFixed(3),
				);
			}

			if (svgRef.current) {
				svgRef.current.style.transform = `translateY(${(-energy * 4).toFixed(2)}px) scale(${(1 + energy * 0.04).toFixed(4)})`;
			}

			rafRef.current = requestAnimationFrame(tick);
		};

		rafRef.current = requestAnimationFrame(tick);
		return () => cancelAnimationFrame(rafRef.current);
	}, []);

	const gradId = `${uid}-bowl`;
	const liquidId = `${uid}-liquid`;
	const glowId = `${uid}-glow`;

	return (
		<div className={className} aria-hidden>
			<svg
				ref={svgRef}
				viewBox="0 0 200 200"
				className="mx-auto h-52 w-full max-w-[240px] overflow-visible transition-transform duration-100 will-change-transform"
			>
				<title>Magical answering bowl</title>
				<defs>
					<radialGradient id={glowId} cx="50%" cy="55%" r="55%">
						<stop
							offset="0%"
							stopColor="currentColor"
							className="text-foreground"
							stopOpacity="0.55"
						/>
						<stop
							offset="55%"
							stopColor="currentColor"
							className="text-muted-foreground"
							stopOpacity="0.18"
						/>
						<stop offset="100%" stopColor="currentColor" stopOpacity="0" />
					</radialGradient>
					<linearGradient id={gradId} x1="0%" y1="0%" x2="100%" y2="100%">
						<stop
							offset="0%"
							stopColor="currentColor"
							className="text-foreground"
							stopOpacity="0.14"
						/>
						<stop
							offset="45%"
							stopColor="currentColor"
							className="text-muted-foreground"
							stopOpacity="0.28"
						/>
						<stop
							offset="100%"
							stopColor="currentColor"
							className="text-foreground"
							stopOpacity="0.5"
						/>
					</linearGradient>
					<linearGradient id={liquidId} x1="0%" y1="0%" x2="0%" y2="100%">
						<stop
							offset="0%"
							stopColor="currentColor"
							className="text-foreground"
							stopOpacity="0.55"
						/>
						<stop
							offset="55%"
							stopColor="currentColor"
							className="text-muted-foreground"
							stopOpacity="0.32"
						/>
						<stop
							offset="100%"
							stopColor="currentColor"
							className="text-foreground"
							stopOpacity="0.12"
						/>
					</linearGradient>
					<filter
						id={`${uid}-soft`}
						x="-30%"
						y="-30%"
						width="160%"
						height="160%"
					>
						<feGaussianBlur stdDeviation="3.5" />
					</filter>
				</defs>

				{/* Floor shadow */}
				<ellipse
					cx="100"
					cy="168"
					rx="46"
					ry="8"
					className="fill-foreground/10"
				/>

				{/* Outer aura */}
				<ellipse
					ref={auraRef}
					cx="100"
					cy="108"
					rx="70"
					ry="42"
					fill={`url(#${glowId})`}
					opacity="0.1"
					filter={`url(#${uid}-soft)`}
				/>

				{/* Inner glow */}
				<ellipse
					ref={glowRef}
					cx="100"
					cy="118"
					rx="48"
					ry="28"
					fill={`url(#${glowId})`}
					opacity="0.2"
				/>

				{/* Vessel body */}
				<path
					ref={vesselRef}
					fill={`url(#${gradId})`}
					className="stroke-foreground/25"
					strokeWidth="1.25"
					d="M 40 90 C 40 150, 160 150, 160 90 C 150 70, 50 70, 40 90 Z"
				/>

				{/* Liquid */}
				<path
					ref={liquidRef}
					fill={`url(#${liquidId})`}
					className="stroke-foreground/15"
					strokeWidth="0.75"
					d="M 50 110 C 70 95, 130 95, 150 110 C 145 145, 55 145, 50 110 Z"
				/>

				{/* Rim */}
				<path
					ref={rimRef}
					fill="none"
					className="stroke-foreground/45"
					strokeWidth="2"
					strokeLinecap="round"
					d="M 38 88 C 60 72, 140 72, 162 88 C 140 98, 60 98, 38 88 Z"
				/>

				{/* Specular highlight on rim */}
				<path
					d="M 62 80 C 85 72, 115 72, 138 80"
					fill="none"
					className="stroke-foreground/35"
					strokeWidth="1.5"
					strokeLinecap="round"
				/>

				{/* Orbiting sparks */}
				{[0, 1, 2, 3, 4, 5, 6].map((i) => (
					<circle
						key={`spark-${i}`}
						ref={(node) => {
							sparkRefs.current[i] = node;
						}}
						cx="100"
						cy="100"
						r="1.5"
						className="fill-foreground"
						opacity="0.15"
					/>
				))}
			</svg>
		</div>
	);
}
