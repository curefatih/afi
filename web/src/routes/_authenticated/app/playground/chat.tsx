import { Button } from "#/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "#/components/ui/combobox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "#/components/ui/empty";
import { Input } from "#/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
} from "#/components/ui/input-group";
import { Label } from "#/components/ui/label";
import { MessageAnimated } from "#/components/ui/message-animated";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "#/components/ui/message-scroller";
import { Separator } from "#/components/ui/separator";
import { Slider } from "#/components/ui/slider";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "#/components/ui/tooltip";
import {
  createFileRoute,
  useNavigate,
  useSearch,
} from "@tanstack/react-router";
import {
  ArrowUpIcon,
  GlobeIcon,
  ImageIcon,
  MessageCircleDashedIcon,
  PaperclipIcon,
  PlusIcon,
  RotateCwIcon,
  TelescopeIcon,
} from "lucide-react";
import { useState } from "react";
import z from "zod";

const chatThreadSearchSchema = z.object({
  thread: z.string().catch(""),
});

export const Route = createFileRoute("/_authenticated/app/playground/chat")({
  staticData: {
    getTitle: () => "Playground",
  },
  component: RouteComponent,
  validateSearch: chatThreadSearchSchema,
});

const initialMessages = [
  {
    id: "1",
    role: "assistant",
    content: [
      {
        type: "text",
        text: "Hello! How can I assist you today?",
      },
    ],
  },
];

const models = [
  { label: "Apple", value: "apple" },
  { label: "Banana", value: "banana" },
  { label: "Blueberry", value: "blueberry" },
  { label: "Grapes", value: "grapes" },
  { label: "Pineapple", value: "pineapple" },
];

type Message = {
  id: string;
  role: "user" | "assistant";
  content: Array<{
    type: "text";
    text: string;
  }>;
};

