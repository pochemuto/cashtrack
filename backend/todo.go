package cashtrack

import (
	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	"cashtrack/backend/gen/db"
	"context"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type TodoService struct {
	db *Db
}

type TodoHandler Handler

func NewTodoHandler(db *Db) *TodoHandler {
	todo := &TodoService{
		db: db,
	}

	path, handler := apiv1connect.NewTodoServiceHandler(
		todo,
		// Validation via Protovalidate is almost always recommended
		connect.WithInterceptors(validate.NewInterceptor()),
	)
	return &TodoHandler{Path: path, Handler: handler}
}

func (s *TodoService) listTodos(ctx context.Context) ([]*apiv1.ListItem, error) {
	todos, err := s.db.Queries.ListTodos(ctx)
	if err != nil {
		return nil, err
	}
	r := make([]*apiv1.ListItem, len(todos))
	for i, todo := range todos {
		r[i] = &apiv1.ListItem{Id: todo.ID, Title: todo.Title}
	}
	return r, nil
}

func (s *TodoService) List(
	ctx context.Context,
	req *apiv1.ListRequest,
) (*apiv1.ListResponse, error) {

	res := &apiv1.ListResponse{}
	items, err := s.listTodos(ctx)
	if err != nil {
		return nil, err
	}
	res.Items = items

	return res, nil
}

func (s *TodoService) Add(ctx context.Context, req *apiv1.AddRequest) (*apiv1.AddResponse, error) {
	for _, item := range req.Items {
		err := s.db.Queries.AddTodo(ctx, db.AddTodoParams{
			Title: item.Title,
		})
		if err != nil {
			return nil, err
		}
	}
	res := &apiv1.AddResponse{}
	items, err := s.listTodos(ctx)
	if err != nil {
		return nil, err
	}
	res.Items = items
	return res, nil
}

func (s *TodoService) Remove(
	ctx context.Context,
	req *apiv1.RemoveRequest,
) (*apiv1.RemoveResponse, error) {

	err := s.db.Queries.RemoveTodo(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	res := &apiv1.RemoveResponse{}
	items, err := s.listTodos(ctx)
	if err != nil {
		return nil, err
	}
	res.Items = items
	return res, nil
}
