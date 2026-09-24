import { Code, ConnectError } from "@connectrpc/connect";

export function isUnimplemented(error: unknown): boolean {
  return ConnectError.from(error).code === Code.Unimplemented;
}

export function requestErrorMessage(
  error: unknown,
  operation: string,
  fallback: string,
): string {
  return isUnimplemented(error)
    ? `${operation} isn't implemented yet.`
    : fallback;
}
