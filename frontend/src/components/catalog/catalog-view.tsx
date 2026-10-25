"use client";

import { SearchIcon, SlidersHorizontalIcon, XIcon } from "lucide-react";
import { useRouter } from "next/navigation";
import { createContext, use, useCallback, useEffect, useState, useTransition } from "react";
import type { Category, ProductQuery, ProductSort } from "@/lib/api/types";
import { SORT_OPTIONS, catalogHref, defaultSort, hasFilters } from "@/lib/catalog/query";
import { isMoney, toCents } from "@/lib/money";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";

const ALL_CATEGORIES = "all";

interface Navigation {
  query: ProductQuery;
  categories: Category[];
  pending: boolean;
  navigate: (changes: Partial<ProductQuery>) => void;
}

const NavigationContext = createContext<Navigation | null>(null);

function useNavigation(): Navigation {
  const value = use(NavigationContext);
  if (!value) {
    throw new Error("catalog controls must be rendered inside CatalogView");
  }
  return value;
}

export function CatalogView({
  query,
  categories,
  total,
  children,
}: {
  query: ProductQuery;
  categories: Category[];
  total: number | null;
  children: React.ReactNode;
}) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  const navigate = useCallback(
    (changes: Partial<ProductQuery>) => {
      startTransition(() => router.replace(catalogHref(query, changes), { scroll: false }));
    },
    [query, router],
  );

  const navigation = { query, categories, pending, navigate };
  return (
    <NavigationContext value={navigation}>
      <div className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">Catalog</h1>
            <p className="mt-1 flex items-center gap-2 text-muted-foreground">
              {total === null ? "Browse our products" : `${total} ${total === 1 ? "product" : "products"}`}
              {pending && <Spinner className="size-4" />}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <MobileFilters />
            <SortSelect />
          </div>
        </div>
        <div className="mt-6">
          <SearchBox />
        </div>
        <div className="mt-8 grid gap-8 lg:grid-cols-[240px_1fr]">
          <aside className="hidden lg:block">
            <Filters />
          </aside>
          <div className={cn("min-w-0 transition-opacity", pending && "opacity-60")}>{children}</div>
        </div>
      </div>
    </NavigationContext>
  );
}

function SearchBox() {
  const { query, navigate, pending } = useNavigation();
  const current = query.q ?? "";
  const [value, setValue] = useState(current);
  const [seen, setSeen] = useState(current);
  const [requested, setRequested] = useState(current);

  if (current !== seen) {
    setSeen(current);
    if (current !== requested) {
      setValue(current);
      setRequested(current);
    }
  }

  useEffect(() => {
    const text = value.trim();
    if (text === current) {
      return;
    }
    const timer = setTimeout(() => {
      setRequested(text);
      navigate({ q: text || undefined, sort: undefined });
    }, 350);
    return () => clearTimeout(timer);
  }, [value, current, navigate]);

  return (
    <InputGroup role="search" className="h-11 bg-card">
      <InputGroupAddon>{pending ? <Spinner /> : <SearchIcon />}</InputGroupAddon>
      <InputGroupInput
        type="text"
        enterKeyHint="search"
        placeholder="Search by name, SKU or description"
        aria-label="Search products"
        value={value}
        maxLength={200}
        onChange={(event) => setValue(event.target.value)}
      />
      {value && (
        <InputGroupAddon align="inline-end">
          <InputGroupButton size="icon-xs" aria-label="Clear search" onClick={() => setValue("")}>
            <XIcon />
          </InputGroupButton>
        </InputGroupAddon>
      )}
    </InputGroup>
  );
}

function SortSelect() {
  const { query, navigate } = useNavigation();
  const options = SORT_OPTIONS.filter((option) => option.value !== "relevance" || query.q);
  return (
    <Select value={query.sort ?? defaultSort(query)} onValueChange={(sort) => navigate({ sort: sort as ProductSort })}>
      <SelectTrigger className="w-48 bg-card" aria-label="Sort products">
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="end">
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function MobileFilters() {
  const { query } = useNavigation();
  const active = [query.category_id, query.min_price || query.max_price, query.in_stock].filter(Boolean).length;
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="outline" className="bg-card lg:hidden">
          <SlidersHorizontalIcon />
          Filters
          {active > 0 && <span className="rounded-full bg-primary px-1.5 text-xs text-primary-foreground">{active}</span>}
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-80">
        <SheetHeader>
          <SheetTitle>Filters</SheetTitle>
        </SheetHeader>
        <div className="px-4">
          <Filters />
        </div>
      </SheetContent>
    </Sheet>
  );
}

function Filters() {
  const { query, categories, navigate } = useNavigation();
  return (
    <div className="flex flex-col gap-6">
      <Field>
        <FieldLabel>Category</FieldLabel>
        <Select
          value={query.category_id ?? ALL_CATEGORIES}
          onValueChange={(value) => navigate({ category_id: value === ALL_CATEGORIES ? undefined : value })}
        >
          <SelectTrigger className="w-full bg-card" aria-label="Category">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_CATEGORIES}>All categories</SelectItem>
            {categories.map((category) => (
              <SelectItem key={category.id} value={category.id}>
                {category.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      <PriceRange key={`${query.min_price ?? ""}-${query.max_price ?? ""}`} />
      <div className="flex items-center justify-between gap-3">
        <Label htmlFor="in-stock">In stock only</Label>
        <Switch
          id="in-stock"
          checked={Boolean(query.in_stock)}
          onCheckedChange={(checked) => navigate({ in_stock: checked || undefined })}
        />
      </div>
      {hasFilters(query) && (
        <Button
          variant="ghost"
          className="justify-start px-0 text-muted-foreground hover:bg-transparent hover:text-foreground"
          onClick={() =>
            navigate({ q: undefined, category_id: undefined, min_price: undefined, max_price: undefined, in_stock: undefined, sort: undefined })
          }
        >
          <XIcon />
          Clear all filters
        </Button>
      )}
    </div>
  );
}

function PriceRange() {
  const { query, navigate } = useNavigation();
  const [min, setMin] = useState(query.min_price ?? "");
  const [max, setMax] = useState(query.max_price ?? "");
  const [error, setError] = useState<string | null>(null);

  function apply() {
    const lo = min.trim();
    const hi = max.trim();
    if ((lo && !isMoney(lo)) || (hi && !isMoney(hi))) {
      setError("Enter amounts like 10 or 9.99.");
      return;
    }
    if (lo && hi && toCents(lo) > toCents(hi)) {
      setError("Minimum must not exceed maximum.");
      return;
    }
    setError(null);
    if (lo !== (query.min_price ?? "") || hi !== (query.max_price ?? "")) {
      navigate({ min_price: lo || undefined, max_price: hi || undefined });
    }
  }

  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel>Price, $</FieldLabel>
      <form
        className="flex items-center gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          apply();
        }}
      >
        <Input inputMode="decimal" placeholder="Min" aria-label="Minimum price" value={min} onChange={(e) => setMin(e.target.value)} onBlur={apply} className="bg-card" />
        <span className="text-muted-foreground">–</span>
        <Input inputMode="decimal" placeholder="Max" aria-label="Maximum price" value={max} onChange={(e) => setMax(e.target.value)} onBlur={apply} className="bg-card" />
        <button type="submit" hidden />
      </form>
      <FieldError>{error}</FieldError>
    </Field>
  );
}
