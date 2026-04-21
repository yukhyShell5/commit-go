package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// =============================================================================
// TYPES & INTERFACES
// =============================================================================

type AIProvider interface {
	GenerateCommit(diff string) (string, error)
	GetName() string
}

// GeminiProvider implements AIProvider for Google Gemini API or Local CLI
type GeminiProvider struct {
	APIKey string
}

// Gemini API structures
type GeminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (p *GeminiProvider) GetName() string { return "Gemini 1.5 Flash (Hybrid)" }

func (p *GeminiProvider) GenerateCommit(diff string) (string, error) {
	prompt := "Tu es un expert Git. Génère uniquement un message de commit concis (norme Conventional Commits) pour ce diff, sans markdown ni texte d'intro :\n\n" + diff

	if p.APIKey != "" {
		// Method 1: REST API
		url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + p.APIKey

		reqBody := GeminiRequest{
			Contents: []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			}{
				{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: prompt},
					},
				},
			},
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return "", err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Gemini API error (Status %d): %s", resp.StatusCode, string(body))
		}

		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return "", fmt.Errorf("erreur décodage JSON: %v", err)
		}

		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			return cleanMarkdown(geminiResp.Candidates[0].Content.Parts[0].Text), nil
		}
		return "", fmt.Errorf("aucun message généré (réponse vide)")
	}

	// Method 2: Fallback to Local CLI
	fmt.Println("API Key non trouvée, utilisation du CLI local 'gemini'...")
	cmd := exec.Command("gemini", "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("échec de l'exécution du CLI 'gemini' : %v", err)
	}

	return cleanMarkdown(string(out)), nil
}

func cleanMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 1. Supprimer les balises Markdown
		if strings.HasPrefix(trimmed, "```") || trimmed == "plaintext" {
			continue
		}

		// 2. Extraire le commit s'il est collé à un message système (MCP etc.)
		if strings.Contains(trimmed, "MCP issues detected") || strings.Contains(trimmed, "Run /mcp list") {
			if idx := strings.LastIndex(trimmed, "status."); idx != -1 {
				trimmed = strings.TrimSpace(trimmed[idx+len("status."):])
			} else {
				continue
			}
		}

		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return strings.TrimSpace(s)
	}

	// 3. Chercher la ligne qui ressemble à un commit (type: message)
	for _, line := range result {
		if strings.Contains(line, ":") {
			return line
		}
	}

	return result[0]
}

// =============================================================================
// GIT LOGIC
// =============================================================================

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	diff := out.String()
	if diff == "" {
		return "", fmt.Errorf("no staged changes found. Use 'git add' to stage files")
	}
	return diff, nil
}

func executeCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// =============================================================================
// UI LOGIC (HUH)
// =============================================================================

func handleCommitMenu(initialMessage string, diff string, provider AIProvider) {
	currentMessage := initialMessage

	for {
		var action string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Generated Commit Message").
					Description(currentMessage),
				huh.NewSelect[string]().
					Title("Action").
					Options(
						huh.NewOption("Apply", "apply"),
						huh.NewOption("Edit", "edit"),
						huh.NewOption("Regenerate", "regenerate"),
						huh.NewOption("Cancel", "cancel"),
					).
					Value(&action),
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		switch action {
		case "apply":
			if err := executeCommit(currentMessage); err != nil {
				fmt.Printf("Error committing: %v\n", err)
			}
			return
		case "edit":
			var newMessage string
			editForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Edit Commit Message").
						Value(&newMessage).
						Placeholder(currentMessage),
				),
			)
			if err := editForm.Run(); err == nil && newMessage != "" {
				currentMessage = newMessage
			}
		case "regenerate":
			fmt.Println("Regenerating...")
			msg, err := provider.GenerateCommit(diff)
			if err != nil {
				fmt.Printf("Error regenerating: %v\n", err)
			} else {
				currentMessage = msg
			}
		case "cancel":
			fmt.Println("Commit cancelled.")
			return
		}
	}
}

// =============================================================================
// COBRA COMMANDS
// =============================================================================

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
		diff, err := getGitDiff()
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

		handleCommitMenu(message, diff, provider)
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

func GetProvider() (AIProvider, error) {
	apiKey := viper.GetString("api_key")

	// Nettoyage au cas où la clé est restée sur "test_key"
	if apiKey == "test_key" {
		apiKey = ""
	}

	return &GeminiProvider{
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
