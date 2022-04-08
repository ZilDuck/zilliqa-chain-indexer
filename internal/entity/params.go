package entity

import (
	"errors"
	"fmt"
	"strconv"
)

type Params []Param

var (
	ErrParamNotFound = errors.New("param not found")
)

type Param struct {
	Type  string `json:"type"`
	Value *Value `json:"value,omitempty"`
	VName string `json:"vname"`
}

type Value struct {
	Primitive interface{} `json:"primitive,omitempty"`

	ArgTypes    interface{} `json:"argtypes,omitempty"`
	Arguments   []*Value    `json:"arguments,omitempty"`
	Constructor string      `json:"constructor,omitempty"`
}

func (v Value) String() string {
	if v.Primitive != nil {
		return v.Primitive.(string)
	}

	return ""
}

func (v Value) Uint64() (uint64, error) {
	if v.String() == "" {
		return 0, errors.New("value not found")
	}

	value, err := strconv.ParseUint(v.String(), 10, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (p Params) GetParam(vName string) (Param, error) {
	for _, param := range p {
		if param.VName == vName {
			return param, nil
		}
	}
	return Param{}, errors.New(fmt.Sprintf("%s param not found", vName))
}

func (p Params) HasParam(vName string, paramType string) bool {
	param, err := p.GetParam(vName)
	if err != nil {
		return false
	}
	return param.Type == paramType
}
