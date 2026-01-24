import {createClient} from "@connectrpc/connect";
import {GreetService} from "$lib/gen/api/v1/greet_pb";
import {createConnectTransport} from "@connectrpc/connect-web";
import {TodoService} from "$lib/gen/api/v1/todo_pb";
import {getApiBaseUrl} from "$lib/url";

const API_BASE_URL = getApiBaseUrl();

const transport = createConnectTransport({
    baseUrl: API_BASE_URL,
});

export const Greet = createClient(GreetService, transport);
export const Todo = createClient(TodoService, transport);
