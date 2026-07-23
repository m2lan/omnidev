"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { MessageProcessor } from "@a2ui/web_core/v0_9";
import { A2uiSurface, basicCatalog } from "@a2ui/react/v0_9";
import type { ReactComponentImplementation } from "@a2ui/react/v0_9";
import type { SurfaceModel } from "@a2ui/web_core/v0_9";
import { shadcnCatalog } from "./shadcn-catalog";

interface A2UIRendererProps {
  /** A2UI JSON messages to render */
  messages: object[];
  /** Optional: called when user triggers an action */
  onAction?: (action: {
    name: string;
    surfaceId: string;
    sourceComponentId: string;
    context: Record<string, unknown>;
  }) => void;
  /** Optional CSS class for the container */
  className?: string;
}

/**
 * A2UI Renderer component.
 * Renders A2UI JSON messages as interactive UI surfaces.
 */
export function A2UIRenderer({
  messages,
  onAction,
  className,
}: A2UIRendererProps) {
  const [processor] = useState(() => {
    const p = new MessageProcessor<ReactComponentImplementation>(
      [basicCatalog, shadcnCatalog],
      onAction
        ? (action) => {
            onAction({
              name: action.name,
              surfaceId: action.surfaceId,
              sourceComponentId: action.sourceComponentId,
              context: action.context as Record<string, unknown>,
            });
          }
        : undefined
    );
    return p;
  });

  const [surfaces, setSurfaces] = useState<
    SurfaceModel<ReactComponentImplementation>[]
  >([]);

  // Sync surfaces on creation/deletion
  useEffect(() => {
    const sync = () => {
      setSurfaces(
        Array.from(processor.model.surfacesMap.values())
      );
    };
    const createdSub = processor.onSurfaceCreated(sync);
    const deletedSub = processor.onSurfaceDeleted(sync);
    // Initial sync
    sync();
    return () => {
      createdSub.unsubscribe();
      deletedSub.unsubscribe();
    };
  }, [processor]);

  // Process incoming messages
  const processedCountRef = useRef(0);

  useEffect(() => {
    if (messages.length === 0) {
      processedCountRef.current = 0;
      return;
    }

    // Only process new messages
    const newMessages = messages.slice(processedCountRef.current);
    if (newMessages.length > 0) {
      try {
        // Split createSurface messages that include components into
        // separate createSurface + updateComponents messages,
        // because the processor's processCreateSurfaceMessage ignores inline components.
        const expanded: object[] = [];
        for (const msg of newMessages) {
          const m = msg as Record<string, unknown>;
          if (m.createSurface) {
            const cs = m.createSurface as Record<string, unknown>;
            if (cs.components) {
              // createSurface without components
              const { components, ...surfaceRest } = cs;
              expanded.push({ version: m.version, createSurface: surfaceRest });
              // updateComponents with the components
              expanded.push({
                version: m.version,
                updateComponents: {
                  surfaceId: cs.surfaceId,
                  components,
                },
              });
              continue;
            }
          }
          expanded.push(msg);
        }
        processor.processMessages(expanded as Parameters<typeof processor.processMessages>[0]);
        processedCountRef.current = messages.length;
      } catch (err) {
        console.error("A2UI: Failed to process messages:", err);
      }
    }
  }, [messages, processor]);

  if (surfaces.length === 0) {
    return null;
  }

  return (
    <div className={className}>
      {surfaces.map((surface) => (
        <A2uiSurface key={surface.id} surface={surface} />
      ))}
    </div>
  );
}

/**
 * Hook to create a MessageProcessor for advanced use cases.
 */
export function useA2UIProcessor(
  onAction?: (action: {
    name: string;
    surfaceId: string;
    sourceComponentId: string;
    context: Record<string, unknown>;
  }) => void
) {
  const [processor] = useState(() => {
    return new MessageProcessor<ReactComponentImplementation>(
      [basicCatalog, shadcnCatalog],
      onAction
        ? (action) => {
            onAction({
              name: action.name,
              surfaceId: action.surfaceId,
              sourceComponentId: action.sourceComponentId,
              context: action.context as Record<string, unknown>,
            });
          }
        : undefined
    );
  });

  return processor;
}
