package zil

import (
	"errors"
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

func (p Params) GetParam(vName string) (Param, error) {
	for _, param := range p {
		if param.VName == vName {
			return param, nil
		}
	}
	return Param{}, ErrParamNotFound
}

func (p Params) HasParam(vName string, paramType string) bool {
	param, err := p.GetParam(vName)
	if err != nil {
		return false
	}
	return param.Type == paramType
}
