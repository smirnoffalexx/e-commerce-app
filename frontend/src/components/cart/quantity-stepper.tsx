"use client";

import { MinusIcon, PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function QuantityStepper({
  value,
  max,
  onChange,
  size = "default",
  label = "Quantity",
}: {
  value: number;
  max: number;
  onChange: (value: number) => void;
  size?: "default" | "sm";
  label?: string;
}) {
  const buttonSize = size === "sm" ? "icon-sm" : "icon";
  return (
    <div className={cn("inline-flex items-center rounded-lg border bg-card", size === "sm" ? "h-8" : "h-9")} role="group" aria-label={label}>
      <Button variant="ghost" size={buttonSize} aria-label="Decrease quantity" disabled={value <= 1} onClick={() => onChange(value - 1)}>
        <MinusIcon />
      </Button>
      <input
        aria-label={label}
        inputMode="numeric"
        className="w-10 bg-transparent text-center text-sm tabular-nums outline-none"
        value={value}
        onChange={(event) => {
          const next = Number(event.target.value.replace(/\D/g, ""));
          if (next >= 1) {
            onChange(Math.min(next, Math.max(1, max)));
          }
        }}
      />
      <Button variant="ghost" size={buttonSize} aria-label="Increase quantity" disabled={value >= max} onClick={() => onChange(value + 1)}>
        <PlusIcon />
      </Button>
    </div>
  );
}
