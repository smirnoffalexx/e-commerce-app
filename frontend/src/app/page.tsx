import { PackageSearchIcon, ShieldCheckIcon, TruckIcon } from "lucide-react";

const features = [
  { icon: PackageSearchIcon, title: "Full-text search", text: "Find products by name, SKU or description." },
  { icon: ShieldCheckIcon, title: "Secure checkout", text: "Prices and stock are verified when you pay." },
  { icon: TruckIcon, title: "Order history", text: "Track every order from your account." },
];

export default function HomePage() {
  return (
    <section className="relative isolate overflow-hidden">
      <div className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(70%_60%_at_50%_0%,color-mix(in_oklch,var(--primary)_18%,transparent),transparent)]" />
      <div className="mx-auto max-w-7xl px-4 py-24 text-center sm:px-6">
        <h1 className="mx-auto max-w-2xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl">
          Everything for your workshop, in one place
        </h1>
        <p className="mx-auto mt-4 max-w-xl text-lg text-muted-foreground">
          Browse the catalog, compare prices and check out in seconds.
        </p>
        <div className="mx-auto mt-16 grid max-w-4xl gap-6 sm:grid-cols-3">
          {features.map(({ icon: Icon, title, text }) => (
            <div key={title} className="rounded-xl border bg-card p-6 text-left shadow-sm">
              <Icon className="size-6 text-primary" />
              <h2 className="mt-4 font-medium">{title}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
