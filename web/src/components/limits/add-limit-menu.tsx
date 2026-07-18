import { Button } from "#/components/ui/button";

import { Plus } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

export function AddLimitMenu() {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger>
        <Button variant="outline">
          <Plus className="mr-2 h-4 w-4" />
          Add Limit
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent>
        <DropdownMenuItem>Request Count</DropdownMenuItem>

        <DropdownMenuItem>Input Tokens</DropdownMenuItem>

        <DropdownMenuItem>Output Tokens</DropdownMenuItem>

        <DropdownMenuItem>Total Tokens</DropdownMenuItem>

        <DropdownMenuItem>Spend</DropdownMenuItem>

        <DropdownMenuItem>Concurrency</DropdownMenuItem>

        <DropdownMenuItem>Images</DropdownMenuItem>

        <DropdownMenuItem>Audio Seconds</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
