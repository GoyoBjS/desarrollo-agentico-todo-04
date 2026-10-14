package actions

import (
	"net/url"
	"strings"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) createTasks(titles ...string) []*models.Task {
	tasks := make([]*models.Task, 0, len(titles))
	for _, title := range titles {
		task := &models.Task{Title: title}
		verrs, err := as.DB.ValidateAndCreate(task)
		as.NoError(err)
		as.False(verrs.HasAny())
		tasks = append(tasks, task)
	}
	return tasks
}

func (as *ActionSuite) orderedTitles() []string {
	tasks := models.Tasks{}
	as.NoError(as.DB.Order(models.TaskOrder).All(&tasks))
	titles := make([]string, 0, len(tasks))
	for _, task := range tasks {
		titles = append(titles, task.Title)
	}
	return titles
}

func (as *ActionSuite) moveTask(task *models.Task, position string) int {
	res := as.HTML("/tasks/%s/position", task.ID).Put(url.Values{"position": {position}})
	return res.Code
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPosition() {
	for _, task := range []*models.Task{
		{Title: "Tarea en segundo lugar", Position: 2},
		{Title: "Tarea en primer lugar", Position: 1},
	} {
		verrs, err := as.DB.ValidateAndCreate(task)
		as.NoError(err)
		as.False(verrs.HasAny())
	}

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	first := strings.Index(body, "Tarea en primer lugar")
	second := strings.Index(body, "Tarea en segundo lugar")
	as.NotEqual(-1, first)
	as.NotEqual(-1, second)
	as.Less(first, second)
}

func (as *ActionSuite) Test_TasksIndex_RendersMoveControls() {
	task := as.createTasks("Tarea movible")[0]

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Contains(body, "Mover tarea: Tarea movible")
	as.Contains(body, `action="/tasks/`+task.ID.String()+`/position"`)
	as.Contains(body, "data-task-handle")
}

func (as *ActionSuite) Test_TasksMove_PersistsOrder() {
	tasks := as.createTasks("Tarea A", "Tarea B", "Tarea C")

	as.Equal(204, as.moveTask(tasks[2], "1"))
	as.Equal([]string{"Tarea C", "Tarea A", "Tarea B"}, as.orderedTitles())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Less(strings.Index(body, "Tarea C"), strings.Index(body, "Tarea A"))
}

func (as *ActionSuite) Test_TasksMove_CompletedTask() {
	as.createTasks("Tarea A", "Tarea B")
	done := &models.Task{Title: "Tarea completada", Completed: true}
	verrs, err := as.DB.ValidateAndCreate(done)
	as.NoError(err)
	as.False(verrs.HasAny())

	as.Equal(204, as.moveTask(done, "1"))
	as.Equal([]string{"Tarea completada", "Tarea A", "Tarea B"}, as.orderedTitles())

	as.NoError(as.DB.Reload(done))
	as.True(done.Completed)
	as.Equal(1, done.Position)
}

func (as *ActionSuite) Test_TasksMove_InvalidPosition() {
	tasks := as.createTasks("Tarea A", "Tarea B", "Tarea C")

	as.Equal(422, as.moveTask(tasks[2], "abc"))
	as.Equal([]string{"Tarea A", "Tarea B", "Tarea C"}, as.orderedTitles())
}

func (as *ActionSuite) Test_TasksMove_NotFound() {
	res := as.HTML("/tasks/%s/position", uuid.Must(uuid.NewV4())).Put(url.Values{"position": {"1"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksMove_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/position").Put(url.Values{"position": {"1"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksCreate_AppendsAtEndAfterMove() {
	tasks := as.createTasks("Tarea A", "Tarea B", "Tarea C")
	as.Equal(204, as.moveTask(tasks[2], "1"))

	res := as.HTML("/tasks").Post(&models.Task{Title: "Tarea D"})
	as.Equal(303, res.Code)
	as.Equal([]string{"Tarea C", "Tarea A", "Tarea B", "Tarea D"}, as.orderedTitles())
}

func (as *ActionSuite) Test_TasksToggle_KeepsPosition() {
	tasks := as.createTasks("Tarea A", "Tarea B", "Tarea C")
	as.Equal(204, as.moveTask(tasks[2], "1"))

	res := as.HTML("/tasks/%s/toggle", tasks[2].ID).Put(url.Values{})
	as.Equal(303, res.Code)
	res = as.HTML("/tasks/%s/toggle", tasks[2].ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(tasks[2]))
	as.Equal(1, tasks[2].Position)
	as.Equal([]string{"Tarea C", "Tarea A", "Tarea B"}, as.orderedTitles())
}

func (as *ActionSuite) Test_TasksUpdate_KeepsPosition() {
	tasks := as.createTasks("Tarea A", "Tarea B")
	as.Equal(204, as.moveTask(tasks[1], "1"))

	res := as.HTML("/tasks/%s", tasks[1].ID).Put(url.Values{"title": {"Tarea B editada"}})
	as.Equal(303, res.Code)

	as.Equal([]string{"Tarea B editada", "Tarea A"}, as.orderedTitles())
}
