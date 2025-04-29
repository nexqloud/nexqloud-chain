// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package keeper

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// OnlineServerMonitorMetaData contains all meta data concerning the OnlineServerMonitor contract.
var OnlineServerMonitorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"getOnlineServerCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"onlineServerCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"reached1000ServerCount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"reached1000ServerCountValue\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_count\",\"type\":\"uint256\"}],\"name\":\"updateOnlineServerCount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// OnlineServerMonitorABI is the input ABI used to generate the binding from.
// Deprecated: Use OnlineServerMonitorMetaData.ABI instead.
var OnlineServerMonitorABI = OnlineServerMonitorMetaData.ABI

// OnlineServerMonitor is an auto generated Go binding around an Ethereum contract.
type OnlineServerMonitor struct {
	OnlineServerMonitorCaller     // Read-only binding to the contract
	OnlineServerMonitorTransactor // Write-only binding to the contract
	OnlineServerMonitorFilterer   // Log filterer for contract events
}

// OnlineServerMonitorCaller is an auto generated read-only Go binding around an Ethereum contract.
type OnlineServerMonitorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OnlineServerMonitorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OnlineServerMonitorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OnlineServerMonitorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OnlineServerMonitorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OnlineServerMonitorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OnlineServerMonitorSession struct {
	Contract     *OnlineServerMonitor // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// OnlineServerMonitorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OnlineServerMonitorCallerSession struct {
	Contract *OnlineServerMonitorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// OnlineServerMonitorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OnlineServerMonitorTransactorSession struct {
	Contract     *OnlineServerMonitorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// OnlineServerMonitorRaw is an auto generated low-level Go binding around an Ethereum contract.
type OnlineServerMonitorRaw struct {
	Contract *OnlineServerMonitor // Generic contract binding to access the raw methods on
}

// OnlineServerMonitorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OnlineServerMonitorCallerRaw struct {
	Contract *OnlineServerMonitorCaller // Generic read-only contract binding to access the raw methods on
}

// OnlineServerMonitorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OnlineServerMonitorTransactorRaw struct {
	Contract *OnlineServerMonitorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOnlineServerMonitor creates a new instance of OnlineServerMonitor, bound to a specific deployed contract.
func NewOnlineServerMonitor(address common.Address, backend bind.ContractBackend) (*OnlineServerMonitor, error) {
	contract, err := bindOnlineServerMonitor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OnlineServerMonitor{OnlineServerMonitorCaller: OnlineServerMonitorCaller{contract: contract}, OnlineServerMonitorTransactor: OnlineServerMonitorTransactor{contract: contract}, OnlineServerMonitorFilterer: OnlineServerMonitorFilterer{contract: contract}}, nil
}

// NewOnlineServerMonitorCaller creates a new read-only instance of OnlineServerMonitor, bound to a specific deployed contract.
func NewOnlineServerMonitorCaller(address common.Address, caller bind.ContractCaller) (*OnlineServerMonitorCaller, error) {
	contract, err := bindOnlineServerMonitor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OnlineServerMonitorCaller{contract: contract}, nil
}

// NewOnlineServerMonitorTransactor creates a new write-only instance of OnlineServerMonitor, bound to a specific deployed contract.
func NewOnlineServerMonitorTransactor(address common.Address, transactor bind.ContractTransactor) (*OnlineServerMonitorTransactor, error) {
	contract, err := bindOnlineServerMonitor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OnlineServerMonitorTransactor{contract: contract}, nil
}

// NewOnlineServerMonitorFilterer creates a new log filterer instance of OnlineServerMonitor, bound to a specific deployed contract.
func NewOnlineServerMonitorFilterer(address common.Address, filterer bind.ContractFilterer) (*OnlineServerMonitorFilterer, error) {
	contract, err := bindOnlineServerMonitor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OnlineServerMonitorFilterer{contract: contract}, nil
}

// bindOnlineServerMonitor binds a generic wrapper to an already deployed contract.
func bindOnlineServerMonitor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OnlineServerMonitorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OnlineServerMonitor *OnlineServerMonitorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OnlineServerMonitor.Contract.OnlineServerMonitorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OnlineServerMonitor *OnlineServerMonitorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.OnlineServerMonitorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OnlineServerMonitor *OnlineServerMonitorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.OnlineServerMonitorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OnlineServerMonitor *OnlineServerMonitorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OnlineServerMonitor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OnlineServerMonitor *OnlineServerMonitorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OnlineServerMonitor *OnlineServerMonitorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.contract.Transact(opts, method, params...)
}

