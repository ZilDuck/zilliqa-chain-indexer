package transaction

import (
	"encoding/json"
	"fmt"
	"github.com/Zilliqa/gozilliqa-sdk/core"
	"github.com/Zilliqa/gozilliqa-sdk/keytools"
	"github.com/Zilliqa/gozilliqa-sdk/util"
	"github.com/dantudor/zil-indexer/internal/service/zilliqa"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"go.uber.org/zap"
	"log"
	"reflect"
	"strconv"
)

type Factory interface {
	CreateTransaction(coreTx core.Transaction, blockNum string) zil.Transaction
}

type factory struct {
	zilliqaService zilliqa.Service
}

func NewTransactionFactory(zilliqaService zilliqa.Service) Factory {
	return factory{zilliqaService}
}

func (f factory) CreateTransaction(coreTx core.Transaction, blockNum string) zil.Transaction {
	zap.L().With(zap.String("id", coreTx.ID)).Debug("Create Transaction")

	tx := zil.Transaction{
		Transaction:         coreTx,
		Code:                coreTx.Code,
		Data:                f.createParamsFromString(coreTx.Data),
		ContractAddress:     coreTx.ContractAddress,
		Receipt:             f.createReceipt(coreTx.Receipt),
		BlockNum:            f.stringToUint64(blockNum),
		IsContractCreation:  coreTx.Code != "",
		IsContractExecution: len(coreTx.Receipt.Transitions) > 0 || len(coreTx.Receipt.EventLogs) > 0,
	}

	if tx.IsContractExecution {
		tx.ContractAddress = fmt.Sprintf("0x%s", coreTx.ToAddr)
		tx.ContractAddressBech32 = getBech32Address(tx.ContractAddress)
	}

	if tx.SenderPubKey != "" {
		senderAddr := keytools.GetAddressFromPublic(util.DecodeHex(tx.SenderPubKey))
		tx.SenderAddr = fmt.Sprintf("0x%s", senderAddr)
		tx.SenderBech32 = getBech32Address(senderAddr)
	}

	return tx
}

func (f factory) createReceipt(coreReceipt core.TransactionReceipt) zil.TransactionReceipt {
	return zil.TransactionReceipt{
		TransactionReceipt: coreReceipt,
		Transitions:        f.createTransitions(coreReceipt.Transitions),
		EventLogs:          f.createEventLogs(coreReceipt.EventLogs),
	}
}

func (f factory) createTransitions(coreTransitions []core.Transition) (transitions []zil.Transition) {
	for _, transition := range coreTransitions {
		transitions = append(transitions, zil.Transition{Transition: transition, Msg: f.createMessage(transition.Msg)})
	}
	return
}

func (f factory) createMessage(coreMessage core.TransactionMessage) zil.TransitionMessage {
	return zil.TransitionMessage{TransactionMessage: coreMessage, Params: f.createParams(coreMessage.Params)}
}

func (f factory) createParams(coreParams []core.ContractValue) (params []zil.Param) {
	for _, coreParam := range coreParams {
		param := zil.Param{VName: coreParam.VName, Type: coreParam.Type}

		switch coreParam.Value.(type) {
		case string:
			param.Value = &zil.Value{Primitive: coreParam.Value}
			break
		case []interface{}:
			values := coreParam.Value.([]interface{})
			if len(values) != 0 {
				switch reflect.TypeOf(values[0]).String() {
				case "string":
					param.Value = &zil.Value{Primitive: values}
					break
				default:
					param.Value = f.createValueObject(values[0].(map[string]interface{}))
				}
			}
			break
		case map[string]interface{}:
			param.Value = f.createValueObject(coreParam.Value.(map[string]interface{}))
			break
		default:
			value, _ := json.Marshal(coreParam.Value)

			zap.L().With(
				zap.ByteString("value", value),
				zap.String("coreParam", fmt.Sprintf("%v", coreParam)),
				zap.String("type", reflect.TypeOf(coreParam.Value).String()),
			).Fatal("Unexpected data type")
		}
		params = append(params, param)
	}

	return params
}

func (f factory) stringToUint64(value string) uint64 {
	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("value", value)).Fatal("Failed to parse string as uint")
	}

	return number
}

