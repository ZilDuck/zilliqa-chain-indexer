package entity

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"github.com/gosimple/slug"
)

type Transaction struct {
	zilliqa.Transaction

	BlockNum uint64 `json:"BlockNum"`

	Code    string             `json:"Code,omitempty"`
	Data    Data               `json:"Data,omitempty"`
	Receipt TransactionReceipt `json:"Receipt"`

	IsContractCreation    bool   `json:"ContractCreation"`
	IsContractExecution   bool   `json:"ContractExecution"`
	ContractAddress       string `json:"ContractAddress,omitempty"`
	ContractAddressBech32 string `json:"ContractAddressBech32,omitempty"`

	SenderAddr   string `json:"SenderAddr"`
	SenderBech32 string `json:"SenderBech32"`
}

type Data struct {
	Tag    string `json:"_tag,omitempty"`
	Params Params `json:"params,omitempty"`
}

type TransactionReceipt struct {
	zilliqa.TransactionReceipt
	Transitions []Transition `json:"transitions"`
	EventLogs   []EventLog   `json:"event_logs"`
}

type EventLog struct {
	EventName string `json:"_eventname"`
	Address   string `json:"address"`
	Params    Params `json:"params"`
}

type Transition struct {
	zilliqa.Transition
	Msg TransitionMessage `json:"msg"`
}

type TransitionMessage struct {
	zilliqa.TransactionMessage
	Params Params `json:"params"`
}

func (tx Transaction) Slug() string {
	return CreateTransactionSlug(tx.Transaction.ID)
}

func CreateTransactionSlug(hash string) string {
	return slug.Make(fmt.Sprintf("tx-%s", hash))
}

func (tx Transaction) GetZrc1EventLogs() []EventLog {
	eventLogs := make([]EventLog, 0)
	for _, event := range tx.Receipt.EventLogs {
		if event.EventName == string(ZRC1MintEvent) ||
			event.EventName == string(ZRC1TransferEvent) ||
			event.EventName == string(ZRC1TransferFromEvent) ||
			event.EventName == string(ZRC1BurnEvent) {
			eventLogs = append(eventLogs, event)
		}
	}
	return eventLogs
}

func (tx Transaction) GetEventLogs(eventName Event) []EventLog {
	eventLogs := make([]EventLog, 0)
	for _, event := range tx.Receipt.EventLogs {
		if event.EventName == string(eventName) {
			eventLogs = append(eventLogs, event)
		}
	}
	return eventLogs
}

func (tx Transaction) GetEventLogForAddr(addr string, eventName Event) (EventLog, error) {
	for _, event := range tx.Receipt.EventLogs {
		if event.Address == addr && event.EventName == string(eventName) {
			return event, nil
		}
	}
	return EventLog{}, errors.New(fmt.Sprintf("Event %s for address %s does not exist", eventName, addr))
}

func (tx Transaction) HasEventLog(eventName Event) bool {
	for _, event := range tx.Receipt.EventLogs {
		if event.EventName == string(eventName) {
			return true
		}
	}
	return false
}

func (tx Transaction) GetTransition(transition string) (transitions []Transition) {
	for _, t := range tx.Receipt.Transitions {
		if t.Msg.Tag == transition {
			transitions = append(transitions, t)
		}
	}
	return transitions
}

func (tx Transaction) HasTransition(transition string) bool {
	for _, t := range tx.Receipt.Transitions {
		if t.Msg.Tag == transition {
			return true
		}
	}
	return false
}

func (tx Transaction) GetZrc1Transitions() []Transition {
	var transitions []Transition
	for _, t := range tx.Receipt.Transitions {
		for _, zrc1Callback := range Zrc1Callbacks {
			if t.Msg.Tag == string(zrc1Callback) {
				transitions = append(transitions, t)
			}
		}
	}

	return transitions
}

func (tx Transaction) GetZrc6Transitions() []Transition {
	var transitions []Transition
	for _, t := range tx.Receipt.Transitions {
		for _, zrc6Callback := range Zrc6Callbacks {
			if t.Msg.Tag == string(zrc6Callback) {
				transitions = append(transitions, t)
			}
		}
	}

	return transitions
}

func (tx Transaction) GetEngagedContracts() (addrs []string) {
	contractAddrs := map[string]interface{}{}
	for _, trans := range tx.Receipt.Transitions {
		contractAddrs[trans.Addr] = nil
	}

	for contractAddr := range contractAddrs {
		addrs = append(addrs, contractAddr)
	}

	return
}

func (tx Transaction) IsMarketplaceTx() bool {
	if tx.IsMarketplaceListing(ZilkroadMarketplace) || tx.IsMarketplaceListing(ArkyMarketplace) || tx.IsMarketplaceListing(OkimotoMarketplace) {
		return true
	}

	if tx.IsMarketplaceDelisting(ZilkroadMarketplace) || tx.IsMarketplaceDelisting(ArkyMarketplace) || tx.IsMarketplaceDelisting(OkimotoMarketplace) {
		return true
	}

	if tx.IsMarketplaceSale(ZilkroadMarketplace) || tx.IsMarketplaceSale(ArkyMarketplace) || tx.IsMarketplaceSale(OkimotoMarketplace) {
		return true
	}

	return false
}

func (tx Transaction) IsMarketplaceListing(marketplace Marketplace) bool {
	switch marketplace {
	case ZilkroadMarketplace:
		return tx.HasEventLog(MpZilkroadListingEvent)
	case ArkyMarketplace:
		return false
	case OkimotoMarketplace:
		if tx.HasEventLog(MpOkiListingEvent) {
			event := tx.GetEventLogs(MpOkiListingEvent)[0]

			recipient, err := event.Params.GetParam("recipient")
			if err != nil {
				return false
			}

			return recipient.Value.String() == OkimotoMarketplaceAddress
		}

		return false
	}
	return false
}

func (tx Transaction) IsMarketplaceDelisting(marketplace Marketplace) bool {
	switch marketplace {
	case ZilkroadMarketplace:
		return tx.HasEventLog(MpZilkroadDelistingEvent)
	case ArkyMarketplace:
		return false
	case OkimotoMarketplace:
		if tx.HasEventLog(MpOkiDelistingEvent) && tx.Data.Tag == "WithdrawalToken" {
			event := tx.GetEventLogs(MpOkiDelistingEvent)[0]

			from, err := event.Params.GetParam("from")
			if err != nil {
				return false
			}

			return from.Value.String() == OkimotoMarketplaceAddress
		}

		return false
	}
	return false
}

func (tx Transaction) IsMarketplaceSale(marketplace Marketplace) bool {
	switch marketplace {
	case ZilkroadMarketplace:
		return tx.HasEventLog(MpZilkroadSaleEvent)
	case ArkyMarketplace:
		return tx.HasEventLog(MpArkySaleEvent)
	case OkimotoMarketplace:
		if tx.HasEventLog(MpOkiSaleEvent) && tx.Data.Tag == "Buy" {
			event := tx.GetEventLogs(MpOkiSaleEvent)[0]

			from, err := event.Params.GetParam("from")
			if err != nil {
				return false
			}

			if from.Value.String() != OkimotoMarketplaceAddress {
				return false
			}

			return true
		}
	}
	return false

	return false
}