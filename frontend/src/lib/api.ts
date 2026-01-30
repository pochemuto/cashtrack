import {createClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {AuthService} from "$lib/gen/api/v1/auth_pb";
import {CategoryService} from "$lib/gen/api/v1/categories_pb";
import {GreetService} from "$lib/gen/api/v1/greet_pb";
import {ReportService} from "$lib/gen/api/v1/reports_pb";
import {TodoService} from "$lib/gen/api/v1/todo_pb";
import {TransactionService} from "$lib/gen/api/v1/transactions_pb";
import {getApiBaseUrl} from "$lib/url";

const API_BASE_URL = getApiBaseUrl();

const transport = createConnectTransport({
    baseUrl: API_BASE_URL,
    fetch: (input, init) => fetch(input, {...init, credentials: "include"}),
});

export const Greet = createClient(GreetService, transport);
export const Todo = createClient(TodoService, transport);
export const Auth = createClient(AuthService, transport);
export const Categories = createClient(CategoryService, transport);
export const Reports = createClient(ReportService, transport);
export const Transactions = createClient(TransactionService, transport);
