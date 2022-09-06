package event

type Type string

const (
	NftMintedEvent              Type = "NftMintedEvent"
	ContractBaseUriUpdatedEvent Type = "ContractBaseUriUpdatedEvent"
	NftBaseUriUpdatedEvent      Type = "NftBaseUriUpdatedEvent"
	TokenUriUpdatedEvent        Type = "TokenUriUpdatedEvent"
	MetadataRefreshedEvent      Type = "MetadataRefreshedEvent"
)
