package entity

import (
	"errors"
)

type Metadata struct {
	Uri  string      `json:"uri"`
	Data interface{} `json:"data"`
	Ipfs bool        `json:"ipfs"`

	Error     string `json:"error"`
	Attempted int    `json:"attempted"`

	AssetError     string `json:"assetError"`
	AssetAttempted int    `json:"assetAttempted"`
}

func (m Metadata) UriEmpty() bool {
	return m.Uri == ""
}

func (m Metadata) GetAssetUri() (string, error) {
	if resource := m.GetData("resource"); resource != nil {
		return resource.(string), nil
	}

	if image := m.GetData("image"); image != nil {
		return image.(string), nil
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
