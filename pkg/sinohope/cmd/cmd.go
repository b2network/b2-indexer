package cmd

import (
	"errors"

	"github.com/google/uuid"
	"github.com/sinohope/sinohope-golang-sdk/common"
	"github.com/sinohope/sinohope-golang-sdk/core/sdk"
	"github.com/spf13/cobra"
)

var (
	FlagBaseURL          = "baseUrl"
	FlagPrivateKey       = "privateKey"
	FlagVaultId          = "vaultId"
	FlagCreateWalletName = "createWalletName"
	FlagWalletId         = "walletId"
	FlagChainSymbol      = "chainSymbol"
)

var (
	FakePrivateKey = ""
	BaseURL        = "https://api.sinohope.com"
)

func Sinohope() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sinohope",
		Short: "sinohope command",
		Long:  `sinohope command`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			key, err := cmd.Flags().GetString(FlagPrivateKey)
			if err != nil {
				return err
			}
			url, err := cmd.Flags().GetString(FlagBaseURL)
			if err != nil {
				return err
			}
			BaseURL = url
			FakePrivateKey = key
			return nil
		},
	}
	cmd.AddCommand(
		listVault(),
		listSupportedChainAndCoins(),
		createWallet(),
		genAddress(),
	)
	cmd.PersistentFlags().String(FlagBaseURL, "https://api.sinohope.com", "sinohope base url")
	cmd.PersistentFlags().String(FlagPrivateKey, "", "fakePrivateKey")
	cmd.PersistentFlags().String(FlagVaultId, "", "Sinohope VaultId")
	cmd.PersistentFlags().String(FlagWalletId, "", "Sinohope wallet id")
	cmd.PersistentFlags().String(FlagChainSymbol, "BTC", "Sinohope ChainSymbol")
	return cmd
}

func listVault() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vaults",
		Short: "list usable vaults",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := sdk.NewCommonAPI(BaseURL, FakePrivateKey)
			if err != nil {
				return err
			}

			var vaults []*common.WaaSVaultInfoData
			if vaults, err = c.GetVaults(); err != nil {
				return err
			} else {
				for _, v := range vaults {
					for _, v2 := range v.VaultInfoOfOpenApiList {
						cmd.Printf("vaultId: %v, vaultName: %v, authorityType: %v, createTime: %v\n",
							v2.VaultId, v2.VaultName, v2.AuthorityType, v2.CreateTime)
					}
				}
			}
			return nil
		},
	}
	return cmd
}

func listSupportedChainAndCoins() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coins",
		Short: "list usable coins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := sdk.NewCommonAPI(BaseURL, FakePrivateKey)
			if err != nil {
				return err
			}
			var supportList []*common.WaasChainData
			if supportList, err = c.GetSupportedChains(); err != nil {
				return err
			} else {
				cmd.Printf("supported chains:\n")
				for _, v := range supportList {
					cmd.Printf("chainName: %v, chainSymbol: %v\n", v.ChainName, v.ChainSymbol)
				}
			}
			cmd.Println()
			var supportCoins []*common.WaaSCoinDTOData
			for _, v := range supportList {
				param := &common.WaasChainParam{
					ChainSymbol: v.ChainSymbol,
				}
				if supportCoins, err = c.GetSupportedCoins(param); err != nil {
					return err
				} else {
					cmd.Printf("supported coins:\n")
					for _, v := range supportCoins {
						cmd.Printf("assetName: %v, assetId: %v, assetDecimal: %v\n",
							v.AssetName, v.AssetId, v.AssetDecimal)
					}
				}
			}
			return nil
		},
	}
	return cmd
}

func createWallet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-wallet",
		Short: "create wallet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := sdk.NewAccountAndAddressAPI(BaseURL, FakePrivateKey)
			if err != nil {
				return err
			}
			vaultId, err := cmd.Flags().GetString(FlagVaultId)
			if err != nil {
				return err
			}
			walletName, err := cmd.Flags().GetString(FlagCreateWalletName)
			if err != nil {
				return err
			}
			if vaultId == "" || walletName == "" {
				return errors.New("vaultId or walletName is empty")
			}
			requestId := genRequestId()

			cmd.Println("VaultId:", vaultId)
			cmd.Println("WalletName:", walletName)
			cmd.Println("RequestId:", requestId)

			cmd.Println()

			var walletInfo []*common.WaaSWalletInfoData
			if walletInfo, err = m.CreateWallets(&common.WaaSCreateBatchWalletParam{
				VaultId:   vaultId,
				RequestId: requestId,
				Count:     1,
				WalletInfos: []common.WaaSCreateWalletInfo{
					{
						WalletName: walletName,
					},
				},
			}); err != nil {
				return err
			} else {
				cmd.Println("create wallet success")
				for _, v := range walletInfo {
					cmd.Printf("walletId:%v, walletName:%v\n", v.WalletId, v.WalletName)
				}
			}
			return nil
		},
	}
	cmd.Flags().String(FlagCreateWalletName, "", "The name of the wallet to be created")
	return cmd
}

func genAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-address",
		Short: "Generate address",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := sdk.NewAccountAndAddressAPI(BaseURL, FakePrivateKey)
			if err != nil {
				return err
			}
			vaultId, err := cmd.Flags().GetString(FlagVaultId)
			if err != nil {
				return err
			}

			walletId, err := cmd.Flags().GetString(FlagWalletId)
			if err != nil {
				return err
			}

			chainSymbol, err := cmd.Flags().GetString(FlagChainSymbol)
			if err != nil {
				return err
			}

			requestId := genRequestId()

			cmd.Println("VaultId:", vaultId)
			cmd.Println("WalletId:", walletId)
			cmd.Println("RequestId:", requestId)
			cmd.Println()

			var walletInfo []*common.WaaSAddressInfoData
			if walletInfo, err = m.GenerateChainAddresses(&common.WaaSGenerateChainAddressParam{
				RequestId:   requestId,
				VaultId:     vaultId,
				WalletId:    walletId,
				ChainSymbol: chainSymbol,
			}); err != nil {
				return err
			} else {
				for _, v := range walletInfo {
					cmd.Printf("address:%v, encoding:%v, hdPath:%v, pubkey:%v\n", v.Address, v.Encoding, v.HdPath, v.Pubkey)
				}
			}
			return nil
		},
	}

	return cmd
}

func genRequestId() string {
	return uuid.New().String()
}
