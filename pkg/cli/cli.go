// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	big_p "github.com/cryptoriums/packages/big"
	client_p "github.com/cryptoriums/packages/client"
	"github.com/cryptoriums/packages/env"
	"github.com/cryptoriums/packages/prompt"
	tx_p "github.com/cryptoriums/packages/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/willabides/kongplete"
)

var CLIInstance CLI

type Gas struct {
	GasPrice float64 `optional:"" help:"gas max fee to use when running the command"`
}

func (self *Gas) Validate() error {
	if self.GasPrice > 300 {
		confirmed, err := prompt.PromptConfirm(fmt.Sprintf("confirm high gas fee:%v", self.GasPrice))
		if err != nil || !confirmed {
			return errors.New("canceled")
		}
	}
	return nil
}

type CLI struct {
	Mnemonic           MnemonicCmd                  `cmd:"" help:"Generate a new mnemonic"`
	CancelTx           CancelTxCmd                  `cmd:"" help:"Cancel a pending TX"`
	Env                EnvCmd                       `cmd:"" help:"Env commands"`
	Encrypt            EncryptCmd                   `cmd:"" help:"Encrypts a string"`
	Decrypt            DecryptCmd                   `cmd:"" help:"Decrypts a string"`
	Token              TokenCmd                     `cmd:"" help:"token commands"`
	SetOwner           SetOwnerCmd                  `cmd:"" help:"set a new owner of a contract"`
	Account            AccountCmd                   `cmd:"" help:"account management"`
	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"install shell completions"`
}

type CancelTxCmd struct{}

func (self *CancelTxCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt.ReadFile()
	if err != nil {
		return err
	}
	_tags, err := prompt.PromptInput("enter tags separated by a comma: ")
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

	hash, err := prompt.PromptInput("TX hash to cancel: ")
	if err != nil {
		return errors.Wrap(err, "TX hash input prompt")
	}

	tx, isPending, err := client.TransactionByHash(ctx, common.HexToHash(hash))
	if err != nil {
		return errors.Wrap(err, "TransactionByHash")
	}
	if !isPending {
		return errors.New("TX is not in pending state")
	}

	signer := types.LatestSignerForChainID(big.NewInt(client.NetworkID()))

	sender, err := signer.Sender(tx)
	if err != nil {
		return errors.Wrap(err, "signer.Sender")
	}

	var acc tx_p.Account
	for _, a := range e.Accounts {
		if a.Pub.Hex() == sender.Hex() {
			if env.IsEncrypted(a.Priv) {
				a.Priv, _, err = env.DecryptWithPasswordLoop(a.Priv)
				if err != nil {
					return errors.Wrap(err, "DecryptWithPasswordLoop")
				}
			}

			acc, err = tx_p.AccountFromPrvKey(a.Priv)
			if err != nil {
				return errors.Wrap(err, "AccountFromPrvKey")
			}
			break
		}
	}

	gasPrice := big_p.ToFloatDiv(tx.GasPrice(), params.GWei)

	nonce, err := client.NonceAt(ctx, acc.PublicKey, nil)
	if err != nil {
		return errors.Wrap(err, "NonceAt")
	}
	tx, _, err = tx_p.NewSignedTX(
		ctx,
		acc.PrivateKey,
		acc.PublicKey,
		"",
		nonce,
		client.NetworkID(),
		"",
		nil,
		300_000,
		gasPrice*1.1,
		gasPrice*1.1,
		0,
	)
	if err != nil {
		return errors.Wrap(err, "NewSignedTX")
	}

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "SendTransaction")
	}

	return nil
}

type EnvCmd struct {
	ReEncrypt EnvReEncryptCmd `cmd:"" help:"Change the env file password"`
	Encrypt   EnvEncryptCmd   `cmd:"" help:"Encrypts all objects with the given tags"`
	Export    EnvExportCmd    `cmd:"" help:"Export the env filtered by given tags"`
}

type EnvExportCmd struct{}

func (self *EnvExportCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt.ReadFile()
	if err != nil {
		return errors.Wrap(err, "prompt.ReadFile")
	}

	_tags, err := prompt.PromptInput("enter tags for objects to be exported separated by a comma: ")
	if err != nil {
		return errors.Wrap(err, "prompt tags")
	}
	tags := strings.Split(_tags, ",")

	e, err := env.LoadFromFile(filePath, tags...)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	decrypt, err := prompt.PromptConfirm("Decrypt env?")
	if err != nil {
		return errors.Wrap(err, "prompt decrypt")
	}

	if decrypt {
		e, _, err = env.DecryptEnvWithPasswordLoop(e)
		if err != nil {
			return errors.Wrap(err, "DecryptEnvWithPasswordLoop")
		}
	}

	content, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		return errors.Wrap(err, "marshal env")
	}
	fmt.Println(string(content))

	return nil
}

type EnvEncryptCmd struct{}

func (self *EnvEncryptCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt.ReadFile()
	if err != nil {
		return errors.Wrap(err, "prompt.ReadFile")
	}

	e, err := env.LoadFromFile(filePath)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	_tags, err := prompt.PromptInput("enter tags for objects to be encrypted separated by a comma:")
	if err != nil {
		return errors.Wrap(err, "prompt tags")
	}
	tags := strings.Split(_tags, ",")

	e, pass, err := env.ReEncryptEnvWithPasswordLoop(e)
	if err != nil {
		return errors.Wrap(err, "ReEncryptEnvWithPasswordLoop")
	}

	for i, acc := range e.Accounts {
		if env.Contains(tags, acc.Tags) {
			if env.IsEncrypted(acc.Priv) {
				continue
			}
			encrypted, err := env.Encrypt(acc.Priv, pass)
			if err != nil {
				return errors.Wrap(err, "env.Encrypt")
			}
			e.Accounts[i].Priv = encrypted
		}
	}

	for i, key := range e.ApiKeys {
		if env.Contains(tags, key.Tags) {
			if env.IsEncrypted(key.Value) {
				continue
			}
			encrypted, err := env.Encrypt(key.Value, pass)
			if err != nil {
				return errors.Wrap(err, "env.Encrypt")
			}
			e.ApiKeys[i].Value = encrypted
		}
	}

	// Verify decryption.
	_, err = env.DecryptEnv(e, pass)
	if err != nil {
		return errors.Wrap(err, "decryption verification")
	}

	content, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		return errors.Wrap(err, "marshal env")
	}

	err = os.WriteFile(filePath, content, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "write env to file")
	}

	level.Info(logger).Log("msg", "env file are encrypted", "tags", _tags)
	return nil
}

type EnvReEncryptCmd struct{}

func (self *EnvReEncryptCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt.ReadFile()
	if err != nil {
		return err
	}
	e, err := env.LoadFromFile(filePath)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	e, pass, err := env.ReEncryptEnvWithPasswordLoop(e)
	if err != nil {
		return errors.Wrap(err, "ReEncryptEnvWithPasswordLoop")
	}

	// Verify decryption.
	_, err = env.DecryptEnv(e, pass)
	if err != nil {
		return errors.Wrap(err, "decryption verification")
	}

	content, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		return errors.Wrap(err, "marshal env")
	}

	err = os.WriteFile(filePath, content, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "write env to file")
	}

	level.Info(logger).Log("msg", "env file re-encrypted")
	return nil
}

type EncryptCmd struct{}

func (self *EncryptCmd) Run(cli *CLI) error {
	input, err := prompt.PromptInput("Input to encrypt: ")
	if err != nil {
		return err
	}
	encrypted, pass, err := env.EncryptWithPasswordLoop(input)
	if err != nil {
		return err
	}
	// Verify decryption.
	decrypted, err := env.Decrypt(encrypted, pass)
	if err != nil {
		return errors.Wrap(err, "decryption verification")
	}
	if decrypted != input {
		return errors.Errorf("decryption verification mismatch exp:%v, got:%v", input, decrypted)

	}
	fmt.Println(encrypted)
	return nil
}

type DecryptCmd struct{}

func (self *DecryptCmd) Run(cli *CLI) error {
	input, err := prompt.PromptInput("Input to decrypt: ")
	if err != nil {
		return err
	}
	decrypted, _, err := env.DecryptWithPasswordLoop(input)
	if err != nil {
		return err
	}
	fmt.Println(decrypted)
	return nil
}

type MnemonicCmd struct{}

func (self *MnemonicCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	entropy := make([]byte, 32)
	rand.Seed(time.Now().UnixNano())
	rand.Read(entropy)
	// the entropy can be any byte slice, generated how pleased,
	// as long its bit size is a multiple of 32 and is within
	// the inclusive range of {128,256}
	mnemomic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return err
	}
	fmt.Println(mnemomic)
	return nil
}

type AccountCmd struct {
	Import   AccountImportCmd  `cmd:"" help:"import an acount by a private key"`
	New      AccountNewCmd     `cmd:"" help:"generate new pub/priv key accounts"`
	Balances AccountBalanceCmd `cmd:"" help:"show all eth or erc20 balances"`
}
