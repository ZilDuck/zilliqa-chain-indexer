package event

type Type string

const (
	NftMintedEvent              Type = "NftMintedEvent"
	ContractIndexedEvent        Type = "ContractIndexedEvent"
	ContractBaseUriUpdatedEvent Type = "ContractBaseUriUpdatedEvent"
	TokenUriUpdatedEvent        Type = "TokenUriUpdatedEvent"
	MetadataRefreshedEvent      Type = "MetadataRefreshedEvent"
)
