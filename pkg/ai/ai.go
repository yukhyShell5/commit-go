package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// AIProvider defines the interface for different AI backends
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
			return CleanMarkdown(geminiResp.Candidates[0].Content.Parts[0].Text), nil
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

	return CleanMarkdown(string(out)), nil
}

func CleanMarkdown(s string) string {
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
