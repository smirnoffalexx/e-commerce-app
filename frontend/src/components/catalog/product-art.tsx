import {
  Apple,
  BookOpen,
  Dumbbell,
  Footprints,
  Gamepad2,
  Gift,
  HeartPulse,
  Home,
  Laptop,
  Mountain,
  Package,
  PawPrint,
  PenTool,
  Shirt,
  Sparkles,
  UtensilsCrossed,
  Watch,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

const icons: Record<string, LucideIcon> = {
  accessories: Watch,
  beauty: Sparkles,
  books: BookOpen,
  clothing: Shirt,
  electronics: Laptop,
  "food-and-beverage": Apple,
  footwear: Footprints,
  games: Gamepad2,
  gifts: Gift,
  health: HeartPulse,
  "home-and-office": Home,
  kitchen: UtensilsCrossed,
  outdoors: Mountain,
  pets: PawPrint,
  sports: Dumbbell,
  stationery: PenTool,
  tools: Wrench,
};

function hue(seed: string): number {
  let hash = 0;
  for (const char of seed) {
    hash = (hash * 31 + char.charCodeAt(0)) % 360;
  }
  return hash;
}

export function ProductArt({ slug, name, className }: { slug?: string; name: string; className?: string }) {
  const Icon = (slug && icons[slug]) || Package;
  const h = hue(slug ?? name);
  return (
    <div
      aria-hidden
      className={cn("flex items-center justify-center overflow-hidden", className)}
      style={{
        background: `linear-gradient(135deg, oklch(0.94 0.05 ${h}), oklch(0.86 0.09 ${(h + 40) % 360}))`,
      }}
    >
      <Icon className="size-1/3 text-black/45" strokeWidth={1.25} />
    </div>
  );
}
