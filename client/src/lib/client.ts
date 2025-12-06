import { createClient } from "@connectrpc/connect";
import { GreetService } from "$lib/gen/greet/v1/greet_pb"
import { createConnectTransport } from "@connectrpc/connect-web";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/";

const transport = createConnectTransport({
    baseUrl: API_BASE_URL,
});

export const Client = createClient(GreetService, transport);