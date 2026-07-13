import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from "@/components/ui/breadcrumb";
import { Link, useMatches } from "@tanstack/react-router";
import * as React from "react";

export function DynamicBreadcrumb() {
  const matches = useMatches();

  // Filter out system layout tracks
  const baseMatches = matches.filter(
    (match) =>
      match.id !== "__root__" &&
      match.pathname !== "/_authenticated/app" &&
      match.pathname !== "/app" &&
      !match.id.endsWith("/_layout"),
  );

  // ✅ Deduplicate sequential entries with matching pathnames
  const breadcrumbMatches = baseMatches.filter((match, index) => {
    if (index === 0) return true;
    // If this match has the exact same pathname as the previous one, skip it
    return match.pathname !== baseMatches[index - 1].pathname;
  });

  if (breadcrumbMatches.length === 0) return null;

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <Link to="/app/dashboard">Home</Link>
        </BreadcrumbItem>

        {breadcrumbMatches.map((match, index) => {
          const isLast = index === breadcrumbMatches.length - 1;

          const fallbackName =
            match.id.split("/").pop()?.replace(/-/g, " ") || "";
          const routeName = match.staticData?.getTitle?.() || fallbackName;

          if (!routeName || routeName.trim() === "" || routeName === "index")
            return null;

          return (
            <React.Fragment key={match.id}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage className="capitalize">
                    {routeName}
                  </BreadcrumbPage>
                ) : (
                  <Link to={match.pathname}>{routeName}</Link>
                )}
              </BreadcrumbItem>
            </React.Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
