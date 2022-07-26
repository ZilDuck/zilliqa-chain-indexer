package entity

type ContractMetadata map[string]interface{}

func (c ContractMetadata) Slug() string {
	return CreateContractSlug(c["contract"].(string))
}
