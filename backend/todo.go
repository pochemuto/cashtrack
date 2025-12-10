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
	items   []*apiv1.ListItem
	nextID  int32
}

func NewTodoHandler() *Handler {
	todo := new(TodoService)

	path, handler := apiv1connect.NewTodoServiceHandler(
		todo,
		// Validation via Protovalidate is almost always recommended
		connect.WithInterceptors(validate.NewInterceptor()),
	)
	todo.initial = []string{"Buy milk", "Buy eggs", "Buy bread", "Do laundry", "Do homework", "Go to the gym"}
	todo.Regenerate()
	return &Handler{Path: path, Handler: handler}
}

func (s *TodoService) Regenerate() {
	for _, item := range s.initial {
		s.items = append(s.items, &apiv1.ListItem{Id: s.nextID, Title: item})
		s.nextID++
	}
}

func (s *TodoService) List(
	_ context.Context,
	req *apiv1.ListRequest,
) (*apiv1.ListResponse, error) {

	res := &apiv1.ListResponse{}
	res.Items = append(res.Items, s.items...)

	return res, nil
}

func (s *TodoService) Add(_ context.Context, req *apiv1.AddRequest) (*apiv1.AddResponse, error) {
	for _, item := range req.Items {
		item.Id = s.nextID
		s.nextID++
	}
	s.items = append(s.items, req.Items...)
	res := &apiv1.AddResponse{}
	res.Items = append(res.Items, s.items...)
	return res, nil
}

func (s *TodoService) Remove(
	_ context.Context,
	req *apiv1.RemoveRequest,
) (*apiv1.RemoveResponse, error) {
	itemIndex := -1
	for i, item := range s.items {
		if item.Id == req.Id {
			itemIndex = i
			break
		}
	}

	if itemIndex == -1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("item not found"))
	}

	s.items = append(s.items[:itemIndex], s.items[itemIndex+1:]...)
	res := &apiv1.RemoveResponse{
		Items: s.items,
	}

	if len(s.items) == 0 {
		s.Regenerate()
	}

	return res, nil
}
