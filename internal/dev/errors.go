package dev

import (
	"github.com/nu7hatch/gouuid"
	"time"
)

type Error struct {
	Time      time.Time              `json:"time"`
	Component string                 `json:"component"`
	Name      string                 `json:"name"`
	Error     string                 `json:"error"`
	Extra     map[string]interface{} `json:"extra"`
}

func (e Error) Slug() string {
	u, _ := uuid.NewV4()
	return u.String()
}

func NewError(component, name string, err error, extra map[string]interface{}) Error {
	return Error{
		Time:      time.Now(),
		Component: component,
		Name:      name,
		Error:     err.Error(),
		Extra:     extra,
	}
}
