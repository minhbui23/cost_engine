
# SoCone Payment network
```sh
make build
make resets
make start
```

### Usage of CLI Commands

To Start a stream payment

cmd :

 `streampayd tx streampay stream-send [recipient] [amount] --duration <stream-duration> --delayed --chain-id <chain-id> --from <key>`

To start a continuous payment stream
```bash
docker exec -it validator1 sh
streampayd keys list  --keyring-backend test
streampayd keys show validator1  --keyring-backend test
SENDER_ADDR=$(streampayd keys show validator1 --keyring-backend test | sed -n 's/^.*address: *\([^ ]*\).*$/\1/p'
)
echo $SENDER_ADDR

RECEIVER_ADDR=$(streampayd keys show validator2 --keyring-backend test | sed -n 's/^.*address: *\([^ ]*\).*$/\1/p'
)
echo $RECEIVER_ADDR

streampayd tx streampay stream-send \
  $RECEIVER_ADDR \
  10000soc   \
  --payment-fee 100soc \
  --duration 180s  \
  --chain-id socp-chain   \
  --from $SENDER_ADDR \
  --keyring-backend test \
  --yes

streampayd q streampay stream-payments --chain-id socp-chain 

# Send
streampayd keys show validator0  --keyring-backend test
streampayd tx bank send $SENDER_ADDR  $RECEIVER_ADDR 2000soc --chain-id socp-chain  --from validator0 --keyring-backend test  #--fees 20soc

streampayd query bank balances $RECEIVER_ADDR --chain-id socp-chain 

http://127.0.0.1:26657


# Claim
streampayd tx streampay claim sp1 --chain-id socp-chain --from validator2 --keyring-backend test --yes
```
Use --delayed flag for delayed payments.

# Reference
- https://github.com/DaevMithran/tendermint-load-test