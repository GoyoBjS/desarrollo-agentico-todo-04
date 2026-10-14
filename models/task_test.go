package models

import "strings"

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) createTasks(titles ...string) []*Task {
	tasks := make([]*Task, 0, len(titles))
	for _, title := range titles {
		task := &Task{Title: title}
		verrs, err := ms.DB.ValidateAndCreate(task)
		ms.NoError(err)
		ms.False(verrs.HasAny())
		tasks = append(tasks, task)
	}
	return tasks
}

func (ms *ModelSuite) assertOrder(titles ...string) {
	tasks := Tasks{}
	ms.NoError(ms.DB.Order(TaskOrder).All(&tasks))
	ms.Len(tasks, len(titles))
	for i, task := range tasks {
		ms.Equal(titles[i], task.Title)
		ms.Equal(i+1, task.Position)
	}
}

func (ms *ModelSuite) Test_Task_Create_AppendsAtEnd() {
	tasks := ms.createTasks("A", "B", "C")
	ms.Equal(1, tasks[0].Position)
	ms.Equal(2, tasks[1].Position)
	ms.Equal(3, tasks[2].Position)
	ms.assertOrder("A", "B", "C")
}

func (ms *ModelSuite) Test_MoveTask_ToFirst() {
	tasks := ms.createTasks("A", "B", "C")
	ms.NoError(MoveTask(ms.DB, tasks[2], 1))
	ms.Equal(1, tasks[2].Position)
	ms.assertOrder("C", "A", "B")
}

func (ms *ModelSuite) Test_MoveTask_ToLast() {
	tasks := ms.createTasks("A", "B", "C")
	ms.NoError(MoveTask(ms.DB, tasks[0], 3))
	ms.Equal(3, tasks[0].Position)
	ms.assertOrder("B", "C", "A")
}

func (ms *ModelSuite) Test_MoveTask_ClampsOutOfRange() {
	tasks := ms.createTasks("A", "B", "C")

	ms.NoError(MoveTask(ms.DB, tasks[1], 0))
	ms.Equal(1, tasks[1].Position)
	ms.assertOrder("B", "A", "C")

	ms.NoError(MoveTask(ms.DB, tasks[1], 99))
	ms.Equal(3, tasks[1].Position)
	ms.assertOrder("A", "C", "B")
}

func (ms *ModelSuite) Test_MoveTask_FillsGaps() {
	created := []*Task{}
	for i, title := range []string{"A", "B", "C"} {
		position := []int{5, 10, 20}[i]
		task := &Task{Title: title, Position: position}
		verrs, err := ms.DB.ValidateAndCreate(task)
		ms.NoError(err)
		ms.False(verrs.HasAny())
		ms.Equal(position, task.Position)
		created = append(created, task)
	}

	ms.NoError(MoveTask(ms.DB, created[1], 1))
	ms.assertOrder("B", "A", "C")
}

func (ms *ModelSuite) Test_Task_Create_AfterMove_AppendsAtEnd() {
	tasks := ms.createTasks("A", "B", "C")
	ms.NoError(MoveTask(ms.DB, tasks[2], 1))

	d := ms.createTasks("D")[0]
	ms.Equal(4, d.Position)
	ms.assertOrder("C", "A", "B", "D")
}
