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

func (tx Transaction) GetTransition(transition TRANSITION) (transitions []Transition) {
	for _, t := range tx.Receipt.Transitions {
		if t.Msg.Tag == string(transition) {
			transitions = append(transitions, t)
		}
	}
	return transitions
}

func (tx Transaction) HasTransition(transition TRANSITION) bool {
	for _, t := range tx.Receipt.Transitions {
		if t.Msg.Tag == string(transition) {
			return true
		}
	}
	return false
}

func (tx Transaction) GetZrc1Transitions() []Transition {
	var transitions []Transition
	for _, t := range tx.Receipt.Transitions {
		for _, zrc1Transition := range Zrc1Transitions {
			if t.Msg.Tag == string(zrc1Transition) {
				transitions = append(transitions, t)
			}
		}
	}

	return transitions
}

func (tx Transaction) GetZrc6Transitions() []Transition {
	var transitions []Transition
	for _, t := range tx.Receipt.Transitions {
		for _, zrc6Transition := range Zrc6Transitions {
			if t.Msg.Tag == string(zrc6Transition) {
				transitions = append(transitions, t)
			}
		}
	}

	return transitions
}

func (tx Transaction) IsMint() bool {
	return tx.HasEventLog("MintSuccess") || tx.HasTransition("Mint")
}

func (tx Transaction) IsTransfer() bool {
	return tx.HasTransition("TransferFrom") &&
		tx.GetTransition("TransferFrom")[0].Msg.Params.HasParam("token_id", "Uint256")
}
