import { createConnectTransport } from "@connectrpc/connect-web";
import { supabase } from "./supabase";

export const transport = createConnectTransport({
  baseUrl: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
  interceptors: [
    (next) => async (request) => {
      const { data } = await supabase.auth.getSession();
      if (data.session?.access_token) {
        request.header.set(
          "Authorization",
          `Bearer ${data.session.access_token}`,
        );
      }
      return next(request);
    },
  ],
});
