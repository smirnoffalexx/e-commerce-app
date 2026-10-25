import { describe, expect, it } from "vitest";
import { pageWindow } from "./pagination";

describe("pageWindow", () => {
  it("shows every page when there are few", () => {
    expect(pageWindow(2, 5)).toEqual([1, 2, 3, 4, 5]);
  });

  it("keeps the first and last pages around a window of neighbours", () => {
    expect(pageWindow(10, 20)).toEqual([1, "gap", 8, 9, 10, 11, 12, "gap", 20]);
    expect(pageWindow(1, 20)).toEqual([1, 2, 3, 4, 5, 6, "gap", 20]);
    expect(pageWindow(20, 20)).toEqual([1, "gap", 15, 16, 17, 18, 19, 20]);
  });
});
