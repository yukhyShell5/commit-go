package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/commit-go/pkg/ai"
	"github.com/user/commit-go/pkg/git"
	"github.com/user/commit-go/pkg/ui"
)

var rootCmd = &cobra.Command{
	Use:   "commit-go",
	Short: "AI-powered Git commit message generator",
	Run: func(cmd *cobra.Command, args []string) {
		commitCmd.Run(cmd, args)
	},
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate and apply a commit message",
	Run: func(cmd *cobra.Command, args []string) {
		diff, err := git.GetGitDiff()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		provider, err := GetProvider()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Generating message using %s...\n", provider.GetName())
		message, err := provider.GenerateCommit(diff)
		if err != nil {
			fmt.Printf("Error generating message: %v\n", err)
			return
		}

		ui.HandleCommitMenu(message, diff, provider)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]
		viper.Set(key, value)
		if err := viper.WriteConfig(); err != nil {
			if err := viper.SafeWriteConfig(); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				return
			}
		}
		fmt.Printf("Configured %s = %s\n", key, value)
	},
}

func GetProvider() (ai.AIProvider, error) {
	apiKey := viper.GetString("api_key")

	// Nettoyage au cas où la clé est restée sur "test_key"
	if apiKey == "test_key" {
		apiKey = ""
	}

	return &ai.GeminiProvider{
		APIKey: apiKey,
	}, nil
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	configDir := filepath.Join(home, ".config", "commit-go")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}

	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		// Ignore if config not found
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	commitCmd.Flags().StringP("provider", "p", "gemini", "AI provider to use")
	commitCmd.Flags().BoolP("debug", "d", false, "Display debug logs")

	viper.BindPFlag("provider", commitCmd.Flags().Lookup("provider"))
	viper.BindPFlag("debug", commitCmd.Flags().Lookup("debug"))

	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
