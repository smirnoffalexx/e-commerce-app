import { SearchIcon } from "lucide-react";
import Form from "next/form";
import Link from "next/link";
import type { Category, ProductQuery } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export function ProductFilters({ query, categories }: { query: ProductQuery; categories: Category[] }) {
  return (
    <Form action="/admin/products" className="mb-4 flex flex-col gap-2 sm:flex-row">
      <InputGroup className="bg-card sm:max-w-sm">
        <InputGroupAddon>
          <SearchIcon />
        </InputGroupAddon>
        <InputGroupInput name="q" defaultValue={query.q} placeholder="Name, SKU or description" aria-label="Search products" />
      </InputGroup>
      <Select name="category_id" defaultValue={query.category_id ?? "all"}>
        <SelectTrigger className="bg-card sm:w-52" aria-label="Category">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All categories</SelectItem>
          {categories.map((category) => (
            <SelectItem key={category.id} value={category.id}>
              {category.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button type="submit" variant="secondary">
        Search
      </Button>
      {(query.q || query.category_id) && (
        <Button variant="ghost" asChild>
          <Link href="/admin/products">Reset</Link>
        </Button>
      )}
    </Form>
  );
}
