// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	big_p "github.com/cryptoriums/packages/big"
	client_p "github.com/cryptoriums/packages/client"
	"github.com/cryptoriums/packages/contracts/bindings/interfaces"
	"github.com/cryptoriums/packages/env"
	prompt_p "github.com/cryptoriums/packages/prompt"
	tx_p "github.com/cryptoriums/packages/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
)

type TokenCmd struct {
	Transfer TokenTransferCmd `cmd:"" help:"transfer eth of other tokens"`
	Approve  TokenApproveCmd  `cmd:"" help:"approve tokens spendings"`
}

type TokenApproveCmd struct{}

func (self *TokenApproveCmd) Run(cliContext *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt_p.ReadFile()
	if err != nil {
		return err
	}

	_tags, err := prompt.Stdin.Prompt("enter tags separated by a comma: ")
	if err != nil {
		return errors.Wrap(err, "prompt tags")
	}
	tags := strings.Split(_tags, ",")

	e, err := env.LoadFromFile(filePath, tags...)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	client, err := client_p.NewClientCachedNetID(ctx, logger, e.Nodes[0].URL)
	if err != nil {
		return errors.Wrap(err, "NewClientCachedNetID")
	}

	token, err := prompt_p.Token(client.NetworkID())
	if err != nil {
		return errors.Wrap(err, "selectToken")
	}

	firstRun := true
	for {
		senderAcc, pass, err := env.SelectAccountAndDecrypt(e.Accounts, firstRun, "Select sender's pub address:")
		if err != nil {
			return errors.Wrap(err, "SelectAccountAndDecrypt sender")
		}
		firstRun = false

		if env.IsEncryptedEnv(e) {
			e, err = env.DecryptEnv(e, pass)
			if err != nil {
				return errors.Wrap(err, "DecryptEnv")
			}
		}

		var spender common.Address
		for {
			_spender, err := prompt.Stdin.PromptInput("Select spender contract: ")
			if err != nil {
				fmt.Println("prompt error for spender contract:", err)
				continue
			}
			if !common.IsHexAddress(_spender) {
				fmt.Println("spender is not a hex address")
				continue
			}
			spender = common.HexToAddress(_spender)
			break
		}

		var amount float64
		for {
			amount, err = prompt_p.Float(token.Name+" approve limit: ", 0, 1000000)
			if err != nil {
				return errors.Wrap(err, "select amount prompt")
			}
			break
		}

		var erc20I *interfaces.IERC20

		tokenAddr, ok := token.Address[client.NetworkID()]
		if !ok {
			return errors.Errorf("unknown token address for network:%v", client.NetworkID())
		}
		erc20I, err = interfaces.NewIERC20(tokenAddr, client)
		if err != nil {
			return errors.Wrap(err, "NewIERC20")
		}

		useProxy, err := prompt.Stdin.PromptConfirm("use proxy?")
		if err != nil {
			return errors.Wrap(err, "select proxy")
		}
		if useProxy {
			contract, _, err := prompt_p.Contract(e.Contracts, false, false)
			if err != nil {
				return errors.Wrap(err, "select contract")
			}
			erc20I, err = interfaces.NewIERC20(*contract, client)
			if err != nil {
				return errors.Wrap(err, "NewIERC20 through a proxy")
			}
		}

		gasPrice, err := prompt_p.Float("enter TX gas price(gwei): ", 0, 300)
		if err != nil {
			return err
		}

		ethAcc, err := tx_p.AccountFromPrvKey(senderAcc.Priv)
		if err != nil {
			return errors.Wrap(err, "AccountFromPrvKey")
		}

		nonce, err := prompt_p.Nonce(ctx, client, ethAcc.PublicKey)
		if err != nil {
			return errors.Wrap(err, "selectNonce")
		}
		confirmed, err := prompt.Stdin.PromptConfirm(fmt.Sprintf("Confirm approve of:%v from:%v, to:%v, amount:%v, gas price:%v", token.Name, senderAcc.Pub, spender, amount, gasPrice))
		if err != nil || !confirmed {
			return errors.New("canceled")
		}

		opts, err := tx_p.NewTxOpts(ctx, client, nonce, ethAcc, gasPrice, gasPrice, 150_000)
		if err != nil {
			return errors.Wrap(err, "NewTxOpts")
		}

		tx, err := erc20I.Approve(opts, spender, big_p.FromFloatMul(amount, params.Ether))
		if err != nil {
			return errors.Wrap(err, "Approve")
		}

		fmt.Println("Tx Created", "nonce", nonce, "hash", tx.Hash())

		anotherRun, err := prompt.Stdin.PromptConfirm("Another run?")
		if err != nil {
			return errors.Wrap(err, "prompt for another run")
		}
		if !anotherRun {
			break
		}
	}
	return nil
}

