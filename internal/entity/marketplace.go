package entity

type Marketplace string

const (
	ZilkroadMarketplace Marketplace = "Zilkroad"
	OkimotoMarketplace  Marketplace = "Okimoto"
	ArkyMarketplace     Marketplace = "Arky"
	MintableMarketplace Marketplace = "Mintable"
)

const (
	ZilkroadPlatformFee uint = 0
	OkimotoPlatformFee uint = 0
	ArkyPlatformFee uint = 200
	MintablePlatformFee uint = 100
)

const (
	OkimotoMarketplaceAddress string = "0x8d329a47bf148c7d63d52b75fb2028adc10a3d2f"
)

type MarketplaceSale struct {
	Marketplace  Marketplace
	Tx           Transaction
	Nft          Nft
	Buyer        string
	Seller       string
	Cost         string
	Fee          string
	Royalty      string
	RoyaltyBps   string
	Fungible     string
}

type MarketplaceListing struct {
	Marketplace  Marketplace
	Tx           Transaction
	Nft          Nft
	Cost         string
	Fungible     string
}

type MarketplaceDelisting struct {
	Marketplace  Marketplace
	Tx           Transaction
	Nft          Nft
}