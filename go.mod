module github.com/nysa-network/cosmos-exporter

go 1.16

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

// replace google.golang.org/grpc => google.golang.org/grpc v1.33.2

replace github.com/tendermint/tendermint => github.com/informalsystems/tendermint v0.34.26

require (
	github.com/cosmos/cosmos-sdk v0.46.10
	github.com/google/uuid v1.3.0
	github.com/prometheus/client_golang v1.14.0
	github.com/rs/zerolog v1.27.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.1
	github.com/tendermint/tendermint v0.34.26
	google.golang.org/grpc v1.50.1
)
