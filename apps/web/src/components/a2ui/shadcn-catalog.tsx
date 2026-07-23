"use client";

import { Catalog } from "@a2ui/web_core/v0_9";
import {
  createComponentImplementation,
  basicCatalog,
  type ReactComponentImplementation,
} from "@a2ui/react/v0_9";
import { z } from "zod";

// Shadcn/ui mapped A2UI components
// These extend the basic catalog with Shadcn-styled implementations

const ShadcnButtonApi = {
  name: "ShadcnButton",
  schema: z.object({
    label: z.string(),
    variant: z.enum(["default", "destructive", "outline", "secondary", "ghost", "link"]).optional(),
    size: z.enum(["default", "sm", "lg", "icon"]).optional(),
    disabled: z.boolean().optional(),
  }),
};

const ShadcnButton = createComponentImplementation(
  ShadcnButtonApi,
  ({ props }) => {
    const variantClasses = {
      default: "bg-primary text-primary-foreground hover:bg-primary/90",
      destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
      outline: "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
      secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
      ghost: "hover:bg-accent hover:text-accent-foreground",
      link: "text-primary underline-offset-4 hover:underline",
    };
    const sizeClasses = {
      default: "h-10 px-4 py-2",
      sm: "h-9 rounded-md px-3",
      lg: "h-11 rounded-md px-8",
      icon: "h-10 w-10",
    };
    const variant = props.variant || "default";
    const size = props.size || "default";
    return (
      <button
        className={`inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 ${variantClasses[variant]} ${sizeClasses[size]}`}
        disabled={props.disabled}
      >
        {props.label}
      </button>
    );
  }
);

const ShadcnInputApi = {
  name: "ShadcnInput",
  schema: z.object({
    placeholder: z.string().optional(),
    type: z.string().optional(),
    disabled: z.boolean().optional(),
  }),
};

const ShadcnInput = createComponentImplementation(
  ShadcnInputApi,
  ({ props }) => {
    return (
      <input
        type={props.type || "text"}
        placeholder={props.placeholder}
        disabled={props.disabled}
        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
      />
    );
  }
);

const ShadcnBadgeApi = {
  name: "ShadcnBadge",
  schema: z.object({
    label: z.string(),
    variant: z.enum(["default", "secondary", "destructive", "outline"]).optional(),
  }),
};

const ShadcnBadge = createComponentImplementation(
  ShadcnBadgeApi,
  ({ props }) => {
    const variantClasses = {
      default: "bg-primary text-primary-foreground hover:bg-primary/80",
      secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
      destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/80",
      outline: "text-foreground border",
    };
    const variant = props.variant || "default";
    return (
      <span
        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${variantClasses[variant]}`}
      >
        {props.label}
      </span>
    );
  }
);

const ShadcnProgressApi = {
  name: "ShadcnProgress",
  schema: z.object({
    value: z.number().min(0).max(100),
    label: z.string().optional(),
  }),
};

const ShadcnProgress = createComponentImplementation(
  ShadcnProgressApi,
  ({ props }) => {
    return (
      <div className="w-full">
        {props.label && (
          <div className="flex justify-between text-sm mb-1">
            <span>{props.label}</span>
            <span>{props.value}%</span>
          </div>
        )}
        <div className="relative h-4 w-full overflow-hidden rounded-full bg-secondary">
          <div
            className="h-full w-full flex-1 bg-primary transition-all"
            style={{ transform: `translateX(-${100 - props.value}%)` }}
          />
        </div>
      </div>
    );
  }
);

const ShadcnAlertApi = {
  name: "ShadcnAlert",
  schema: z.object({
    title: z.string().optional(),
    description: z.string(),
    variant: z.enum(["default", "destructive"]).optional(),
  }),
};

const ShadcnAlert = createComponentImplementation(
  ShadcnAlertApi,
  ({ props }) => {
    const variantClasses = {
      default: "bg-background text-foreground border",
      destructive: "border-destructive/50 text-destructive dark:border-destructive [&>svg]:text-destructive",
    };
    const variant = props.variant || "default";
    return (
      <div
        role="alert"
        className={`relative w-full rounded-lg p-4 [&>svg+div]:translate-y-[-3px] [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:text-foreground [&>svg~*]:pl-7 ${variantClasses[variant]}`}
      >
        {props.title && <h5 className="mb-1 font-medium leading-none tracking-tight">{props.title}</h5>}
        <div className="text-sm [&_p]:leading-relaxed">{props.description}</div>
      </div>
    );
  }
);

/**
 * Extended catalog with Shadcn/ui components.
 * Merges the basic A2UI catalog with custom Shadcn-styled components.
 */
export const shadcnCatalog = new Catalog<ReactComponentImplementation>(
  "https://omnidev.dev/catalogs/shadcn/v1.json",
  [
    ...Array.from(basicCatalog.components.values()),
    ShadcnButton,
    ShadcnInput,
    ShadcnBadge,
    ShadcnProgress,
    ShadcnAlert,
  ],
  [...Array.from(basicCatalog.functions.values())]
);
