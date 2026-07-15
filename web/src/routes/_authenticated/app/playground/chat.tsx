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
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
} from "#/components/ui/input-group";
import { MessageAnimated } from "#/components/ui/message-animated";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "#/components/ui/message-scroller";
import { Separator } from "#/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "#/components/ui/tooltip";
import { createFileRoute } from "@tanstack/react-router";
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

export const Route = createFileRoute("/_authenticated/app/playground/chat")({
  staticData: {
    getTitle: () => "Playground",
  },
  component: RouteComponent,
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

type Message = {
  id: string;
  role: "user" | "assistant";
  content: Array<{
    type: "text";
    text: string;
  }>;
};

function RouteComponent() {
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
  }

  return (
    <div className="flex flex-row h-full gap-4">
      <div className="chat flex-1 h-full">
        <Card className="flex h-full flex-row p-2">
          <div className="threads w-64 border-r p-2">
            <div className="">
              <h5 className="font-bold">Threads</h5>
              <span className="font-thin ">Conversations that you have created before</span>
            </div>
            <Separator className={"m-2"}/>
            <Button variant={"ghost"} className={"w-full text-left cursor-pointer"}>
              text
            </Button>
          </div>
          <div className="flex-1 h-full">
            <MessageScrollerProvider>
              <div className="relative h-full p-2">
                <Card className="mx-auto h-full w-full max-w-sm gap-0">
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
                              disabled={isBusy}
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
                            disabled={!nextMessage || isBusy}
                            className="ml-auto"
                          >
                            <ArrowUpIcon />
                            <span className="sr-only">Send</span>
                          </InputGroupButton>
                        </InputGroupAddon>
                      </InputGroup>
                    </form>
                    <div className="px-0.5 text-center text-xs text-muted-foreground">
                      Press send to send messages.
                    </div>
                  </CardFooter>
                </Card>
              </div>
            </MessageScrollerProvider>
          </div>
        </Card>
      </div>
      <div className="settings w-64"></div>
    </div>
  );
}
