import { describe, expect, it } from "vitest";
import { AxiosError, AxiosHeaders } from "axios";
import { getApiErrorMessage, getApiErrorStatus } from "./errors";

function axiosErrorWith(status: number, data: unknown): AxiosError {
  const config = { headers: new AxiosHeaders() };
  return new AxiosError("failed", "ERR_BAD_REQUEST", config, {}, {
    status,
    statusText: "",
    headers: {},
    config,
    data,
  } as never);
}

describe("getApiErrorMessage", () => {
  it("reads top-level message", () => {
    const error = axiosErrorWith(400, { message: " invalid input " });
    expect(getApiErrorMessage(error)).toBe("invalid input");
  });

  it("reads nested error.message", () => {
    const error = axiosErrorWith(400, { error: { message: "nested" } });
    expect(getApiErrorMessage(error)).toBe("nested");
  });

  it("returns empty string for non-axios errors and non-string bodies", () => {
    expect(getApiErrorMessage(new Error("plain"))).toBe("");
    expect(getApiErrorMessage(axiosErrorWith(400, { message: 123 }))).toBe("");
    expect(getApiErrorMessage(axiosErrorWith(400, undefined))).toBe("");
  });
});

describe("getApiErrorStatus", () => {
  it("returns response status for axios errors", () => {
    expect(getApiErrorStatus(axiosErrorWith(403, {}))).toBe(403);
  });

  it("returns 0 for non-axios errors", () => {
    expect(getApiErrorStatus(new Error("plain"))).toBe(0);
  });
});
