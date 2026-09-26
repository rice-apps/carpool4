import test from "node:test";
import assert from "node:assert/strict";
import { safeReturnTo } from "./returnTo";

test("return navigation stays on this site", () => {
  assert.equal(safeReturnTo("/rides/123"), "/rides/123");
  assert.equal(safeReturnTo("https://other.example/path"), "/");
  assert.equal(safeReturnTo("//other.example/path"), "/");
  assert.equal(safeReturnTo("/\\other.example"), "/");
  assert.equal(safeReturnTo("/\n/other.example"), "/");
  assert.equal(safeReturnTo("/\t/other.example"), "/");
});
