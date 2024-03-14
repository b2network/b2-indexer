package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"os"

	"github.com/b2network/b2-indexer/internal/server"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/spf13/cobra"
)

const (
	FlagHome = "home"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "b2-indexer",
		Short: "index tx",
		Long:  "b2-indexer is a application that index bitcoin tx",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, server.ServerContextKey, server.NewDefaultContext())
			cmd.SetContext(ctx)
		},
	}

	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(generateECDSAPrivateKey())
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(startHTTPServer())
	return rootCmd
}

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start index tx service",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			home, err := cmd.Flags().GetString(FlagHome)
			if err != nil {
				return err
			}
			return server.InterceptConfigsPreRunHandler(cmd, home)
		},
		Run: func(cmd *cobra.Command, _ []string) {
			err := server.Start(GetServerContextFromCmd(cmd), cmd)
			if err != nil {
				log.Error("start index tx service failed")
			}
		},
	}
	cmd.Flags().String(FlagHome, "", "The application home directory")
	return cmd
}

func generateECDSAPrivateKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-ecdsa-key",
		Short: "generate ECDSA PrivateKey",
		RunE: func(cmd *cobra.Command, _ []string) error {
			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return err
			}
			pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
			if err != nil {
				return err
			}
			pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
			if err != nil {
				return err
			}
			cmd.Println("pubKey:", hex.EncodeToString(pubKeyBytes))
			cmd.Println("privateKey:", hex.EncodeToString(pkcs8Bytes))
			return nil
		},
	}
	return cmd
}

// GetServerContextFromCmd returns a Context from a command or an empty Context
// if it has not been set.
func GetServerContextFromCmd(cmd *cobra.Command) *server.Context {
	if v := cmd.Context().Value(server.ServerContextKey); v != nil {
		serverCtxPtr := v.(*server.Context)
		return serverCtxPtr
	}

	return server.NewDefaultContext()
}

func startHTTPServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "start http service",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			home, err := cmd.Flags().GetString(FlagHome)
			if err != nil {
				return err
			}
			return server.HTTPConfigsPreRunHandler(cmd, home)
		},
		Run: func(cmd *cobra.Command, _ []string) {
			err := server.Run(GetServerContextFromCmd(cmd))
			if err != nil {
				log.Error("start http service failed")
			}
		},
	}
	cmd.Flags().String(FlagHome, "", "The application home directory")
	return cmd
}
