package streampay

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"payment-engine/internal/config"
)

// GrpcClient wraps gRPC connection and Cosmos SDK client context.
type GrpcClient struct {
	conn      *grpc.ClientConn
	clientCtx client.Context
	txConfig  client.TxConfig
}

// NewGrpcClient creates a new gRPC client and initializes connections.
func NewGrpcClient(cfg config.Config, reg codectypes.InterfaceRegistry, cdc codec.ProtoCodecMarshaler) (*GrpcClient, error) {
	log.Printf("Connecting to gRPC node at %s", cfg.GrpcAddress)
	conn, err := grpc.Dial(cfg.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC node %s: %w", cfg.GrpcAddress, err)
	}
	log.Printf("gRPC connection established to %s", cfg.GrpcAddress)

	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	clientCtx := client.Context{}.
		WithGRPCClient(conn).
		WithChainID(cfg.ChainID).
		WithCodec(cdc).
		WithInterfaceRegistry(reg).
		WithTxConfig(txCfg).
		WithBroadcastMode("sync")

	return &GrpcClient{
		conn:      conn,
		clientCtx: clientCtx,
		txConfig:  txCfg,
	}, nil
}

// Close terminates the gRPC connection.
func (gc *GrpcClient) Close() error {
	log.Printf("Closing gRPC connection to %s", gc.conn.Target())
	if gc.conn != nil {
		return gc.conn.Close()
	}
	return nil
}

// getAccountInfo queries account details (number, sequence).
func (gc *GrpcClient) getAccountInfo(senderAddress string) (*authtypes.BaseAccount, error) {
	queryClient := authtypes.NewQueryClient(gc.clientCtx)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := queryClient.Account(ctx, &authtypes.QueryAccountRequest{Address: senderAddress})
	if err != nil {
		return nil, fmt.Errorf("failed to query account info for %s: %w", senderAddress, err)
	}
	if res.Account != nil {
		log.Printf("DEBUG: Account Type URL received: %s", res.Account.GetTypeUrl())
	} else {
		log.Printf("DEBUG: Received nil account from query for %s", senderAddress)
		return nil, fmt.Errorf("account %s not found or query returned nil account", senderAddress)
	}

	var acc authtypes.AccountI
	if err := gc.clientCtx.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return nil, fmt.Errorf("failed to unpack account info: %w", err)
	}

	// Convert về *BaseAccount nếu cần thao tác sâu hơn
	baseAcc, ok := acc.(*authtypes.BaseAccount)
	if !ok {
		return nil, fmt.Errorf("unexpected account type: %T", acc)
	}

	return baseAcc, nil

	return baseAcc, nil
}

// SendTxParams holds parameters for sending a standard bank transfer.
type SendTxParams struct {
	SenderPrivateKey cryptotypes.PrivKey
	RecipientAddress string
	Amount           sdk.Coin
	GasLimit         uint64
	GasFee           sdk.Coin
	Memo             string
}

