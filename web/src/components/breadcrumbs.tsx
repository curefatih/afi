import * as React from "react";
import { Link, useMatches } from "@tanstack/react-router";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"; // Adjust path to your shadcn setup

export function DynamicBreadcrumb() {
  // Get all active route matches for the current URL path
  const matches = useMatches();

  // Filter out the root layout and pathless routes that don't need UI names
  const breadcrumbMatches = matches.filter(
    (match) => match.staticData?.getTitle && match.pathname !== "/_authenticated/app",
  );

  if (breadcrumbMatches.length === 0) return null;

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink>
            <Link to="/app/dashboard">Home</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>

        {breadcrumbMatches.map((match, index) => {
          const isLast = index === breadcrumbMatches.length - 1;

          const routeName = match.staticData?.getTitle?.();

          return (
            <React.Fragment key={match.id}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage className="capitalize">
                    {routeName}
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink className="capitalize">
                    <Link to={match.pathname}>{routeName}</Link>
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </React.Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
