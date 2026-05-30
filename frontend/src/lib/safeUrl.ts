export function safeHttpUrl(value?: string | null): string | undefined {
  if (!value) {
    return undefined;
  }

  try {
    const parsed = new URL(value);
    // Keep display/navigation URLs absolute and browser-safe. Relative URLs are
    // rejected so user-controlled values cannot unexpectedly target app routes.
    if (parsed.protocol === "http:" || parsed.protocol === "https:") {
      return parsed.href;
    }
  } catch {
    return undefined;
  }

  return undefined;
}
