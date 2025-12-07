import {createClient} from "@connectrpc/connect";
import {GreetService} from "$lib/gen/api/v1/greet_pb"
import {createConnectTransport} from "@connectrpc/connect-web";
import {TodoService} from "$lib/gen/api/v1/todo_pb";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/";

const transport = createConnectTransport({
    baseUrl: API_BASE_URL,
});

export const Greet = createClient(GreetService, transport);
export const Todo = createClient(TodoService, transport);
