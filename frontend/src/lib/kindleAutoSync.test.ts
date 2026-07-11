import { beforeEach, describe, expect, it } from "vitest";
import {
  KINDLE_AUTO_SYNC_COOLDOWN_MS,
  buildVisibleBookSignature,
  readKindleAutoSyncSnapshot,
  shouldSkipKindleAutoSync,
  writeKindleAutoSyncSnapshot,
  type KindleAutoSyncSnapshot,
} from "./kindleAutoSync";
import type { ExtensionKindleBook } from "../types/kindle";

function book(overrides: Partial<ExtensionKindleBook>): ExtensionKindleBook {
  return {
    id: "b1",
    asin: "A1",
    book_title: "title",
    book_author: "author",
    ...overrides,
  } as ExtensionKindleBook;
}

describe("buildVisibleBookSignature", () => {
  it("is order-independent", () => {
    const a = book({ id: "b1", asin: "A1" });
    const b = book({ id: "b2", asin: "A2" });
    expect(buildVisibleBookSignature([a, b])).toBe(
      buildVisibleBookSignature([b, a]),
    );
  });

  it("changes when book identity changes", () => {
    expect(buildVisibleBookSignature([book({ asin: "A1" })])).not.toBe(
      buildVisibleBookSignature([book({ asin: "A2" })]),
    );
  });
});

describe("shouldSkipKindleAutoSync", () => {
  const now = Date.now();
  const signature = buildVisibleBookSignature([book({})]);
  const snapshot: KindleAutoSyncSnapshot = {
    status: "done",
    message: "",
    synced_at: new Date(now - 60_000).toISOString(),
    visible_signature: signature,
  };

  it("skips within cooldown when signature matches", () => {
    expect(shouldSkipKindleAutoSync(snapshot, signature, now)).toBe(true);
  });

  it("does not skip after cooldown", () => {
    expect(
      shouldSkipKindleAutoSync(
        snapshot,
        signature,
        now + KINDLE_AUTO_SYNC_COOLDOWN_MS,
      ),
    ).toBe(false);
  });

  it("does not skip when visible books changed", () => {
    expect(shouldSkipKindleAutoSync(snapshot, "other-signature", now)).toBe(
      false,
    );
  });

  it("does not skip without a snapshot or synced_at", () => {
    expect(shouldSkipKindleAutoSync(null, signature, now)).toBe(false);
    expect(
      shouldSkipKindleAutoSync(
        { status: "idle", message: "" },
        signature,
        now,
      ),
    ).toBe(false);
  });
});

describe("snapshot storage round-trip", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("writes and reads back a snapshot", () => {
    const snapshot: KindleAutoSyncSnapshot = {
      status: "done",
      message: "同期完了",
      saved_count: 3,
    };
    writeKindleAutoSyncSnapshot(snapshot);
    expect(readKindleAutoSyncSnapshot()).toEqual(snapshot);
  });

  it("returns null for corrupted storage", () => {
    window.localStorage.setItem(
      "ai-study-tool:kindle-auto-sync:v1",
      "{not json",
    );
    expect(readKindleAutoSyncSnapshot()).toBeNull();
  });
});
