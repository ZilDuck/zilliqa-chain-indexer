package entity

import (
	"errors"
)

type Metadata struct {
	Uri   string      `json:"uri"`
	Error string      `json:"error"`
	Data  interface{} `json:"data"`
	Ipfs  bool        `json:"ipfs"`
}

func (m Metadata) GetAssetUri() (string, error) {
	if resource := m.GetData("resource"); resource != nil {
		return resource.(string), nil
	}

	return "", errors.New("asset uri not found")
}

func (m Metadata) GetData(key string) interface{} {
	switch m.Data.(type) {
	case map[string]interface{}:
		data := m.Data.(map[string]interface{})
		if val, ok := data[key]; ok {
			return val
		}
	}

	return nil
}
