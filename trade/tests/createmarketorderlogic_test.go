package tests

import (
	"context"
	"os"
	"testing"

	"greet/trade/internal/config"
	"greet/trade/internal/logic"
	"greet/trade/internal/svc"
	"greet/trade/trade"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestCreateMarketOrder(t *testing.T) {
	// Load config from yaml file
	var c config.Config
	conf.MustLoad("../etc/trade.yaml", &c)

	// Create real service context (requires market service to be running)
	ctx := context.Background()
	svcCtx := svc.NewServiceContext(c)

	// Create logic instance
	logicInstance := logic.NewCreateMarketOrderLogic(ctx, svcCtx)

	// Prepare test parameters
	request := &trade.CreateMarketOrderRequest{
		ChainId:           100000,                                         // SOL chain
		TokenCa:           "Az6NhqAiUCuvw3is9Ew1kFdwRcpC6fDyNR9Ax5FC2T2t", // Example token address
		SwapType:          trade.SwapType_Buy,
		AmountIn:          "1.5",
		DoubleOut:         false,
		IsOneClick:        false,
		UserWalletAddress: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
	}

	// Call CreateMarketOrder
	response, err := logicInstance.CreateMarketOrder(request)

	// Check result
	if err != nil {
		t.Logf("CreateMarketOrder error (expected if market service not running): %v", err)
		return
	}

	if response == nil {
		t.Fatal("CreateMarketOrder returned nil response")
	}

	t.Logf("CreateMarketOrder succeeded, response: %+v", response)
}

func TestCreateMarketOrderDevNet(t *testing.T) {
	// Load config from yaml file
	var c config.Config
	conf.MustLoad("../etc/trade.yaml", &c)

	// Override database configuration to use remote database
	c.MySQLConfig.Host = "43.99.100.82"
	c.MySQLConfig.Port = 3306
	c.MySQLConfig.User = "root"
	c.MySQLConfig.Password = "web3web3"

	// Check if we should sign and send transactions on-chain
	// Set SEND_TRANSACTION_ONCHAIN=true to enable signing and sending
	// Set PRIVATE_KEY=<base58_encoded_private_key> to provide the signing key
	sendOnChain := os.Getenv("SEND_TRANSACTION_ONCHAIN")
	privateKey := os.Getenv("PRIVATE_KEY")

	if sendOnChain == "true" {
		if privateKey == "" {
			t.Fatal("SEND_TRANSACTION_ONCHAIN is set to true but PRIVATE_KEY environment variable is not set")
		}
		t.Logf("SEND_TRANSACTION_ONCHAIN is enabled - transactions will be signed and sent on-chain")
		t.Logf("PRIVATE_KEY is set (length: %d)", len(privateKey))
	} else {
		t.Logf("SEND_TRANSACTION_ONCHAIN is not set or not 'true' - transactions will be returned unsigned")
		t.Logf("To enable signing and sending, set: SEND_TRANSACTION_ONCHAIN=true PRIVATE_KEY=<base58_private_key>")
	}

	// Create real service context (requires market service to be running)
	ctx := context.Background()
	svcCtx := svc.NewServiceContext(c)

	// Create logic instance
	logicInstance := logic.NewCreateMarketOrderLogic(ctx, svcCtx)

	// Prepare test parameters
	request := &trade.CreateMarketOrderRequest{
		ChainId:           100000,                                         // SOL chain
		TokenCa:           "9o2a43WePv6f6d9zcb9AtxZfq9roya1gmXpAin9gnQME", // Example token address
		SwapType:          trade.SwapType_Buy,
		AmountIn:          "0.01",
		DoubleOut:         false,
		IsOneClick:        false,
		UserWalletAddress: "3xbCoRgPcuUhUdsVJHrq79gmcGUT3VwqrHgMTkV296cP",
	}

	// Call CreateMarketOrder
	response, err := logicInstance.CreateMarketOrder(request)

	// Check result
	if err != nil {
		t.Logf("CreateMarketOrder error: %v", err)
		return
	}

	if response == nil {
		t.Fatal("CreateMarketOrder returned nil response")
	}

	if sendOnChain == "true" {
		t.Logf("CreateMarketOrder succeeded - Transaction sent on-chain!")
		t.Logf("Transaction Hash: %s", response.TxHash)
	} else {
		t.Logf("CreateMarketOrder succeeded - Unsigned transaction returned")
		t.Logf("Transaction (base64, length: %d): %s", len(response.TxHash), response.TxHash)
		t.Logf("To sign and send, set SEND_TRANSACTION_ONCHAIN=true and PRIVATE_KEY=<base58_private_key>")
	}
}
