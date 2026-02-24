package main

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseNullUUID(t *testing.T) {
	good := uuid.New().String()
	tests := []struct {
		name    string
		input   string
		wantVal uuid.NullUUID
		wantErr bool
	}{
		{name: "empty", input: "", wantVal: uuid.NullUUID{}, wantErr: false},
		{name: "valid", input: good, wantVal: uuid.NullUUID{UUID: uuid.MustParse(good), Valid: true}, wantErr: false},
		{name: "invalid", input: "not-a-uuid", wantVal: uuid.NullUUID{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNullUUID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseNullUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantVal {
				t.Errorf("parseNullUUID() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func TestParseNullTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	str := now.Format(time.RFC3339)

	tests := []struct {
		name    string
		input   string
		wantVal sql.NullTime
		wantErr bool
	}{
		{name: "empty", input: "", wantVal: sql.NullTime{}, wantErr: false},
		{name: "valid", input: str, wantVal: sql.NullTime{Time: now, Valid: true}, wantErr: false},
		{name: "invalid", input: "not-a-time", wantVal: sql.NullTime{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNullTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseNullTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Valid != tt.wantVal.Valid {
				t.Errorf("parseNullTime() validity = %v, want %v", got.Valid, tt.wantVal.Valid)
				return
			}
			if got.Valid && !got.Time.Equal(tt.wantVal.Time) {
				t.Errorf("parseNullTime() time = %v, want %v", got.Time, tt.wantVal.Time)
			}
		})
	}
}

func TestParseNullInt32(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantVal sql.NullInt32
		wantErr bool
	}{
		{name: "empty", input: "", wantVal: sql.NullInt32{}, wantErr: false},
		{name: "zero", input: "0", wantVal: sql.NullInt32{Int32: 0, Valid: true}, wantErr: false},
		{name: "positive", input: "42", wantVal: sql.NullInt32{Int32: 42, Valid: true}, wantErr: false},
		{name: "negative", input: "-1", wantVal: sql.NullInt32{}, wantErr: true},
		{name: "invalid", input: "not-an-int", wantVal: sql.NullInt32{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNullInt32(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseNullInt32() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantVal {
				t.Errorf("parseNullInt32() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func TestNullConverters(t *testing.T) {
	str := "hello"
	i := int32(123)
	tm := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name    string
		input   interface{}
		wantNil bool
	}{
		{name: "string valid", input: sql.NullString{String: str, Valid: true}, wantNil: false},
		{name: "string invalid", input: sql.NullString{}, wantNil: true},
		{name: "int32 valid", input: sql.NullInt32{Int32: i, Valid: true}, wantNil: false},
		{name: "int32 invalid", input: sql.NullInt32{}, wantNil: true},
		{name: "time valid", input: sql.NullTime{Time: tm, Valid: true}, wantNil: false},
		{name: "time invalid", input: sql.NullTime{}, wantNil: true},
	}

	for _, tt := range tests {
		switch v := tt.input.(type) {
		case sql.NullString:
			p := nullStringToPtr(v)
			if (p == nil) != tt.wantNil {
				t.Errorf("nullStringToPtr(%v) nil? %v, want %v", v, p == nil, tt.wantNil)
			}
			if p != nil && *p != str {
				t.Errorf("nullStringToPtr returned %v, want %v", *p, str)
			}
		case sql.NullInt32:
			p := nullInt32ToPtr(v)
			if (p == nil) != tt.wantNil {
				t.Errorf("nullInt32ToPtr(%v) nil? %v, want %v", v, p == nil, tt.wantNil)
			}
			if p != nil && *p != i {
				t.Errorf("nullInt32ToPtr returned %v, want %v", *p, i)
			}
		case sql.NullTime:
			p := nullTimeToPtr(v)
			if (p == nil) != tt.wantNil {
				t.Errorf("nullTimeToPtr(%v) nil? %v, want %v", v, p == nil, tt.wantNil)
			}
			if p != nil && !p.Equal(tm) {
				t.Errorf("nullTimeToPtr returned %v, want %v", *p, tm)
			}
		default:
			t.Fatalf("unexpected type %T", tt.input)
		}
	}
}
