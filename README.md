# Zildexr

## Build dependencies
```shell
go run ./cmd/internal/injectDependencies/main.go ./generated
```

## Run locally
```shell
go run ./cmd/cli/main.go
go run ./cmd/assetServer/main.go
go run ./cmd/indexerd/main.go
go run ./cmd/metadata/main.go
```

## ZRC1 Support
- [x] Mint NFT
- [x] Transfer NFT
- [X] Burn NFT

## ZRC6 Support
- [x] Mint NFTs
- [x] Batch Mint NFTs
- [x] Update contract base uri
- [X] Burn NFT
- [X] Batch Burn NFT
- [X] Transfer From
- [ ] Batch Transfer From