type TokenTransferCmd struct{}

func (self *TokenTransferCmd) Run(cliContext *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt_p.ReadFile()
	if err != nil {
		return err
	}

	_tags, err := prompt.Stdin.Prompt("enter tags separated by a comma: ")
	if err != nil {
		return errors.Wrap(err, "prompt tags")
	}
	tags := strings.Split(_tags, ",")

	e, err := env.LoadFromFile(filePath, tags...)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	client, err := client_p.NewClientCachedNetID(ctx, logger, e.Nodes[0].URL)
	if err != nil {
		return errors.Wrap(err, "NewClientCachedNetID")
	}

	token, err := prompt_p.Token(client.NetworkID())
	if err != nil {
		return errors.Wrap(err, "selectToken")
	}

	firstRun := true
	for {
		senderAcc, pass, err := env.SelectAccountAndDecrypt(e.Accounts, firstRun, "Select sender's pub address:")
		if err != nil {
			return errors.Wrap(err, "SelectAccountAndDecrypt sender")
		}
		firstRun = false

		if env.IsEncryptedEnv(e) {
			e, err = env.DecryptEnv(e, pass)
			if err != nil {
				return errors.Wrap(err, "DecryptEnv")
			}
		}

		// Deep copy to not modify the original slice.
		var accountsAndContracts []env.Account
		copier.CopyWithOption(&accountsAndContracts, e.Accounts, copier.Option{DeepCopy: true})
		for _, contract := range e.Contracts {
			accountsAndContracts = append(accountsAndContracts, env.Account{
				Pub:  contract.Address,
				Tags: append(contract.Tags, "contract"),
			})
		}

		receiverAcc, err := env.SelectAccount(accountsAndContracts, false, "Select receiver's pub address:")
		if err != nil {
			return errors.Wrap(err, "SelectAccount receiver")
		}

		var amount float64
		for {
			_amount, err := prompt.Stdin.PromptInput(token.Name + " аmount: ")
			if err != nil {
				return errors.Wrap(err, "select amount prompt")
			}
			amount, err = strconv.ParseFloat(_amount, 64)
			if err != nil {
				level.Error(logger).Log("msg", "casting input to float", "err", err)
				continue
			}

			break
		}

		gasPrice, err := prompt_p.Float("enter TX gas price(gwei): ", 0, 300)
		if err != nil {
			return err
		}

		ethAcc, err := tx_p.AccountFromPrvKey(senderAcc.Priv)
		if err != nil {
			return errors.Wrap(err, "AccountFromPrvKey")
		}

		nonce, err := prompt_p.Nonce(ctx, client, ethAcc.PublicKey)
		if err != nil {
			return errors.Wrap(err, "selectNonce")
		}

		var tx *types.Transaction
		if token.Name == env.ETH_TOKEN.Name {
			tx, _, err = tx_p.NewSignedTX(
				ctx,
				ethAcc.PrivateKey,
				receiverAcc.Pub,
				"",
				nonce,
				client.NetworkID(),
				"",
				nil,
				21_000,
				gasPrice,
				gasPrice,
				amount,
			)
			if err != nil {
				return errors.Wrap(err, "NewSignedTX")
			}
			err = client.SendTransaction(ctx, tx)
			if err != nil {
				fmt.Println("SendTransaction", "err", err.Error())
				continue

			}
		} else {
			erc20I, err := interfaces.NewIERC20(token.Address[client.NetworkID()], client)
			if err != nil {
				return errors.Wrap(err, "NewIERC20")
			}

			proxy, _, err := prompt_p.Contract(e.Contracts, false, true)
			if err != nil {
				return errors.Wrap(err, "selectProxy")
			}
			if proxy != nil {
				erc20I, err = interfaces.NewIERC20(*proxy, client)
				if err != nil {
					return errors.Wrap(err, "NewIERC20 through a proxy")
				}
			}

			opts, err := tx_p.NewTxOpts(ctx, client, nonce, ethAcc, gasPrice, gasPrice, 150_000)
			if err != nil {
				return errors.Wrap(err, "NewTxOpts")
			}
			tx, err = erc20I.Transfer(opts, receiverAcc.Pub, big_p.FromFloatMul(amount, params.Ether))
			if err != nil {
				fmt.Println("Transfer", "err", err.Error())
				continue
			}

		}

		fmt.Println("Tx Created", "nonce", nonce, "hash", tx.Hash())

		anotherRun, err := prompt.Stdin.PromptConfirm("Another run?")
		if err != nil {
			return errors.Wrap(err, "prompt for another run")
		}
		if !anotherRun {
			break
		}
	}
	return nil
}
