package main

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nysa-network/cosmos-exporter/pkg/cosmosdirectory"
	"google.golang.org/grpc"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


func queryGovCount(conn *grpc.ClientConn) (int, error) {
	// Handle v1beta1
	govv1beta1Client := govv1beta1.NewQueryClient(conn)
	{
		proposals, err := govv1beta1Client.Proposals(context.Background(), &govv1beta1.QueryProposalsRequest{
			ProposalStatus: govv1beta1.StatusVotingPeriod,
		})
		if err != nil {
			if !strings.Contains(err.Error(), `invalid type: can't convert a gov/v1 Proposal to gov/v1beta1 Proposal`) {
				return 0, err
			}
		} else if err == nil {
			pp := proposals.GetProposals()
			return len(pp), nil
		}
	}

	govv1Client := govv1.NewQueryClient(conn)
	{
		proposals, err := govv1Client.Proposals(context.Background(), &govv1.QueryProposalsRequest{
			ProposalStatus: govv1.StatusVotingPeriod,
		})
		if err != nil {
			return 0, err
		}
		pp := proposals.GetProposals()
		return len(pp), nil
	}

	return 0, nil
}



func (s *service) GeneralHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	generalBondedTokensGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_general_bonded_tokens",
			Help:        "Bonded tokens",
			ConstLabels: ConstLabels,
		},
	)

	generalNotBondedTokensGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_general_not_bonded_tokens",
			Help:        "Not bonded tokens",
			ConstLabels: ConstLabels,
		},
	)

	generalCommunityPoolGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "cosmos_general_community_pool",
			Help:        "Community pool",
			ConstLabels: ConstLabels,
		},
		[]string{"denom"},
	)

	generalSupplyTotalGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "cosmos_general_supply_total",
			Help:        "Total supply",
			ConstLabels: ConstLabels,
		},
		[]string{"denom"},
	)

	generalInflationGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_general_inflation",
			Help:        "Total supply",
			ConstLabels: ConstLabels,
		},
	)

	generalAnnualProvisions := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "cosmos_general_annual_provisions",
			Help:        "Annual provisions",
			ConstLabels: ConstLabels,
		},
		[]string{"denom"},
	)

	generalLatestBlockHeight := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_latest_block_height",
			Help:        "Latest block height",
			ConstLabels: ConstLabels,
		},
	)

	generalTokenPrice := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_token_price",
			Help:        "Cosmos token price",
			ConstLabels: ConstLabels,
		},
	)

	paramsGovVotingPeriodProposals := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_gov_voting_period_proposals",
			Help:        "Voting period proposals",
			ConstLabels: ConstLabels,
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(generalBondedTokensGauge)
	registry.MustRegister(generalNotBondedTokensGauge)
	registry.MustRegister(generalCommunityPoolGauge)
	registry.MustRegister(generalSupplyTotalGauge)
	registry.MustRegister(generalInflationGauge)
	registry.MustRegister(generalAnnualProvisions)
	registry.MustRegister(generalLatestBlockHeight)
	registry.MustRegister(generalTokenPrice)
	registry.MustRegister(paramsGovVotingPeriodProposals)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		chain, err := cosmosdirectory.GetChainByChainID(ChainID)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get chain informations")
			return
		}

		price := chain.GetPriceUSD()
		generalTokenPrice.Set(price)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying base pool")

		queryStart := time.Now()

		status, err := s.tmRPC.Status(context.Background())
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not status")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying rpc status")

		generalLatestBlockHeight.Set(float64(status.SyncInfo.LatestBlockHeight))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying staking pool")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		response, err := stakingClient.Pool(
			context.Background(),
			&stakingtypes.QueryPoolRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get staking pool")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying staking pool")

		bondedTokensBigInt := response.Pool.BondedTokens.BigInt()
		bondedTokens, _ := new(big.Float).SetInt(bondedTokensBigInt).Float64()

		notBondedTokensBigInt := response.Pool.NotBondedTokens.BigInt()
		notBondedTokens, _ := new(big.Float).SetInt(notBondedTokensBigInt).Float64()

		generalBondedTokensGauge.Set(bondedTokens)
		generalNotBondedTokensGauge.Set(notBondedTokens)
		//fmt.Println("response: ", response.Pool.BondedTokens)
		//generalBondedTokensGauge.Set(float64(response.Pool.BondedTokens.Int64()))
		//generalNotBondedTokensGauge.Set(float64(response.Pool.NotBondedTokens.Int64()))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying distribution community pool")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(s.grpcConn)
		response, err := distributionClient.CommunityPool(
			context.Background(),
			&distributiontypes.QueryCommunityPoolRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get distribution community pool")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying distribution community pool")

		for _, coin := range response.Pool {
			if value, err := strconv.ParseFloat(coin.Amount.String(), 64); err != nil {
				sublogger.Error().
					Err(err).
					Msg("Could not get community pool coin")
			} else {
				generalCommunityPoolGauge.With(prometheus.Labels{
					"denom": Denom,
				}).Set(value / DenomCoefficient)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying bank total supply")
		queryStart := time.Now()

		bankClient := banktypes.NewQueryClient(s.grpcConn)
		response, err := bankClient.TotalSupply(
			context.Background(),
			&banktypes.QueryTotalSupplyRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get bank total supply")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying bank total supply")

		for _, coin := range response.Supply {
			if value, err := strconv.ParseFloat(coin.Amount.String(), 64); err != nil {
				sublogger.Error().
					Err(err).
					Msg("Could not get total supply")
			} else {
				generalSupplyTotalGauge.With(prometheus.Labels{
					"denom": Denom,
				}).Set(value / DenomCoefficient)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying inflation")
		queryStart := time.Now()

		mintClient := minttypes.NewQueryClient(s.grpcConn)
		response, err := mintClient.Inflation(
			context.Background(),
			&minttypes.QueryInflationRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get inflation")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying inflation")

		if value, err := strconv.ParseFloat(response.Inflation.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get inflation")
		} else {
			generalInflationGauge.Set(value)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying annual provisions")
		queryStart := time.Now()

		mintClient := minttypes.NewQueryClient(s.grpcConn)
		response, err := mintClient.AnnualProvisions(
			context.Background(),
			&minttypes.QueryAnnualProvisionsRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get annual provisions")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying annual provisions")

		if value, err := strconv.ParseFloat(response.AnnualProvisions.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get annual provisions")
		} else {
			generalAnnualProvisions.With(prometheus.Labels{
				"denom": Denom,
			}).Set(value / DenomCoefficient)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global gov params")

		govCount, err := queryGovCount(s.grpcConn)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get active proposals")
			govCount = -1
		}

		paramsGovVotingPeriodProposals.Set(float64(govCount))
	}()

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/general").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}
