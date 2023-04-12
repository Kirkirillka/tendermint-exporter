package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	tmrpc "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
)

var (
	ConfigPath string

	Denom         string
	ListenAddress string
	NodeAddress   string
	TendermintRPC string
	LogLevel      string
	JSONOutput    bool
	Limit         uint64

	ChainID     string
	Moniker     string
	NodeID      string
	ConstLabels map[string]string
)

var log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

var rootCmd = &cobra.Command{
	Use:  "terdermint-exporter",
	Long: "Scrape the data from Tendermint node, for example, Cosmos gaiad.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if ConfigPath == "" {
			return nil
		}

		viper.SetConfigFile(ConfigPath)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Info().Err(err).Msg("Error reading config file")
				return err
			}
		}

		// Credits to https://carolynvanslyck.com/blog/2020/08/sting-of-the-viper/
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && viper.IsSet(f.Name) {
				val := viper.Get(f.Name)
				if err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
					log.Fatal().Err(err).Msg("Could not set flag")
				}
			}
		})

		return nil
	},
	Run: Execute,
}

func Execute(cmd *cobra.Command, args []string) {
	logLevel, err := zerolog.ParseLevel(LogLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse log level")
	}

	if JSONOutput {
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(logLevel)

	log.Info().
		Str("--listen-address", ListenAddress).
		Str("--node", NodeAddress).
		Str("--log-level", LogLevel).
		Msg("Started with following parameters")

	config := sdk.GetConfig()
	config.Seal()

	grpcConn, err := grpc.Dial(
		NodeAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to gRPC node")
	}

	setStaticLabels()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		TendermintHandler(w, r, grpcConn)
	})

	log.Info().Str("address", ListenAddress).Msg("Listening")
	err = http.ListenAndServe(ListenAddress, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start application")
	}
}

func setStaticLabels() {
	client, err := tmrpc.New(TendermintRPC, "/websocket")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create Tendermint client")
	}

	status, err := client.Status(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Could not query Tendermint status")
	}

	log.Info().Str("network", status.NodeInfo.Network).Msg("Got status from Tendermint")
	ChainID = status.NodeInfo.Network
	Moniker = status.NodeInfo.Moniker
	NodeID = string(status.NodeInfo.ID())
	ConstLabels = map[string]string{
		"chain_id": ChainID,
		"moniker":  Moniker,
		"node_id":  NodeID,
	}
}

func main() {
	rootCmd.PersistentFlags().StringVar(&ConfigPath, "config", "", "Config file path")
	rootCmd.PersistentFlags().StringVar(&ListenAddress, "listen-address", ":9301", "The address this exporter would listen on")
	rootCmd.PersistentFlags().StringVar(&NodeAddress, "node", "localhost:9090", "RPC node address")
	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "info", "Logging level")
	rootCmd.PersistentFlags().Uint64Var(&Limit, "limit", 1000, "Pagination limit for gRPC requests")
	rootCmd.PersistentFlags().StringVar(&TendermintRPC, "tendermint-rpc", "http://localhost:26657", "Tendermint RPC address")
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "Output logs as JSON")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Could not start application")
	}
}
