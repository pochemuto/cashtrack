package cashtrack

import (
	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	"context"
	"fmt"
	"math/rand"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type GreetService struct{}

type Greeting struct {
	Text     string
	Language string
}

var greetings = []Greeting{
	{Text: "Hello", Language: "English"},
	{Text: "Ola", Language: "Portuguese"},
	{Text: "Привет", Language: "Russian"},
	{Text: "Bonjour", Language: "French"},
	{Text: "Hola", Language: "Spanish"},
	{Text: "Ciao", Language: "Italian"},
	{Text: "Guten Tag", Language: "German"},
	{Text: "Salam", Language: "Arabic"},
	{Text: "Shalom", Language: "Hebrew"},
	{Text: "Namaste", Language: "Hindi"},
	{Text: "Konnichiwa", Language: "Japanese"},
	{Text: "Nǐ hǎo", Language: "Chinese (Mandarin)"},
	{Text: "Annyeonghaseyo", Language: "Korean"},
	{Text: "Merhaba", Language: "Turkish"},
	{Text: "Sawubona", Language: "Zulu"},
	{Text: "Habari", Language: "Swahili"},
	{Text: "Sveiki", Language: "Latvian"},
	{Text: "Tere", Language: "Estonian"},
	{Text: "Ahoj", Language: "Czech/Slovak"},
	{Text: "Zdravo", Language: "Serbian/Croatian/Bosnian"},
	{Text: "Selam", Language: "Amharic/Tigrinya/Turkish informal"},
	{Text: "Hej", Language: "Swedish"},
	{Text: "Hei", Language: "Norwegian"},
	{Text: "God dag", Language: "Danish/Norwegian"},
	{Text: "Dzień dobry", Language: "Polish"},
	{Text: "Yassas", Language: "Greek"},
	{Text: "Szia", Language: "Hungarian"},
	{Text: "Halo", Language: "Indonesian"},
	{Text: "Selamat siang", Language: "Indonesian (daytime)"},
	{Text: "Xin chào", Language: "Vietnamese"},
	{Text: "Shwmae", Language: "Welsh"},
	{Text: "Dia dhuit", Language: "Irish"},
	{Text: "Olá", Language: "Portuguese"},
	{Text: "Salut", Language: "Romanian/French informal"},
	{Text: "Sawasdee", Language: "Thai"},
	{Text: "Marhaba", Language: "Arabic (Levant/Gulf)"},
	{Text: "Sain baina uu", Language: "Mongolian"},
}

type GreetHandler Handler

func NewGreetHandler() *GreetHandler {
	greeter := new(GreetService)

	path, handler := apiv1connect.NewGreetServiceHandler(
		greeter,
		// Validation via Protovalidate is almost always recommended
		connect.WithInterceptors(validate.NewInterceptor()),
	)
	return &GreetHandler{Path: path, Handler: handler}
}

func (s *GreetService) Greet(
	_ context.Context,
	req *apiv1.GreetRequest,
) (*apiv1.GreetResponse, error) {

	n := rand.Int() % len(greetings)
	res := &apiv1.GreetResponse{
		Greeting: fmt.Sprintf("%s, %s!", greetings[n].Text, req.Name),
		Language: greetings[n].Language,
	}

	return res, nil
}
