/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"github.com/arya-analytics/aspen"
	aspentransport "github.com/arya-analytics/aspen/transport/grpc"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/gorp"
	grpcx "github.com/arya-analytics/x/grpc"
	xsignal "github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"os"
	"os/signal"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := configureLogging()
		if err != nil {
			return err
		}

		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, os.Interrupt)

		ctx, cancel := xsignal.WithCancel(cmd.Context())
		defer cancel()

		// Perform the rest of the startup within a separate goroutine
		// we can properly handle signal interrupts.
		ctx.Go(func(ctx xsignal.Context) (err error) {
			defer cancel()

			// Set up the tracing backend.
			exp := configureObservability()

			// Set up the storage backend.
			store, err := storage.Open(newStorageConfig(exp, logger))
			defer func() {
				err = errors.CombineErrors(err, store.Close())
			}()

			if err != nil {
				return err
			}

			// Open our base listener and start serving requests.
			// We'll defer closing the listener here which signals to our grpc servers
			// to shut down.
			lis, err := openListener()
			defer func() {
				err = errors.CombineErrors(err, lis.Close())
			}()
			if err != nil {
				return err
			}
			// Open our grpc and http listeners.
			grpcL, httpL, err := configureCmux(ctx, lis)
			if err != nil {
				return err
			}

			// Open our server and pool.
			rpcServer, rpcPool, err := openRPC(ctx, grpcL)
			if err != nil {
				return err
			}

			// Join the cluster.
			aspenDB, err := joinCluster(ctx, store, exp, logger, rpcPool, rpcServer)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.CombineErrors(err, aspenDB.Close())
			}()

			// Open up the gorp DB for storing all of our go types.
			gorpDB := gorp.Wrap(aspenDB)

			// Configure the cluster ontology.
			otg, err := configureOntology(gorpDB)
			if err != nil {
				return err
			}

			// 1. Open our channel distribution layer.
			// 2. Open our segment distribution layer.
			// 3. Open our access legislator and enforcer.
			// 4. Open our authentication service.
			// 5. Open the fiber server.
			// 6. Open our UI.
			// 7. Bind our
			//		Access Services
			//		Authentication Services
			//		Ontology Services
			// 		UI Services
			// 	To our fiber server.
			// 8. Start our fiber server.

			return nil
		})

		select {
		case <-sigC:
			cancel()
		case <-ctx.Stopped():
		}

		return ctx.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringP(
		"listen-address",
		"l",
		"127.0.0.1:9090",
		`
			`,
	)

	startCmd.Flags().StringSliceP(
		"peer-addresses",
		"p",
		nil,
		`
			Addresses of additional peers in the cluster.
		`,
	)

	startCmd.Flags().StringP(
		"data",
		"d",
		"delta-data",
		`
			Dirname where Delta will store its data.
		`,
	)

	if err := viper.BindPFlags(startCmd.Flags()); err != nil {
		panic(err)
	}
}

func newStorageConfig(
	exp alamos.Experiment,
	logger *zap.Logger,
) storage.Config {
	return storage.Config{
		MemBacked:  viper.GetBool("mem"),
		Dirname:    viper.GetString("data"),
		Logger:     logger.Named("storage"),
		Experiment: exp,
	}
}

func configureLogging() (*zap.Logger, error) {
	return zap.NewProduction()
}

var (
	rootExperimentKey = "experiment"
)

func configureObservability() alamos.Experiment {
	debug := viper.GetBool("debug")
	var opts alamos.Option
	if debug {
		opts = alamos.WithFilters(alamos.LevelFilterAll{})
	} else {
		opts = alamos.WithFilters(alamos.LevelFilterThreshold{
			Level: alamos.Production,
		})
	}
	return alamos.New(rootExperimentKey, opts...)
}

func openRPC(
	ctx xsignal.Context,
	lis net.Listener,
) (*grpc.Server, *grpcx.Pool, error) {
	s := grpc.NewServer()
	pool := grpcx.NewPool(grpc.WithTransportCredentials(insecure.NewCredentials()))
	ctx.Go(func(ctx xsignal.Context) error { return s.Serve(lis) })
	return s, pool, nil
}

func openListener() (net.Listener, error) {
	return net.Listen("tcp", viper.GetString("listen-address"))
}

func configureCmux(
	ctx xsignal.Context,
	lis net.Listener,
) (net.Listener, net.Listener, error) {
	m := cmux.New(lis)
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())
	ctx.Go(func(ctx xsignal.Context) error { return m.Serve() })
	return grpcL, httpL, nil
}

func joinCluster(
	ctx context.Context,
	storage storage.Storage,
	exp alamos.Experiment,
	logger *zap.Logger,
	pool *grpcx.Pool,
	server *grpc.Server,
) (aspen.DB, error) {

	peerAddresses, err := parsePeerAddresses()
	if err != nil {
		return nil, err
	}

	return aspen.Open(
		ctx,
		storage.Cfg.Dirname,
		address.Address(viper.GetString("listen-address")),
		peerAddresses,
		aspen.WithEngine(storage.KV),
		aspen.WithExperiment(exp),
		aspen.WithLogger(logger.Named("aspen").Sugar()),
		aspen.WithTransport(aspentransport.NewWithPoolAndServer(pool, server)),
	)
}

func parsePeerAddresses() ([]address.Address, error) {
	peerStrings := viper.GetStringSlice("peer-addresses")
	peerAddresses := make([]address.Address, len(peerStrings))
	for i, listenString := range peerStrings {
		peerAddresses[i] = address.Address(listenString)
	}
	return peerAddresses, nil
}

func configureOntology(db *gorp.DB) (*ontology.Ontology, error) {
	return ontology.Open(db)
}
