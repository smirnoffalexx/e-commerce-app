import { describe, expect, it } from "vitest";
import { allowedOrigins, isTrustedRequest } from "./origin";

const allowed = ["http://localhost:3000"];

describe("isTrustedRequest", () => {
  it("lets safe methods through without an Origin header", () => {
    expect(isTrustedRequest("GET", new Headers(), allowed)).toBe(true);
  });

  it("accepts mutations from the app origin", () => {
    expect(isTrustedRequest("POST", new Headers({ origin: "http://localhost:3000" }), allowed)).toBe(true);
  });

  it("rejects mutations from another site", () => {
    expect(isTrustedRequest("DELETE", new Headers({ origin: "https://evil.example" }), allowed)).toBe(false);
  });

  it("rejects mutations without Origin unless the browser marks them same-origin", () => {
    expect(isTrustedRequest("POST", new Headers(), allowed)).toBe(false);
    expect(isTrustedRequest("POST", new Headers({ "sec-fetch-site": "cross-site" }), allowed)).toBe(false);
    expect(isTrustedRequest("POST", new Headers({ "sec-fetch-site": "same-origin" }), allowed)).toBe(true);
  });
});

describe("allowedOrigins", () => {
  it("prefers configured origins and normalises trailing slashes", () => {
    expect(allowedOrigins("https://shop.example/, http://localhost:3000", "http://internal:3000")).toEqual([
      "https://shop.example",
      "http://localhost:3000",
    ]);
  });

  it("falls back to the request origin", () => {
    expect(allowedOrigins(undefined, "http://localhost:3000")).toEqual(["http://localhost:3000"]);
  });
});