func (f factory) createEventLogs(coreEventLogs []interface{}) []zil.EventLog {
	eventLogs := make([]zil.EventLog, 0)

	for _, coreEventLog := range coreEventLogs {
		eventLog := zil.EventLog{}
		if eventName, ok := coreEventLog.(map[string]interface{})["_eventname"]; ok {
			eventLog.EventName = eventName.(string)
		}
		if address, ok := coreEventLog.(map[string]interface{})["address"]; ok {
			eventLog.Address = address.(string)
		}

		if coreParams, ok := coreEventLog.(map[string]interface{})["params"]; ok {
			eventLog.Params = f.createParamsFromInterface(coreParams)
		}
		eventLogs = append(eventLogs, eventLog)
	}

	return eventLogs
}

func (f factory) createParamsFromInterface(coreParams interface{}) zil.Params {
	if coreParams == nil {
		return nil
	}
	var params zil.Params

	for _, coreParam := range coreParams.([]interface{}) {
		coreParamMap := coreParam.(map[string]interface{})
		param := zil.Param{
			Type:  coreParamMap["type"].(string),
			Value: f.createValueObject(coreParamMap["value"]),
			VName: coreParamMap["vname"].(string),
		}

		params = append(params, param)
	}

	return params
}

func (f factory) createParamsFromString(paramString interface{}) (data zil.Data) {
	if paramString == nil {
		return
	}

	var coreParams []interface{}
	err := json.Unmarshal([]byte(paramString.(string)), &coreParams)
	if err != nil {
		var coreParamsObj map[string]interface{}
		err := json.Unmarshal([]byte(paramString.(string)), &coreParamsObj)
		if err != nil {
			zap.L().With(zap.Error(err)).Fatal("Failed to unmarshal data")
		}
		if val, ok := coreParamsObj["_tag"]; ok {
			data.Tag = val.(string)
		}
		if val, ok := coreParamsObj["params"]; ok {
			coreParams = val.([]interface{})
		} else {
			coreParams = nil
		}
	}

	if coreParams == nil {
		return
	}

	for _, coreParam := range coreParams {
		if coreParam, ok := coreParam.(map[string]interface{}); ok {
			if coreParam != nil {
				param := zil.Param{
					Type:  f.getParam("type", coreParam).(string),
					Value: f.createValueObject(f.getParam("value", coreParam)),
					VName: f.getParam("vname", coreParam).(string),
				}

				data.Params = append(data.Params, param)
			}
		}
	}

	return
}

func (f factory) getParam(key string, coreParam map[string]interface{}) interface{} {
	if val, ok := coreParam[key]; ok {
		return val
	}
	return ""
}

func (f factory) createValueObject(valueObj interface{}) *zil.Value {
	if valueObj == nil {
		return nil
	}

	switch reflect.TypeOf(valueObj).String() {
	case "string":
		return &zil.Value{Primitive: valueObj.(string)}
	case "float64":
		return &zil.Value{Primitive: valueObj}
	case "[]interface {}":
		valueJson, _ := json.Marshal(valueObj)
		return &zil.Value{Primitive: string(valueJson)}
	default:
		if valueMap, ok := valueObj.(map[string]interface{}); ok {
			if _, ok := valueMap["arguments"]; ok {
				switch reflect.TypeOf(valueMap["arguments"]).String() {
				case "[]interface {}":
					arguments := make([]*zil.Value, 0)
					for _, argument := range valueMap["arguments"].([]interface{}) {
						arguments = append(arguments, f.createValueObject(argument))
					}
					return &zil.Value{
						ArgTypes:    valueMap["argtypes"],
						Arguments:   arguments,
						Constructor: valueMap["constructor"].(string),
					}
				default:
					zap.L().Info("TYPE: " + reflect.TypeOf(valueObj).String())
					log.Println("We want to know what object type arguments is")
					zap.L().Fatal(reflect.TypeOf(valueMap["arguments"]).String())
				}
			}
		} else {
			zap.L().Info("TYPE: " + reflect.TypeOf(valueObj).String())
			mapJson, _ := json.Marshal(valueObj)
			log.Println(string(mapJson))

			zap.L().With(zap.String("type", reflect.TypeOf(valueMap["value"]).String())).Fatal("ADT is not a map[string]interface{}")
		}
	}

	return &zil.Value{}
}
