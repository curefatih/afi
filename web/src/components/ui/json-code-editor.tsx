import { indentWithTab } from "@codemirror/commands";
import { json } from "@codemirror/lang-json";
import { EditorView, keymap } from "@codemirror/view";
import { vscodeDark, vscodeLight } from "@uiw/codemirror-theme-vscode";
import CodeMirror, { type ReactCodeMirrorProps } from "@uiw/react-codemirror";
import { useEffect, useMemo, useState } from "react";
import { cn } from "@/lib/utils";

function useResolvedDark() {
	const [dark, setDark] = useState(() =>
		typeof document !== "undefined"
			? document.documentElement.classList.contains("dark")
			: false,
	);

	useEffect(() => {
		const root = document.documentElement;
		const sync = () => setDark(root.classList.contains("dark"));
		sync();
		const observer = new MutationObserver(sync);
		observer.observe(root, { attributes: true, attributeFilter: ["class"] });
		return () => observer.disconnect();
	}, []);

	return dark;
}

type JsonCodeEditorProps = {
	id?: string;
	value: string;
	onChange: (value: string) => void;
	className?: string;
	minHeight?: string;
	invalid?: boolean;
	placeholder?: string;
	readOnly?: boolean;
};

export function JsonCodeEditor({
	id,
	value,
	onChange,
	className,
	minHeight = "12rem",
	invalid = false,
	placeholder,
	readOnly = false,
}: JsonCodeEditorProps) {
	const dark = useResolvedDark();

	const extensions = useMemo(
		() => [
			json(),
			keymap.of([indentWithTab]),
			EditorView.lineWrapping,
			EditorView.theme({
				"&": {
					fontSize: "12px",
				},
				".cm-content": {
					fontFamily:
						'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
					padding: "0.625rem 0",
				},
				".cm-gutters": {
					fontSize: "11px",
				},
				"&.cm-focused": {
					outline: "none",
				},
			}),
		],
		[],
	);

	const handleChange: ReactCodeMirrorProps["onChange"] = (next) => {
		onChange(next);
	};

	return (
		<div
			id={id}
			data-slot="json-code-editor"
			aria-invalid={invalid || undefined}
			className={cn(
				"overflow-hidden rounded-lg border border-input bg-transparent transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 dark:bg-input/30",
				invalid &&
					"border-destructive ring-3 ring-destructive/20 dark:border-destructive/50 dark:ring-destructive/40",
				className,
			)}
		>
			<CodeMirror
				value={value}
				height="auto"
				minHeight={minHeight}
				theme={dark ? vscodeDark : vscodeLight}
				extensions={extensions}
				onChange={handleChange}
				editable={!readOnly}
				basicSetup={{
					lineNumbers: true,
					foldGutter: true,
					highlightActiveLine: true,
					highlightActiveLineGutter: true,
					bracketMatching: true,
					closeBrackets: true,
					autocompletion: true,
					indentOnInput: true,
					tabSize: 2,
				}}
				placeholder={placeholder}
				className="text-xs [&_.cm-editor]:bg-transparent [&_.cm-gutters]:border-r-border [&_.cm-gutters]:bg-muted/40 [&_.cm-scroller]:overflow-auto"
			/>
		</div>
	);
}
