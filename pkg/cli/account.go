package cli

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	big_p "github.com/cryptoriums/packages/big"
	client_p "github.com/cryptoriums/packages/client"
	"github.com/cryptoriums/packages/contracts/bindings/interfaces"
	"github.com/cryptoriums/packages/env"
	prompt_p "github.com/cryptoriums/packages/prompt"
	tx_p "github.com/cryptoriums/packages/tx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/console/prompt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
)

type AccountImportCmd struct{}

func (self *AccountImportCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt_p.ReadFile()
	if err != nil {
		return err
	}
	e, err := env.LoadFromFile(filePath)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	prvKeys, err := prompt.Stdin.PromptInput("Enter private keys separated by a comma: ")
	if err != nil {
		return errors.Wrap(err, "private key prompt")
	}
	if prvKeys == "" {
		return nil
	}

	var newAccs []env.Account
	for _, prvKey := range strings.Split(prvKeys, ",") {
		if !strings.HasPrefix(prvKey, "0x") {
			prvKey = "0x" + prvKey
		}
		_acc, err := tx_p.AccountFromPrvKey(prvKey)
		if err != nil {
			return errors.Wrap(err, "AccountFromPrvKey")
		}

		newAccs = append(newAccs, env.Account{
			Pub:  _acc.PublicKey,
			Priv: prvKey,
		})
	}

	yes, err := prompt.Stdin.PromptConfirm("Encrypt the imported account?")
	if err != nil {
		return errors.Wrap(err, "encrypt accounts prompt")
	}

	if yes {
		_, pass, err := env.DecryptEnvWithPasswordLoop(e)
		if err != nil {
			return errors.Wrap(err, "DecryptEnvWithPasswordLoop")
		}
		newAccs, err = env.EncryptAccounts(newAccs, pass)
		if err != nil {
			return errors.Wrap(err, "EncryptAccounts")
		}
		_, err = env.DecryptEnv(e, pass)
		if err != nil {
			return errors.Wrap(err, "verifying that the new env can be decrypted")
		}
	}
	acc, removed := DedupAccounts(append(e.Accounts, newAccs...))
	if len(removed) > 0 {
		level.Warn(logger).Log("msg", "!!!! removed duplicated accounts", "accounts", fmt.Sprintf("%+v", removed))
	}
	e.Accounts = acc

	content, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		return errors.Wrap(err, "marshal env")
	}

	err = os.WriteFile(filePath, content, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "write env to file")
	}

	level.Info(logger).Log("msg", "accounts imported to the env file")
	return nil

}

type AccountNewCmd struct{}

func (self *AccountNewCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	var count int
	for {
		_count, err := prompt.Stdin.PromptInput("How many accounts: ")
		if err != nil {
			return errors.Wrap(err, "accounts count prompt")
		}
		count, err = strconv.Atoi(_count)
		if err == nil {
			break
		}
		level.Error(logger).Log("msg", "casting count input", "err", err)
	}

	mnemonic, err := prompt.Stdin.PromptInput("Enter mnemonic or leave empty for random accounts: ")
	if err != nil {
		return errors.Wrap(err, "accounts count prompt")
	}

	var privKeys []*ecdsa.PrivateKey

	if mnemonic != "" {
		seed := bip39.NewSeed(mnemonic, "") // Here you can choose to pass in the specified password or empty string , Different passwords generate different mnemonics

		master, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
		if err != nil {
			return errors.Wrap(err, "hdkeychain.NewMaster")
		}

		for i := 0; i < count; i++ {
			path, err := accounts.ParseDerivationPath("m/44'/60'/" + strconv.Itoa(i) + "'/0/0")
			if err != nil {
				return errors.Wrap(err, "accounts.ParseDerivationPath")
			}

			privateKey, err := derivePrivateKey(master, path)
			if err != nil {
				return errors.Wrap(err, "derivePrivateKey")
			}

			privKeys = append(privKeys, privateKey)
		}
	} else {
		for i := 0; i < count; i++ {
			privateKey, err := crypto.GenerateKey()
			if err != nil {
				return errors.Wrap(err, "GenerateKey")
			}

			privKeys = append(privKeys, privateKey)

		}
	}

	var newAccs []env.Account
	for _, privKey := range privKeys {
		publicKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
		if !ok {
			return errors.New("failed to cast to public key")
		}

		publicAddr := crypto.PubkeyToAddress(*publicKeyECDSA)

		privateKey := hexutil.Encode(crypto.FromECDSA(privKey))

		newAccs = append(newAccs, env.Account{Pub: publicAddr, Priv: privateKey})

		err = tx_p.TestSignMessage(publicAddr, privKey)
		if err != nil {
			return errors.Wrap(err, "TestSignMessage")
		}

		fmt.Println(publicAddr, privateKey)

	}

	yes, err := prompt.Stdin.PromptConfirm("Add to the env file?")
	if err != nil {
		return errors.Wrap(err, "prompting for adding accounts to the env file")
	}
	if !yes {
		return nil
	}
	_, filePath, err := prompt_p.ReadFile()
	if err != nil {
		return err
	}
	e, err := env.LoadFromFile(filePath)
	if err != nil {
		return errors.Wrap(err, "loading env from file")
	}

	yes, err = prompt.Stdin.PromptConfirm("Encrypt new accounts?")
	if err != nil {
		return errors.Wrap(err, "encrypt accounts prompt")
	}
	if yes {
		_, pass, err := env.DecryptEnvWithPasswordLoop(e)
		if err != nil {
			return errors.Wrap(err, "DecryptEnvWithPasswordLoop")
		}
		newAccs, err = env.EncryptAccounts(newAccs, pass)
		if err != nil {
			return errors.Wrap(err, "EncryptAccounts")
		}

	}

	acc, removed := DedupAccounts(append(e.Accounts, newAccs...))
	if len(removed) > 0 {
		level.Warn(logger).Log("msg", "removed duplicated accounts", "accounts", fmt.Sprintf("%+v", removed))
	}
	e.Accounts = acc

	content, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		return errors.Wrap(err, "marshal env")
	}

	err = os.WriteFile(filePath, content, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "write env to file")
	}

	level.Info(logger).Log("msg", "new account added to the env file")
	return nil
}

