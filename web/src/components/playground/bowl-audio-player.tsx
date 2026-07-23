import { PauseIcon, PlayIcon, RotateCcwIcon } from "lucide-react";
import { type RefObject, useEffect, useRef, useState } from "react";
import { Button } from "#/components/ui/button";
import { Slider } from "#/components/ui/slider";
import { cn } from "@/lib/utils";

type BowlAudioPlayerProps = {
	audioRef: RefObject<HTMLAudioElement | null>;
	src: string | null;
	busy?: boolean;
	className?: string;
};

function formatTime(seconds: number): string {
	if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
	const m = Math.floor(seconds / 60);
	const s = Math.floor(seconds % 60);
	return `${m}:${s.toString().padStart(2, "0")}`;
}

function sliderValue(value: number | readonly number[], fallback: number) {
	if (typeof value === "number") return value;
	if (Array.isArray(value) && typeof value[0] === "number") return value[0];
	return fallback;
}

export function BowlAudioPlayer({
	audioRef,
	src,
	busy = false,
	className,
}: BowlAudioPlayerProps) {
	const [playing, setPlaying] = useState(false);
	const [current, setCurrent] = useState(0);
	const [duration, setDuration] = useState(0);
	const seekingRef = useRef(false);
	const srcRef = useRef(src);

	if (srcRef.current !== src) {
		srcRef.current = src;
		setCurrent(0);
		setDuration(0);
		setPlaying(false);
		seekingRef.current = false;
	}

	useEffect(() => {
		const el = audioRef.current;
		if (!el) return;

		const onTime = () => {
			if (!seekingRef.current) setCurrent(el.currentTime || 0);
		};
		const onMeta = () => {
			const d = el.duration;
			setDuration(Number.isFinite(d) ? d : 0);
		};
		const onPlay = () => setPlaying(true);
		const onPause = () => setPlaying(false);
		const onEnded = () => {
			setPlaying(false);
			setCurrent(el.duration || 0);
		};

		el.addEventListener("timeupdate", onTime);
		el.addEventListener("loadedmetadata", onMeta);
		el.addEventListener("durationchange", onMeta);
		el.addEventListener("play", onPlay);
		el.addEventListener("pause", onPause);
		el.addEventListener("ended", onEnded);

		setPlaying(!el.paused && !el.ended);
		if (!seekingRef.current) setCurrent(el.currentTime || 0);
		const d = el.duration;
		setDuration(Number.isFinite(d) ? d : 0);

		return () => {
			el.removeEventListener("timeupdate", onTime);
			el.removeEventListener("loadedmetadata", onMeta);
			el.removeEventListener("durationchange", onMeta);
			el.removeEventListener("play", onPlay);
			el.removeEventListener("pause", onPause);
			el.removeEventListener("ended", onEnded);
		};
	}, [audioRef]);

	const toggle = () => {
		const el = audioRef.current;
		if (!el || !src || busy) return;
		if (el.paused || el.ended) {
			void el.play().catch(() => {});
		} else {
			el.pause();
		}
	};

	const restart = () => {
		const el = audioRef.current;
		if (!el || !src || busy) return;
		el.currentTime = 0;
		setCurrent(0);
		void el.play().catch(() => {});
	};

	const seekTo = (next: number) => {
		const el = audioRef.current;
		if (!el || !src) return;
		const clamped = Math.min(Math.max(next, 0), duration || next);
		el.currentTime = clamped;
		setCurrent(clamped);
	};

	const disabled = !src || busy;
	const max = duration > 0 ? duration : 1;

	return (
		<div className={cn("relative space-y-3", className)}>
			<div className="flex items-center gap-2">
				<Button
					type="button"
					variant="outline"
					size="icon-lg"
					onClick={toggle}
					disabled={disabled}
					aria-label={playing ? "Pause" : "Play"}
					className="rounded-full"
				>
					{playing ? <PauseIcon /> : <PlayIcon />}
				</Button>
				<Button
					type="button"
					variant="ghost"
					size="icon"
					onClick={restart}
					disabled={disabled}
					aria-label="Replay"
				>
					<RotateCcwIcon />
				</Button>
				<div className="text-muted-foreground min-w-0 flex-1 text-right font-mono text-xs tabular-nums">
					<span className="text-foreground">{formatTime(current)}</span>
					<span className="mx-1 opacity-50">/</span>
					<span>{formatTime(duration)}</span>
				</div>
			</div>

			<Slider
				value={[Math.min(current, max)]}
				min={0}
				max={max}
				step={0.05}
				disabled={disabled || duration <= 0}
				onValueChange={(value) => {
					seekingRef.current = true;
					setCurrent(sliderValue(value, current));
				}}
				onValueCommitted={(value) => {
					seekTo(sliderValue(value, current));
					seekingRef.current = false;
				}}
				aria-label="Seek"
			/>
		</div>
	);
}
