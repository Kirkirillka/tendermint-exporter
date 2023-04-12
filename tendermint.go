package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	tmrpc "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
)

func TendermintHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	requestStart := time.Now()

	client, err := tmrpc.New(TendermintRPC, "/websocket")

	if err != nil {
		log.Fatal().Err(err).Msg("Could not create Tendermint client")
	}

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	tendermintLatestBlockHeightGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tendermint_latest_block_height",
			Help:        "Latest Block ID",
			ConstLabels: ConstLabels,
		},
	)

	tendermintLatestBlockTimeGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tendermint_latest_block_time",
			Help:        "Latest Block Time",
			ConstLabels: ConstLabels,
		},
	)

	tendermintLatestBlockTimeDiffGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tendermint_latest_block_time_diff",
			Help:        "Latest Block Time Differnece",
			ConstLabels: ConstLabels,
		},
	)

	tendermintPeersGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tendermint_peers",
			Help:        "Number of connected peers",
			ConstLabels: ConstLabels,
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(tendermintLatestBlockHeightGauge)
	registry.MustRegister(tendermintLatestBlockTimeGauge)
	registry.MustRegister(tendermintLatestBlockTimeDiffGauge)
	registry.MustRegister(tendermintPeersGauge)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started calculating Node statistics")
		queryStart := time.Now()

		response, err := client.Status(context.Background())

		if err != nil {
			log.Fatal().Err(err).Msg("Could not query Tendermint Node Status")
			return
		}

		log.Info().Str("node_id", string(response.NodeInfo.DefaultNodeID)).Msg("Got node information from Tendermint")

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying staking pool")

		timeDifference := time.Since(response.SyncInfo.LatestBlockTime)

		tendermintLatestBlockHeightGauge.Set(float64(response.SyncInfo.LatestBlockHeight))
		tendermintLatestBlockTimeGauge.Set(float64(response.SyncInfo.LatestBlockTime.Unix()))
		tendermintLatestBlockTimeDiffGauge.Set(timeDifference.Seconds())

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started calculating Peers")
		queryStart := time.Now()

		response, err := client.NetInfo(context.Background())

		if err != nil {
			log.Fatal().Err(err).Msg("Could not query Tendermint Net Infromation")
			return
		}

		log.Info().Str("peers", fmt.Sprint(response.NPeers)).Msg("Got node information from Tendermint")

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying staking pool")

		tendermintPeersGauge.Set(float64(response.NPeers))

	}()

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}
