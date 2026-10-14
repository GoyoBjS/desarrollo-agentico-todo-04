package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Position  int       `json:"position" db:"position" form:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// TaskOrder es el orden en el que se muestra la lista de tareas.
const TaskOrder = "position asc, created_at asc, id asc"

// BeforeCreate coloca la tarea nueva al final de la lista, salvo que ya tenga posición.
func (t *Task) BeforeCreate(tx *pop.Connection) error {
	if t.Position > 0 {
		return nil
	}

	last := Tasks{}
	if err := tx.Order("position desc").Limit(1).All(&last); err != nil {
		return err
	}

	t.Position = 1
	if len(last) > 0 {
		t.Position = last[0].Position + 1
	}
	return nil
}

// MoveTask mueve la tarea a la posición indicada (1 = primera) y renumera la lista sin huecos.
// Una posición fuera de rango se acota al principio o al final de la lista.
func MoveTask(tx *pop.Connection, task *Task, position int) error {
	if err := tx.RawQuery("SELECT id FROM tasks FOR UPDATE").Exec(); err != nil {
		return err
	}

	tasks := Tasks{}
	if err := tx.Order(TaskOrder).All(&tasks); err != nil {
		return err
	}

	others := make(Tasks, 0, len(tasks))
	var moved *Task
	for i := range tasks {
		if tasks[i].ID == task.ID {
			moved = &tasks[i]
			continue
		}
		others = append(others, tasks[i])
	}
	if moved == nil {
		return sql.ErrNoRows
	}

	if position < 1 {
		position = 1
	}
	if position > len(tasks) {
		position = len(tasks)
	}

	ordered := make(Tasks, 0, len(tasks))
	ordered = append(ordered, others[:position-1]...)
	ordered = append(ordered, *moved)
	ordered = append(ordered, others[position-1:]...)

	for i, t := range ordered {
		if t.Position == i+1 {
			continue
		}
		if err := tx.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", i+1, t.ID).Exec(); err != nil {
			return err
		}
	}

	task.Position = position
	return nil
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	return verrs, nil
}
