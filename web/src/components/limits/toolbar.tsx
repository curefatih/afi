import { useState } from "react";

import { Button } from "@/components/ui/button";

import { PolicySheet } from "./policy-sheet";

export function LimitsToolbar() {
	const [open, setOpen] = useState(false);

	return (
		<>
			<div className="flex items-center justify-between">
				{/* search */}

				<Button onClick={() => setOpen(true)}>New Policy</Button>
			</div>

			<PolicySheet policy={null} open={open} onOpenChange={setOpen} />
		</>
	);
}
