package cashtrack

import (
	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	"cashtrack/backend/gen/db"
	"context"
	"errors"
	"math/rand"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type TodoService struct {
	db *Db
}

var todos = []string{
	"Купить молоко",
	"Оплатить коммуналку",
	"Записаться к стоматологу",
	"Сделать бэкап телефона",
	"Выкинуть просрочку из холодильника",
	"Приготовить ужин на 2 дня",
	"Сходить в аптеку",
	"Заменить лампочку в коридоре",
	"Постирать и развесить бельё",
	"Сдать бутылки/стекло на переработку",
	"Проверить давление в шинах",
	"Заправить машину/пополнить проездной",
	"Поменять батарейки в пульте",
	"Разобрать почту (бумажную)",
	"Проверить банковские списания",
	"Составить список покупок",
	"Купить подарок заранее (на всякий случай)",
	"Позвонить родителям",
	"Написать другу, которому давно обещал",
	"Запланировать отпуск (хотя бы мечтательно)",
	"Проверить сроки паспортов/документов",
	"Записаться на техосмотр/сервис",
	"Сходить в спортзал",
	"Сделать растяжку 10 минут",
	"Прогуляться 30 минут без телефона",
	"Выпить воды (прямо сейчас)",
	"Убрать рабочий стол",
	"Разобрать папку “Загрузки”",
	"Удалить 50 скриншотов, которые «точно пригодятся»",
	"Почистить клавиатуру",
	"Протереть экран монитора",
	"Настроить автоплатёж",
	"Проверить подписки и отменить лишнее",
	"Разобрать шкаф: оставить только то, что радует",
	"Сдать одежду на благотворительность",
	"Погладить рубашку/платье",
	"Зашить/починить мелкую дырку",
	"Почистить обувь",
	"Подстричь ногти (всем участникам семьи)",
	"Записаться к парикмахеру",
	"Купить фильтр для воды/кофемашины",
	"Почистить кофемашину",
	"Разморозить морозилку (в теории)",
	"Почистить плиту",
	"Помыть окна (смело)",
	"Пропылесосить",
	"Помыть полы",
	"Помыть посуду сразу после еды (легенда)",
	"Вытереть пыль на полках",
	"Навести порядок в ванной",
	"Постирать полотенца",
	"Заменить постельное бельё",
	"Сделать генеральную уборку на балконе",
	"Полить растения",
	"Пересадить растение, которое держится на силе воли",
	"Купить корм/наполнитель для питомца",
	"Записать питомца к ветеринару на плановый осмотр",
	"Сделать прививку/проверить календарь прививок",
	"Проверить аптечку: что там вообще происходит",
	"Сделать список экстренных контактов",
	"Обновить резюме",
	"Ответить на 3 важных письма",
	"Закрыть вкладки в браузере (хотя бы 20)",
	"Разобрать закладки",
	"Настроить двухфакторку везде",
	"Сменить пароли (перестать использовать “qwerty123”)",
	"Почитать 20 страниц книги",
	"Послушать подкаст без параллельной прокрутки",
	"Выучить 10 новых слов",
	"Повторить основы первой помощи",
	"Составить меню на неделю",
	"Сделать заготовки: крупы/овощи",
	"Купить контейнеры (не очередные, а подходящие)",
	"Сварить суп, как взрослая версия себя",
	"Испечь что-то простое",
	"Сделать домашний лимонад/чай",
	"Проверить срок годности специй (это больно)",
	"Сделать фото документов и сохранить в облако",
	"Разобрать фотографии за месяц",
	"Сделать фотокнигу (когда-нибудь)",
	"Отправить открытку/сообщение благодарности",
	"Сказать «нет» одной лишней задаче",
	"Сделать паузу 5 минут и просто подышать",
	"Лечь спать пораньше",
	"Поставить будильник без второй сотни повторов",
	"Сделать план на завтра из 3 пунктов",
	"Довести до конца одну маленькую задачу",
	"Помыть кружку, которая «ещё норм»",
	"Не покупать очередной блокнот для списка дел",
	"Сделать вид, что “инбокс зеро” реален",
	"Проверить, где лежит зарядка (она опять ушла)",
	"Найти второй носок (это квест)",
	"Перестать спорить с микроволновкой",
	"Переименовать файлы “final_final2(1).docx”",
	"Сделать “финал” действительно финалом",
	"Сохранить работу, пока не поздно",
	"Проверить, выключен ли утюг (даже если его нет)",
	"Вынести мусор",
	"Сдать батарейки на утилизацию",
	"Купить что-то для души в пределах бюджета",
	"Проверить бюджет и не плакать",
	"Составить список целей на месяц",
	"Отметить маленькую победу",
	"Улыбнуться своему отражению (не обязательно искренне)",
}

type TodoHandler Handler

func NewTodoHandler(db *Db) *TodoHandler {
	todo := &TodoService{
		db: db,
	}

	path, handler := apiv1connect.NewTodoServiceHandler(
		todo,
		// Validation via Protovalidate is almost always recommended
		connect.WithInterceptors(validate.NewInterceptor(), NewAuthInterceptor(db)),
	)
	return &TodoHandler{Path: path, Handler: handler}
}

func requireUser(ctx context.Context) (AuthUser, error) {
	user, ok := userFromContext(ctx)
	if !ok {
		return AuthUser{}, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	return user, nil
}

func (s *TodoService) listTodos(ctx context.Context, userID int32) ([]*apiv1.ListItem, error) {
	todos, err := s.db.Queries.ListTodosByUser(ctx, userID)
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
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.listTodos(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	res.Items = items

	return res, nil
}

func (s *TodoService) AddRandom(ctx context.Context, req *apiv1.AddRandomRequest) (*apiv1.AddRandomResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	indices := rand.Perm(len(todos))[:10]
	randomTodos := make([]string, len(indices))
	for i, idx := range indices {
		randomTodos[i] = todos[idx]
	}
	if err := s.db.Queries.AddTodosBatch(ctx, db.AddTodosBatchParams{
		Titles: randomTodos,
		UserID: user.ID,
	}); err != nil {
		return nil, err
	}
	res := &apiv1.AddRandomResponse{}
	items, err := s.listTodos(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	res.Items = items
	return res, nil

}

func (s *TodoService) Add(ctx context.Context, req *apiv1.AddRequest) (*apiv1.AddResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range req.Items {
		err := s.db.Queries.AddTodo(ctx, db.AddTodoParams{
			Title:  item.Title,
			UserID: user.ID,
		})
		if err != nil {
			return nil, err
		}
	}
	res := &apiv1.AddResponse{}
	items, err := s.listTodos(ctx, user.ID)
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

	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	err = s.db.Queries.RemoveTodo(ctx, db.RemoveTodoParams{
		ID:     req.Id,
		UserID: user.ID,
	})
	if err != nil {
		return nil, err
	}
	res := &apiv1.RemoveResponse{}
	items, err := s.listTodos(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	res.Items = items
	return res, nil
}
