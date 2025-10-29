package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DateFormat = `20060102`

func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	if strings.TrimSpace(repeat) == "" {
		return "", errors.New("repeat rule is empty")
	}

	start, err := time.Parse(DateFormat, dstart)
	if err != nil {
		return "", fmt.Errorf("invalid start date: %w", err)
	}

	repeat = strings.TrimSpace(repeat)

	var next time.Time

	switch {
	case repeat == "y":
		if start.After(now) {
			next = start.AddDate(1, 0, 0)
		} else {
			next = start
			for !next.After(now) {
				next = next.AddDate(1, 0, 0)
			}
		}

	case strings.HasPrefix(repeat, "d "):
		parts := strings.Split(repeat, " ")
		if len(parts) != 2 {
			return "", errors.New("invalid repeat format: missing day count")
		}

		days, err := strconv.Atoi(parts[1])
		if err != nil || days <= 0 {
			return "", errors.New("invalid day interval")
		}
		if days > 400 {
			return "", errors.New("day interval exceeds 400")
		}

		if start.After(now) {
			next = start.AddDate(0, 0, days)
		} else {
			next = start
			for !next.After(now) {
				next = next.AddDate(0, 0, days)
			}
		}

	default:
		return "", errors.New("invalid repeat format")
	}

	return next.Format(DateFormat), nil
}

func nextDayHandler(w http.ResponseWriter, r *http.Request) {
	// Чтение параметров
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	if dateStr == "" || repeat == "" {
		http.Error(w, "missing required parameters: date or repeat", http.StatusBadRequest)
		return
	}

	var now time.Time
	var err error

	if nowStr == "" {
		now = time.Now()
	} else {
		now, err = time.Parse(DateFormat, nowStr)
		if err != nil {
			http.Error(w, "invalid now date format", http.StatusBadRequest)
			return
		}
	}

	result, err := NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, result)
}
