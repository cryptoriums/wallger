package cli

import (
	"context"
	"fmt"
	"strings"

	client_p "github.com/cryptoriums/packages/client"
	"github.com/cryptoriums/packages/contracts/bindings/interfaces"
	"github.com/cryptoriums/packages/env"
	prompt_p "github.com/cryptoriums/packages/prompt"
	tx_p "github.com/cryptoriums/packages/tx"
	"github.com/ethereum/go-ethereum/console/prompt"

	"github.com/go-kit/log"
	"github.com/pkg/errors"
)

type SetOwnerCmd struct{}

func (self *SetOwnerCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt_p.ReadFile()
	if err != nil {
		return err
	}
	_tags, err := prompt.Stdin.Prompt("enter tags separated by a comma: ")
	if err != nil {
		return errors.Wrap(err, "prompt tags")
	}
	tags := strings.Split(_tags, ",")

	envr, err := env.LoadFromFile(filePath, tags...)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	client, err := client_p.NewClientCachedNetID(ctx, logger, envr.Nodes[0].URL)
	if err != nil {
		return errors.Wrap(err, "NewClientCachedNetID")
	}

	for {
		currentOwner, pass, err := env.SelectAccountAndDecrypt(envr.Accounts, false, "Select current owner's pub address:")
		if err != nil {
			return errors.Wrap(err, "SelectAccountAndDecrypt sender")
		}

		if env.IsEncryptedEnv(envr) {
			envr, err = env.DecryptEnv(envr, pass)
			if err != nil {
				return errors.Wrap(err, "DecryptEnv")
			}
		}

		currentOwnerAcc, err := tx_p.AccountFromPrvKey(currentOwner.Priv)
		if err != nil {
			return errors.Wrap(err, "AccountFromPrvKey")
		}

		newOwner, err := env.SelectAccount(envr.Accounts, false, "Select new owner's pub address:")
		if err != nil {
			return errors.Wrap(err, "SelectAccount receiver")
		}

		conract, _, err := prompt_p.Contract(envr.Contracts, false, false)
		if err != nil {
			return errors.Wrap(err, "selectProxy")
		}

		ownable, err := interfaces.NewOwneable(*conract, client)
		if err != nil {
			return errors.Wrap(err, "NewIERC20")
		}

		nonce, err := prompt_p.Nonce(ctx, client, currentOwner.Pub)
		if err != nil {
			return errors.Wrap(err, "selectNonce")
		}

		gasPrice, err := prompt_p.Float("enter TX gas price(gwei): ", 0, 300)
		if err != nil {
			return err
		}

		opts, err := tx_p.NewTxOpts(ctx, client, nonce, currentOwnerAcc, gasPrice, gasPrice, 150_000)
		if err != nil {
			return errors.Wrap(err, "NewTxOpts")
		}
		tx, err := ownable.SetOwner(opts, newOwner.Pub)
		if err != nil {
			return errors.Wrap(err, "Transfer")
		}

		fmt.Println("Tx Created", "nonce", nonce, "hash", tx.Hash())

		anotherRun, err := prompt.Stdin.PromptConfirm("Another run?")
		if err != nil {
			return errors.Wrap(err, "prompt for another transfer")
		}
		if !anotherRun {
			break
		}
	}

	return nil

}
