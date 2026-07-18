import { Button } from "#/components/ui/button";

import { Plus } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../ui/dropdown-menu";

export function AddConditionMenu() {
  return (
    <DropdownMenu>

      <DropdownMenuTrigger>

        <Button variant="outline">

          <Plus className="mr-2 h-4 w-4"/>

          Add Condition

        </Button>

      </DropdownMenuTrigger>

      <DropdownMenuContent>

        <DropdownMenuItem>
          Provider
        </DropdownMenuItem>

        <DropdownMenuItem>
          Model
        </DropdownMenuItem>

        <DropdownMenuItem>
          Region
        </DropdownMenuItem>

        <DropdownMenuItem>
          API Key
        </DropdownMenuItem>

        <DropdownMenuItem>
          Metadata
        </DropdownMenuItem>

      </DropdownMenuContent>

    </DropdownMenu>
  )
}