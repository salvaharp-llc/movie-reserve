package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func parseNullGeneric[T any](s string, parse func(string) (T, error)) (T, bool, error) {
	var zero T
	if s == "" {
		return zero, false, nil
	}
	v, err := parse(s)
	if err != nil {
		return zero, false, err
	}
	return v, true, nil
}

func parseNullUUID(s string) (uuid.NullUUID, error) {
	id, valid, err := parseNullGeneric(s, uuid.Parse)
	if err != nil {
		return uuid.NullUUID{}, err
	}
	if !valid {
		return uuid.NullUUID{}, nil
	}
	return uuid.NullUUID{UUID: id, Valid: true}, nil
}

func parseNullTime(s string) (sql.NullTime, error) {
	t, valid, err := parseNullGeneric(s, func(str string) (time.Time, error) {
		return time.Parse(time.RFC3339, str)
	})
	if err != nil {
		return sql.NullTime{}, err
	}
	if !valid {
		return sql.NullTime{}, nil
	}
	return sql.NullTime{Time: t, Valid: true}, nil
}

func parseNullInt32(s string) (sql.NullInt32, error) {
	i, valid, err := parseNullGeneric(s, func(str string) (int32, error) {
		v, err := strconv.Atoi(str)
		return int32(v), err
	})
	if err != nil {
		return sql.NullInt32{}, err
	}
	if !valid {
		return sql.NullInt32{}, nil
	}
	if i < 0 {
		return sql.NullInt32{}, fmt.Errorf("value must be non-negative")
	}
	return sql.NullInt32{Int32: i, Valid: true}, nil
}

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		str := ns.String
		return &str
	}
	return nil
}

func nullInt32ToPtr(ni sql.NullInt32) *int32 {
	if ni.Valid {
		v := ni.Int32
		return &v
	}
	return nil
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		t := nt.Time
		return &t
	}
	return nil
}
