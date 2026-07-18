import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "#/components/ui/select";
import { Trash2 } from "lucide-react";
import { Label } from "../ui/label";

type Props = {
    title: string;
    onDelete(): void;
};

export function LimitRuleCard({
    title,
    onDelete,
}: Props) {
    return (
        <Card>

            <CardContent className="space-y-6 p-6">

                <div className="flex items-center justify-between">

                    <h4 className="font-medium">
                        {title}
                    </h4>

                    <Button
                        size="icon"
                        variant="ghost"
                        onClick={onDelete}
                    >
                        <Trash2 className="h-4 w-4"/>
                    </Button>

                </div>

                <div className="grid grid-cols-3 gap-4">

                    <div className="space-y-2">
                        <Label>Maximum</Label>

                        <Input
                            type="number"
                            placeholder="100"
                        />
                    </div>

                    <div className="space-y-2">

                        <Label>Per</Label>

                        <Select defaultValue="minute">

                            <SelectTrigger>
                                <SelectValue/>
                            </SelectTrigger>

                            <SelectContent>

                                <SelectItem value="second">
                                    Second
                                </SelectItem>

                                <SelectItem value="minute">
                                    Minute
                                </SelectItem>

                                <SelectItem value="hour">
                                    Hour
                                </SelectItem>

                                <SelectItem value="day">
                                    Day
                                </SelectItem>

                                <SelectItem value="month">
                                    Month
                                </SelectItem>

                            </SelectContent>

                        </Select>

                    </div>

                    <div className="space-y-2">

                        <Label>Burst</Label>

                        <Input
                            type="number"
                            placeholder="200"
                        />

                    </div>

                </div>

            </CardContent>

        </Card>
    )
}