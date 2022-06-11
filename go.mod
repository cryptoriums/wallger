module github.com/cryptoriums/wallger

go 1.18

require (
	github.com/alecthomas/kong v0.5.1-0.20220518080721-195d56c42e0f
	github.com/btcsuite/btcd v0.22.0-beta.0.20220330201728-074266215c26
	github.com/btcsuite/btcd/btcutil v1.1.1
	github.com/cryptoriums/packages v0.0.0-20220529143219-92e77e6cf241
	github.com/ethereum/go-ethereum v1.10.19-0.20220526072637-0287e1a7c00c
	github.com/go-kit/log v0.2.0
	github.com/jinzhu/copier v0.3.6-0.20220210061904-7948fe2be217
	github.com/pkg/errors v0.9.1
	github.com/posener/complete v1.2.3
	github.com/tyler-smith/go-bip39 v1.1.1-0.20201031083441-3423700f9707
	github.com/willabides/kongplete v0.3.0
)

require (
	github.com/bluele/gcache v0.0.2 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/peterh/liner v1.2.2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/sys v0.0.0-20220223155357-96fed51e1446 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

replace github.com/cryptoriums/packages => ../packages
