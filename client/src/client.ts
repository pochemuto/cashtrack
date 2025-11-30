import { createClient } from "@connectrpc/connect";
import { GreetService } from "../../gen/greet/v1/greet_pb"
import { createConnectTransport } from "@connectrpc/connect-web";

const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
});

export const Client = createClient(GreetService, transport);