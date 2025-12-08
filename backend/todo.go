package cashtrack

import (
	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	"context"
	"fmt"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type TodoService struct {
	initial []string
	items   []string
}

func NewTodoHandler() *Handler {
	todo := new(TodoService)

	path, handler := apiv1connect.NewTodoServiceHandler(
		todo,
		// Validation via Protovalidate is almost always recommended
		connect.WithInterceptors(validate.NewInterceptor()),
	)
	todo.items = []string{"Buy milk", "Buy eggs", "Buy bread", "Do laundry", "Do homework", "Go to the gym"}
	todo.initial = append([]string{}, todo.items...)
	return &Handler{Path: path, Handler: handler}
}

func (s *TodoService) List(
	_ context.Context,
	req *apiv1.ListRequest,
) (*apiv1.ListResponse, error) {

	res := &apiv1.ListResponse{}
	res.Item = s.items

	return res, nil
}

func (s *TodoService) Remove(
	_ context.Context,
	req *apiv1.RemoveRequest,
) (*apiv1.RemoveResponse, error) {
	itemIndex := -1
	for i, item := range s.items {
		if item == req.Item {
			itemIndex = i
			break
		}
	}

	if itemIndex == -1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("item not found"))
	}

	s.items = append(s.items[:itemIndex], s.items[itemIndex+1:]...)
	res := &apiv1.RemoveResponse{
		Item: s.items,
	}
	res.Item = s.items

	if len(s.items) == 0 {
		s.items = append([]string{}, s.initial...)
	}

	return res, nil
}
