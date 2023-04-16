package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	tmrpc "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
)

type TendermintError struct {
	Message string
}

func (e *TendermintError) Error() string {
	return e.Message
}

func GenerateMetrics(registry *prometheus.Registry, logger zerolog.Logger) error {

	client, err := tmrpc.New(TendermintRPC, "/websocket")

	if err != nil {
		return fmt.Errorf("could not create tendermint client")
	}

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
			Help:        "Latest Block Time Difference",
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

	registry.MustRegister(tendermintLatestBlockHeightGauge)
	registry.MustRegister(tendermintLatestBlockTimeGauge)
	registry.MustRegister(tendermintLatestBlockTimeDiffGauge)
	registry.MustRegister(tendermintPeersGauge)

	// Node statistics

	nodeStatusResponse, err := client.Status(context.Background())

	if err != nil {
		return fmt.Errorf("could not query tendermint node status")
	}

	timeDifference := time.Since(nodeStatusResponse.SyncInfo.LatestBlockTime)

	tendermintLatestBlockHeightGauge.Set(float64(nodeStatusResponse.SyncInfo.LatestBlockHeight))
	tendermintLatestBlockTimeGauge.Set(float64(nodeStatusResponse.SyncInfo.LatestBlockTime.Unix()))
	tendermintLatestBlockTimeDiffGauge.Set(timeDifference.Seconds())

	// NetInfo statistics

	netInfoResponse, err := client.NetInfo(context.Background())

	if err != nil {
		return fmt.Errorf("could not query tendermint net infromation")
	}

	tendermintPeersGauge.Set(float64(netInfoResponse.NPeers))

	return nil

}

func TendermintHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {

	registry := prometheus.NewRegistry()

	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	err := GenerateMetrics(registry, sublogger)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		sublogger.Err(err).Msg("Cannot generate metrics")
	} else {
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		})
		h.ServeHTTP(w, r)
	}

	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}
