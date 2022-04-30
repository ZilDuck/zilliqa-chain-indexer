package event

type Type string

const (
	NftMintedEvent              Type = "NftMintedEvent"
	ContractBaseUriUpdatedEvent Type = "ContractBaseUriUpdatedEvent"
	TokenUriUpdatedEvent        Type = "TokenUriUpdatedEvent"
	MetadataRefreshedEvent      Type = "MetadataRefreshedEvent"
)
