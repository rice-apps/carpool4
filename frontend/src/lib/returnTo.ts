export const returnToKey = "carpool-return-to";

export function safeReturnTo(value: string | null): string {
  if (!value?.startsWith("/") || value.startsWith("//") || value.includes("\\")) {
    return "/";
  }
  for (const character of value) {
    const code = character.charCodeAt(0);
    if (code < 32 || code === 127) return "/";
  }
  return value;
}