// GetOnlineServerCount is a free data retrieval call binding the contract method 0xabaf6060.
//
// Solidity: function getOnlineServerCount() view returns(uint256)
func (_OnlineServerMonitor *OnlineServerMonitorCaller) GetOnlineServerCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OnlineServerMonitor.contract.Call(opts, &out, "getOnlineServerCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetOnlineServerCount is a free data retrieval call binding the contract method 0xabaf6060.
//
// Solidity: function getOnlineServerCount() view returns(uint256)
func (_OnlineServerMonitor *OnlineServerMonitorSession) GetOnlineServerCount() (*big.Int, error) {
	return _OnlineServerMonitor.Contract.GetOnlineServerCount(&_OnlineServerMonitor.CallOpts)
}

// GetOnlineServerCount is a free data retrieval call binding the contract method 0xabaf6060.
//
// Solidity: function getOnlineServerCount() view returns(uint256)
func (_OnlineServerMonitor *OnlineServerMonitorCallerSession) GetOnlineServerCount() (*big.Int, error) {
	return _OnlineServerMonitor.Contract.GetOnlineServerCount(&_OnlineServerMonitor.CallOpts)
}

// OnlineServerCount is a free data retrieval call binding the contract method 0xf36f3fc9.
//
// Solidity: function onlineServerCount() view returns(uint256)
func (_OnlineServerMonitor *OnlineServerMonitorCaller) OnlineServerCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OnlineServerMonitor.contract.Call(opts, &out, "onlineServerCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OnlineServerCount is a free data retrieval call binding the contract method 0xf36f3fc9.
//
// Solidity: function onlineServerCount() view returns(uint256)
func (_OnlineServerMonitor *OnlineServerMonitorSession) OnlineServerCount() (*big.Int, error) {
	return _OnlineServerMonitor.Contract.OnlineServerCount(&_OnlineServerMonitor.CallOpts)
}

// OnlineServerCount is a free data retrieval call binding the contract method 0xf36f3fc9.
//
// Solidity: function onlineServerCount() view returns(uint256)
func (_OnlineServerMonitor *OnlineServerMonitorCallerSession) OnlineServerCount() (*big.Int, error) {
	return _OnlineServerMonitor.Contract.OnlineServerCount(&_OnlineServerMonitor.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OnlineServerMonitor *OnlineServerMonitorCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OnlineServerMonitor.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OnlineServerMonitor *OnlineServerMonitorSession) Owner() (common.Address, error) {
	return _OnlineServerMonitor.Contract.Owner(&_OnlineServerMonitor.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OnlineServerMonitor *OnlineServerMonitorCallerSession) Owner() (common.Address, error) {
	return _OnlineServerMonitor.Contract.Owner(&_OnlineServerMonitor.CallOpts)
}

// Reached1000ServerCountValue is a free data retrieval call binding the contract method 0xa08c4eec.
//
// Solidity: function reached1000ServerCountValue() view returns(bool)
func (_OnlineServerMonitor *OnlineServerMonitorCaller) Reached1000ServerCountValue(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _OnlineServerMonitor.contract.Call(opts, &out, "reached1000ServerCountValue")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Reached1000ServerCountValue is a free data retrieval call binding the contract method 0xa08c4eec.
//
// Solidity: function reached1000ServerCountValue() view returns(bool)
func (_OnlineServerMonitor *OnlineServerMonitorSession) Reached1000ServerCountValue() (bool, error) {
	return _OnlineServerMonitor.Contract.Reached1000ServerCountValue(&_OnlineServerMonitor.CallOpts)
}

// Reached1000ServerCountValue is a free data retrieval call binding the contract method 0xa08c4eec.
//
// Solidity: function reached1000ServerCountValue() view returns(bool)
func (_OnlineServerMonitor *OnlineServerMonitorCallerSession) Reached1000ServerCountValue() (bool, error) {
	return _OnlineServerMonitor.Contract.Reached1000ServerCountValue(&_OnlineServerMonitor.CallOpts)
}

// Reached1000ServerCount is a paid mutator transaction binding the contract method 0xbbcba63c.
//
// Solidity: function reached1000ServerCount() returns()
func (_OnlineServerMonitor *OnlineServerMonitorTransactor) Reached1000ServerCount(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OnlineServerMonitor.contract.Transact(opts, "reached1000ServerCount")
}

// Reached1000ServerCount is a paid mutator transaction binding the contract method 0xbbcba63c.
//
// Solidity: function reached1000ServerCount() returns()
func (_OnlineServerMonitor *OnlineServerMonitorSession) Reached1000ServerCount() (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.Reached1000ServerCount(&_OnlineServerMonitor.TransactOpts)
}

// Reached1000ServerCount is a paid mutator transaction binding the contract method 0xbbcba63c.
//
// Solidity: function reached1000ServerCount() returns()
func (_OnlineServerMonitor *OnlineServerMonitorTransactorSession) Reached1000ServerCount() (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.Reached1000ServerCount(&_OnlineServerMonitor.TransactOpts)
}

// UpdateOnlineServerCount is a paid mutator transaction binding the contract method 0x63f48996.
//
// Solidity: function updateOnlineServerCount(uint256 _count) returns()
func (_OnlineServerMonitor *OnlineServerMonitorTransactor) UpdateOnlineServerCount(opts *bind.TransactOpts, _count *big.Int) (*types.Transaction, error) {
	return _OnlineServerMonitor.contract.Transact(opts, "updateOnlineServerCount", _count)
}

// UpdateOnlineServerCount is a paid mutator transaction binding the contract method 0x63f48996.
//
// Solidity: function updateOnlineServerCount(uint256 _count) returns()
func (_OnlineServerMonitor *OnlineServerMonitorSession) UpdateOnlineServerCount(_count *big.Int) (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.UpdateOnlineServerCount(&_OnlineServerMonitor.TransactOpts, _count)
}

// UpdateOnlineServerCount is a paid mutator transaction binding the contract method 0x63f48996.
//
// Solidity: function updateOnlineServerCount(uint256 _count) returns()
func (_OnlineServerMonitor *OnlineServerMonitorTransactorSession) UpdateOnlineServerCount(_count *big.Int) (*types.Transaction, error) {
	return _OnlineServerMonitor.Contract.UpdateOnlineServerCount(&_OnlineServerMonitor.TransactOpts, _count)
}
