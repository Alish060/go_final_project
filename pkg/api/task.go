package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"scheduler/pkg/db"
)

type TasksResp struct {
	Tasks []*db.Task `json:"tasks"`
}

func tasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := db.Tasks(50)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	writeJSON(w, TasksResp{Tasks: tasks}, http.StatusOK)
}
func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		addTaskHandler(w, r)
	case http.MethodGet:
		getTaskHandler(w, r)
	case http.MethodPut:
		updateTaskHandler(w, r)
	case http.MethodDelete:
		deleteTaskHandler(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Неподдерживаемый метод",
		})
	}
}

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func afterNow(now, t time.Time) bool {
	return t.After(now)
}

func checkDate(task *db.Task) error {
	now := time.Now()

	if strings.TrimSpace(task.Date) == "" {
		task.Date = now.Format(DateFormat)
		return nil
	}

	if strings.TrimSpace(task.Date) == "today" {
		task.Date = now.Format(DateFormat)
		return nil
	}

	t, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		return fmt.Errorf("некорректная дата: %w", err)
	}

	if t.After(now) {
		return nil
	}

	if t.Format(DateFormat) == now.Format(DateFormat) {
		return nil
	}

	if strings.TrimSpace(task.Repeat) != "" {
		next, err := NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return fmt.Errorf("некорректный repeat: %w", err)
		}
		task.Date = next
		return nil
	}

	task.Date = now.Format(DateFormat)
	return nil
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Ошибка при чтении"}, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var task db.Task
	if err := json.Unmarshal(body, &task); err != nil {
		writeJSON(w, map[string]string{"error": "Невалидный json"}, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(task.Title) == "" {
		writeJSON(w, map[string]string{"error": "Заголовок обязателен"}, http.StatusBadRequest)
		return
	}

	if err := checkDate(&task); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return
	}

	id, err := db.AddTask(&task)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"id": fmt.Sprintf("%d", id)}, http.StatusOK)
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, map[string]string{"error": "Не указан идентификатор"}, http.StatusBadRequest)
		return
	}

	task, err := db.GetTask(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Задача не найдена"}, http.StatusNotFound)
		return
	}

	writeJSON(w, task, http.StatusOK)
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task db.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJSON(w, map[string]string{"error": "Некорректные данные"}, http.StatusBadRequest)
		return
	}

	if task.ID == "" {
		writeJSON(w, map[string]string{"error": "Не указан идентификатор"}, http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		writeJSON(w, map[string]string{"error": "Не указано название задачи"}, http.StatusBadRequest)
		return
	}

	if task.Date == "today" {
		task.Date = time.Now().Format(DateFormat)
	} else if task.Date != "" {
		if _, err := time.Parse(DateFormat, task.Date); err != nil {
			writeJSON(w, map[string]string{"error": fmt.Sprintf("Некорректная дата: %v", err)}, http.StatusBadRequest)
			return
		}
	}

	if task.Repeat != "" {
		if _, err := NextDate(time.Now(), task.Date, task.Repeat); err != nil {
			writeJSON(w, map[string]string{"error": fmt.Sprintf("Некорректный repeat: %v", err)}, http.StatusBadRequest)
			return
		}
	}

	if err := db.UpdateTask(&task); err != nil {
		writeJSON(w, map[string]string{"error": "Задача не найдена"}, http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{}, http.StatusOK)
}

func taskDoneHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, map[string]string{"error": "Не указан идентификатор"}, http.StatusBadRequest)
		return
	}

	task, err := db.GetTask(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Задача не найдена"}, http.StatusNotFound)
		return
	}

	if strings.TrimSpace(task.Repeat) == "" {
		if err := db.DeleteTask(id); err != nil {
			writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		writeJSON(w, struct{}{}, http.StatusOK)
		return
	}

	now := time.Now()
	next, err := NextDate(now, task.Date, task.Repeat)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Некорректный repeat"}, http.StatusBadRequest)
		return
	}

	if err := db.UpdateDate(next, id); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{}, http.StatusOK)
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, map[string]string{"error": "Не указан идентификатор"}, http.StatusBadRequest)
		return
	}

	err := db.DeleteTask(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{}, http.StatusOK)
}