function RouteComponent() {
  const navigate = useNavigate();
  const { thread } = useSearch({ strict: false });
  const [nextMessage, setNextMessage] = useState(null);
  const [isBusy, setIsBusy] = useState(false);
  const [messages, setMessages] = useState(initialMessages);

  const getMessageText = (message: Message) => {
    if (!message || !message.content) {
      return "";
    }

    const textParts = message.content
      .filter((part) => part.type === "text" && part.text)
      .map((part) => part.text);

    return textParts.join(" ");
  };

  return (
    <div className="flex flex-row h-full">
      <div className="chat flex-1 h-full">
        <div className="flex h-full flex-row">
          <div className="threads w-64 p-2">
            <div className="">
              <h5 className="">Threads</h5>
              <span className="font-thin text-xs">
                Conversations that you have created before
              </span>
            </div>
            <Separator className={"m-2"} />
            <Button
              variant={thread && thread == "tobedone" ? "outline" : "ghost"}
              onClick={() => {
                navigate({
                  to: `/app/playground/chat`,
                  search: {
                    thread: "tobedone",
                  },
                });
              }}
              className={"w-full text-left cursor-pointer block"}
            >
              text
            </Button>
          </div>
          <div className="flex-1 h-full w-full p-2">
            <MessageScrollerProvider>
              <div className="relative h-full w-full p-2">
                <Card className="mx-auto h-full w-full gap-0">
                  <CardHeader className="gap-1 border-b">
                    <CardTitle>New Chat</CardTitle>
                    <CardDescription>How can I help you today?</CardDescription>
                    <CardAction>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              variant="outline"
                              size="icon"
                              aria-label="Reset conversation"
                              onClick={() => setMessages(initialMessages)}
                            >
                              <RotateCwIcon />
                            </Button>
                          }
                        />
                        <TooltipContent>
                          <p>Reset</p>
                        </TooltipContent>
                      </Tooltip>
                    </CardAction>
                  </CardHeader>
                  <CardContent className="flex-1 overflow-hidden p-0">
                    {messages.length === 0 ? (
                      <Empty className="h-full">
                        <EmptyHeader>
                          <EmptyMedia variant="icon">
                            <MessageCircleDashedIcon />
                          </EmptyMedia>
                          <EmptyTitle>Morning, shadcn!</EmptyTitle>
                          <EmptyDescription>
                            What are we working on today? Press send to start a
                            new conversation
                          </EmptyDescription>
                        </EmptyHeader>
                      </Empty>
                    ) : (
                      <MessageScroller>
                        <MessageScrollerViewport>
                          <MessageScrollerContent
                            aria-busy={isBusy}
                            className="p-(--card-spacing)"
                          >
                            {messages.map((message) => (
                              <MessageAnimated
                                key={message.id}
                                message={message}
                                scrollAnchor={message.role === "user"}
                              />
                            ))}
                          </MessageScrollerContent>
                        </MessageScrollerViewport>
                        <MessageScrollerButton />
                      </MessageScroller>
                    )}
                  </CardContent>
                  <CardFooter className="flex-col gap-2">
                    <form
                      onSubmit={(e) => {
                        e.preventDefault();
                        if (!nextMessage || isBusy) {
                          return;
                        }
                      }}
                      className="w-full"
                    >
                      <InputGroup>
                        <div className="h-14 w-full px-3 py-2.5">
                          <span
                            className="line-clamp-2 opacity-60 data-[status=ready]:opacity-100"
                            data-status={status}
                          >
                            {nextMessage ? (
                              getMessageText(nextMessage)
                            ) : (
                              <span className="text-muted-foreground">
                                No messages queued. Reset the conversation.
                              </span>
                            )}
                          </span>
                        </div>
                        <InputGroupAddon align="block-end" className="pt-1">
                          <DropdownMenu>
                            <DropdownMenuTrigger
                              render={
                                <InputGroupButton
                                  aria-label="Add files"
                                  type="button"
                                  size="icon-sm"
                                  variant="outline"
                                >
                                  <PlusIcon />
                                </InputGroupButton>
                              }
                            />
                            <DropdownMenuContent
                              align="start"
                              side="top"
                              className="w-44"
                            >
                              <DropdownMenuItem>
                                <PaperclipIcon />
                                Add Photos & Files
                              </DropdownMenuItem>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem>
                                <ImageIcon />
                                Create Image
                              </DropdownMenuItem>
                              <DropdownMenuItem>
                                <TelescopeIcon />
                                Deep Research
                              </DropdownMenuItem>
                              <DropdownMenuItem>
                                <GlobeIcon />
                                Web Search
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                          <InputGroupButton
                            type="submit"
                            variant="default"
                            size="icon-sm"
                            className="ml-auto"
                          >
                            <ArrowUpIcon />
                            <span className="sr-only">Send</span>
                          </InputGroupButton>
                        </InputGroupAddon>
                      </InputGroup>
                    </form>
                  </CardFooter>
                </Card>
              </div>
              <div className="px-0.5 text-center text-xs text-muted-foreground">
                Press send to send messages.
              </div>
            </MessageScrollerProvider>
          </div>
        </div>
      </div>
      <div className="settings w-64 flex flex-col gap-2 p-2">
        <div className="model flex flex-col gap-2 p-2">
          <Tooltip>
            <TooltipTrigger>
              <Label>Model </Label>
            </TooltipTrigger>
            <TooltipContent side="left" sideOffset={4}>
              <p>Choose the model you want to use for this conversation.</p>
            </TooltipContent>
          </Tooltip>
          <Combobox items={models} autoHighlight>
            <ComboboxInput placeholder="Select a model" showClear />
            <ComboboxContent>
              <ComboboxEmpty>No items found.</ComboboxEmpty>
              <ComboboxList>
                {(item) => (
                  <ComboboxItem key={item} value={item}>
                    {item.label}
                  </ComboboxItem>
                )}
              </ComboboxList>
            </ComboboxContent>
          </Combobox>
        </div>

        <div className="temperature flex flex-col gap-4 p-2">
          <div className="flex flex-row items-center justify-between gap-2">
            <Tooltip>
              <TooltipTrigger>
                <Label>Temperature</Label>
              </TooltipTrigger>
              <TooltipContent side="left" sideOffset={4}>
                <p>
                  Slide to adjust the temperature of the model. Higher values
                  will make the model more creative, while lower values will
                  make it more focused and deterministic.
                </p>
              </TooltipContent>
            </Tooltip>
            <div className="text-muted-foreground text-xs">
              <span contentEditable="true">0.02</span>
            </div>
          </div>
          <div className="">
            <Slider
              defaultValue={[0]}
              max={1}
              min={-1}
              step={0.01}
              className="mx-auto w-full max-w-xs"
            />
          </div>
        </div>

        <div className="topp flex flex-col gap-4 p-2">
          <div className="flex flex-row items-center justify-between gap-2">
            <Tooltip>
              <TooltipTrigger>
                <Label>Top P</Label>
              </TooltipTrigger>
              <TooltipContent side="left" sideOffset={4}>
                <p>
                  Slide to adjust the top p of the model. Higher values will
                  make the model more creative, while lower values will make it
                  more focused and deterministic.
                </p>
              </TooltipContent>
            </Tooltip>
            <div className="text-muted-foreground text-xs">
              <span contentEditable="true">0.02</span>
            </div>
          </div>
          <div className="">
            <Slider
              defaultValue={[0]}
              max={1}
              min={-1}
              step={0.01}
              className="mx-auto w-full max-w-xs"
            />
          </div>
        </div>

        <div className="max-tokens flex flex-col gap-4 p-2">
          <div className="flex flex-row items-center justify-between gap-2">
            <Tooltip>
              <TooltipTrigger>
                <Label>Max tokens</Label>
              </TooltipTrigger>
              <TooltipContent side="left" sideOffset={4}>
                <p></p>
              </TooltipContent>
            </Tooltip>
            <div className="text-muted-foreground text-xs">
              <span contentEditable="true">0.02</span>
            </div>
          </div>
          <div className="">
            <Input
              defaultValue={5076}
              type="number"
              min={1}
              className="mx-auto w-full max-w-xs"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
