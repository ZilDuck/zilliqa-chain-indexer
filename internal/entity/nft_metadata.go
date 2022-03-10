package entity

import (
	"encoding/json"
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
	if resources := m.GetData("resources"); resources != nil {
		resourcesJson, err := json.Marshal(resources)
		if err != nil {
			return "", err
		}

		var resourcesMap []map[string]string
		err = json.Unmarshal(resourcesJson, &resourcesMap)
		if err != nil {
			return "", err
		}

		for _, resource := range resourcesMap {
			if _, ok := resource["uri"]; ok {
				return resource["uri"], nil
			}
		}
	}

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