// SendBankTransferViaGrpc builds, signs, and broadcasts a bank MsgSend transaction using gRPC.
func (gc *GrpcClient) SendBankTransferViaGrpc(params SendTxParams) (*sdk.TxResponse, error) {
	senderPubKey := params.SenderPrivateKey.PubKey()
	senderAddr := sdk.AccAddress(senderPubKey.Address())

	log.Printf("Querying account info for sender: %s", senderAddr.String())
	accInfo, err := gc.getAccountInfo(senderAddr.String())
	if err != nil {
		log.Printf("[ERROR] Failed to get account info for %s: %v", senderAddr.String(), err)
		return nil, fmt.Errorf("could not get account info: %w", err)
	}
	log.Printf("Got AccountNumber: %d, Sequence: %d", accInfo.GetAccountNumber(), accInfo.GetSequence())

	// Build MsgSend
	msg := banktypes.NewMsgSend(senderAddr, sdk.MustAccAddressFromBech32(params.RecipientAddress), sdk.NewCoins(params.Amount))
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("MsgSend validation failed: %w", err)
	}

	// Build Tx
	txf := tx.Factory{}.
		WithChainID(gc.clientCtx.ChainID).
		WithTxConfig(gc.txConfig).
		WithGas(params.GasLimit).
		WithFees(params.GasFee.String()).
		WithMemo(params.Memo).
		WithAccountNumber(accInfo.GetAccountNumber()).
		WithSequence(accInfo.GetSequence())

	txBuilder, err := txf.BuildUnsignedTx(msg)
	if err != nil {
		log.Printf("[ERROR] Failed to build unsigned tx: %v", err)
		return nil, fmt.Errorf("failed to build unsigned tx: %w", err)
	}

	// Đặt signature rỗng ban đầu
	placeholderSig := signing.SignatureV2{
		PubKey: senderPubKey,
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode(gc.txConfig.SignModeHandler().DefaultMode()),
		},
		Sequence: accInfo.GetSequence(),
	}
	if err := txBuilder.SetSignatures(placeholderSig); err != nil {
		return nil, fmt.Errorf("failed to set empty signature: %w", err)
	}

	// Ký giao dịch
	log.Println("Signing transaction...")
	signerData := xauthsigning.SignerData{
		ChainID:       gc.clientCtx.ChainID,
		AccountNumber: accInfo.GetAccountNumber(),
		Sequence:      accInfo.GetSequence(),
	}

	signedSig, err := tx.SignWithPrivKey(
		signing.SignMode(gc.txConfig.SignModeHandler().DefaultMode()),
		signerData,
		txBuilder,
		params.SenderPrivateKey,
		gc.txConfig,
		accInfo.GetSequence(),
	)
	if err != nil {
		log.Printf("[ERROR] Failed to sign transaction: %v", err)
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := txBuilder.SetSignatures(signedSig); err != nil {
		return nil, fmt.Errorf("failed to set final signature: %w", err)
	}

	// Encode Tx
	txBytes, err := gc.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		log.Printf("[ERROR] Failed to encode transaction: %v", err)
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}
	log.Printf("Transaction built and signed. Encoded size: %d bytes", len(txBytes))

	// Broadcast Tx
	log.Println("Broadcasting transaction via gRPC...")

	// Xác định client tx và context cho gRPC
	txClient := txtypes.NewServiceClient(gc.conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Broadcast request
	grpcRes, err := txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC, // Thử sử dụng MODE_BLOCK nếu cần đồng bộ nhanh hơn
		TxBytes: txBytes,                                   // Đây là transaction đã được mã hóa
	})
	if err != nil {
		log.Printf("[ERROR] Failed to broadcast tx via gRPC: %v", err)
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	// Kiểm tra kết quả từ gRPC
	txResponse := grpcRes.GetTxResponse()
	if txResponse == nil {
		log.Println("[ERROR] Received nil TxResponse from gRPC broadcast")
		return nil, fmt.Errorf("received nil TxResponse from broadcast")
	}

	// Xử lý kết quả
	log.Printf("Broadcast response received. TxHash: %s, Code: %d", txResponse.TxHash, txResponse.Code)

	// Kiểm tra mã lỗi
	if txResponse.Code != 0 {
		log.Printf("[ERROR] Transaction failed on chain. Code: %d, RawLog: %s", txResponse.Code, txResponse.RawLog)
		return txResponse, fmt.Errorf("tx failed with code %d: %s", txResponse.Code, txResponse.RawLog)
	}

	// Thành công
	log.Printf("Transaction successfully broadcasted! TxHash: %s", txResponse.TxHash)
	return txResponse, nil

}

// ParseCoin converts a string to sdk.Coin.
func ParseCoin(coinStr string) (sdk.Coin, error) {
	coin, err := sdk.ParseCoinNormalized(coinStr)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("invalid coin format '%s': %w", coinStr, err)
	}
	return coin, nil
}
