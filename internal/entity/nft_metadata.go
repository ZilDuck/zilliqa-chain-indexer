package entity

import (
	"encoding/json"
	"errors"
)

type Metadata struct {
	Uri        string                 `json:"uri"`
	Properties map[string]interface{} `json:"properties"`
	IsIpfs     bool                   `json:"ipfs"`

	Status   MetadataStatus `json:"status"`
	Error    string         `json:"error"`
	Attempts int            `json:"attempts"`
}

type MetadataStatus string
var (
	MetadataPending MetadataStatus = "pending"
	MetadataSuccess MetadataStatus = "success"
	MetadataFailure MetadataStatus = "failure"
)

type MetadataProperty struct {
	Key     string               `json:"key"`
	String  *string              `json:"string"`
	Bool    *bool                `json:"bool"`
	Long    *int64               `json:"long"`
	Double  *float64             `json:"double"`
	Object  MetadataProperties   `json:"object"`
	Objects []MetadataProperties `json:"objects"`
}

type MetadataProperties []MetadataProperty

func (m Metadata) UriEmpty() bool {
	return m.Uri == ""
}

func (m Metadata) GetAssetUri() (string, error) {
	if resources := m.GetProperty("resources"); resources != nil {
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

	if resource := m.GetProperty("resource"); resource != nil {
		return resource.(string), nil
	}

	if image := m.GetProperty("image"); image != nil {
		return image.(string), nil
	}

	return "", errors.New("asset uri not found")
}

func (m Metadata) GetProperty(key string) interface{} {
	for k, v := range m.Properties {
		if key == k {
			return v
		}
	}

	return nil
}
