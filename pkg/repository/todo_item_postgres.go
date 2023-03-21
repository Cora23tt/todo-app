package repository

import (
	"fmt"
	"strings"

	"github.com/cora23tt/todo-app"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type TodoItemPostgres struct {
	db *sqlx.DB
}

func NewTodoItemPostgres(db *sqlx.DB) *TodoItemPostgres {
	return &TodoItemPostgres{db: db}
}

func (s *TodoItemPostgres) Create(listId int, item todo.TodoItem) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, nil
	}

	var itemId int
	createItemQuery := fmt.Sprintf(`
	INSERT INTO %s (title, description)
	VALUES ($1, $2)
	RETURNING id
	`, todoItemsTable)

	row := tx.QueryRow(createItemQuery, item.Title, item.Description)
	if err := row.Scan(&itemId); err != nil {
		tx.Rollback()
		return 0, err
	}

	createListItemsQuery := fmt.Sprintf(`
	INSERT INTO %s (list_id, item_id)
	VALUES ($1, $2)
	`, listsItemsTable)

	if _, err := tx.Exec(createListItemsQuery, listId, itemId); err != nil {
		tx.Rollback()
		return 0, err
	}

	return itemId, tx.Commit()
}

func (s *TodoItemPostgres) GetAll(userId, listId int) ([]todo.TodoItem, error) {
	var items []todo.TodoItem
	query := fmt.Sprintf(`
	SELECT ti.id, ti.title, ti.description, ti.done
	FROM %s ti
		INNER JOIN %s li ON li.item_id = ti.id
		INNER JOIN %s ul ON ul.list_id = li.list_id
	WHERE ul.user_id=$1 AND ul.list_id=$2
	`, todoItemsTable, listsItemsTable, usersListTable)
	if err := s.db.Select(&items, query, userId, listId); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *TodoItemPostgres) GetById(userId, itemId int) (item todo.TodoItem, err error) {

	query := fmt.Sprintf(`
	SELECT ti.id, ti.title, ti.description, ti.done 
	FROM %s ti 
	INNER JOIN %s li ON 
	ti.id = li.item_id 
	INNER JOIN %s ul ON 
	li.list_id = ul.list_id 
	WHERE ul.user_id=$1 AND ti.id=$2`,
		todoItemsTable, listsItemsTable, usersListTable)

	err = s.db.Get(&item, query, userId, itemId)
	return item, err
}

func (s *TodoItemPostgres) Update(userId, itemId int, input todo.UpdateItemInput) error {
	setValues := make([]string, 0)
	args := make([]interface{}, 0)
	argId := 1

	if input.Title != nil {
		setValues = append(setValues, fmt.Sprintf("title=$%d", argId))
		args = append(args, *input.Title)
		argId++
	}
	if input.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description=$%d", argId))
		args = append(args, *input.Description)
		argId++
	}
	if input.Done != nil {
		setValues = append(setValues, fmt.Sprintf("done=$%d", argId))
		args = append(args, *input.Done)
		argId++
	}

	setQuery := strings.Join(setValues, ", ")

	query := fmt.Sprintf(`UPDATE %s ti SET %s FROM %s li, %s ul
								WHERE ti.id = li.item_id AND li.list_id = ul.list_id AND ul.user_id = $%d AND ti.id = $%d `,
		todoItemsTable, setQuery, listsItemsTable, usersListTable, argId, argId+1)
	args = append(args, userId, itemId)
	logrus.Debugf("updateQuery: %s", query)
	logrus.Debugf("args %s", args)

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *TodoItemPostgres) Delete(userId, itemId int) error {
	query := fmt.Sprintf(`
	DELETE FROM %s ti 
	USING %s li, %s ul 
	WHERE ti.id = li.item_id 
	AND li.list_id = ul.list_id 
	AND ul.user_id = $1 
	AND ti.id = $2
	`, todoItemsTable, listsItemsTable, usersListTable)

	_, err := s.db.Exec(query, userId, itemId)

	return err
}