func derivePrivateKey(_masterKey *hdkeychain.ExtendedKey, path accounts.DerivationPath) (*ecdsa.PrivateKey, error) {
	// Create a copy to not modify the source byte slices.
	masterKey := &hdkeychain.ExtendedKey{}
	copier.Copy(masterKey, _masterKey)

	var err error
	for _, n := range path {
		masterKey, err = masterKey.Derive(n)
		if err != nil {
			return nil, err
		}
	}

	privateKey, err := masterKey.ECPrivKey()
	privateKeyECDSA := privateKey.ToECDSA()
	if err != nil {
		return nil, err
	}

	return privateKeyECDSA, nil
}

type AccountBalanceCmd struct{}

func (self *AccountBalanceCmd) Run(cli *CLI, ctx context.Context, logger log.Logger) error {
	_, filePath, err := prompt_p.ReadFile()
	if err != nil {
		return errors.Wrap(err, "prompt.ReadFile")
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
	erc20I, err := interfaces.NewIERC20(token.Address[client.NetworkID()], client)
	if err != nil {
		return errors.Wrap(err, "NewIERC20")
	}
	fmt.Println("Token:" + token.Name)
	fmt.Println("Accounts")
	for i, account := range e.Accounts {
		var balance *big.Int

		if token.Name == env.ETH_TOKEN.Name {
			balance, err = client.BalanceAt(ctx, account.Pub, nil)
			if err != nil {
				return errors.Wrap(err, "client.BalanceAt")
			}
		} else {
			balance, err = erc20I.BalanceOf(&bind.CallOpts{Context: ctx}, account.Pub)
			if err != nil {
				return errors.Wrap(err, "erc20I.BalanceOf")
			}
		}
		fmt.Println(strconv.Itoa(i) + ": " + account.Pub.Hex() + " " + fmt.Sprintf("%.6f", big_p.ToFloatDiv(balance, params.Ether)) + " " + strings.Join(account.Tags, ","))

	}

	fmt.Println("Contracts")
	for i, contract := range e.Contracts {
		var balance *big.Int
		if token.Name == env.ETH_TOKEN.Name {
			balance, err = client.BalanceAt(ctx, contract.Address, nil)
			if err != nil {
				return errors.Wrap(err, "client.BalanceAt")
			}
		} else {
			balance, err = erc20I.BalanceOf(&bind.CallOpts{Context: ctx}, contract.Address)
			if err != nil {
				return errors.Wrap(err, "erc20I.BalanceOf")
			}
		}

		fmt.Println(strconv.Itoa(i) + ": " + contract.Address.Hex() + " " + fmt.Sprintf("%.6f", big_p.ToFloatDiv(balance, params.Ether)) + " " + strings.Join(contract.Tags, ","))
	}
	return nil
}

func DedupAccounts(accs []env.Account) ([]env.Account, map[common.Address]bool) {
	added := make(map[string]bool)
	var accsDedup []env.Account
	duplicated := make(map[common.Address]bool)
	for _, acc := range accs {
		if !added[acc.Pub.Hex()] {
			accsDedup = append(accsDedup, acc)
			added[acc.Pub.Hex()] = true
			continue
		}
		duplicated[acc.Pub] = true
	}
	return accsDedup, duplicated
}
