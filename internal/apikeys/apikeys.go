package apikeys

import (
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/types"
)

// familyEnvVars maps model family IDs to their environment variable names
var familyEnvVars = map[string]string{
	models.Grok:     "GROK_KEY",
	models.GPT:      "GPT_KEY",
	models.Claude:   "CLAUDE_KEY",
	models.Gemini:   "GEMINI_KEY",
	models.DeepSeek: "DEEPSEEK_KEY",
	models.Mistral:  "MISTRAL_KEY",
}

// Load loads API keys from environment variables, .env file, and keys.json
// and assigns them to the provided model infos
func Load(modelInfos []*types.ModelInfo) {
	// Try environment variables first
	for _, mi := range modelInfos {
		if envVar, ok := familyEnvVars[mi.ID]; ok {
			key := os.Getenv(envVar)
			if key != "" {
				mi.APIKey = key
				continue
			}
		}
	}

	// Try .env file
	godotenv.Load()
	for _, mi := range modelInfos {
		if mi.APIKey != "" {
			continue // Already loaded from env
		}
		if envVar, ok := familyEnvVars[mi.ID]; ok {
			key := os.Getenv(envVar)
			if key != "" {
				mi.APIKey = key
				continue
			}
		}
	}

	// Try keys.json (uses family ID as key)
	if file, err := os.Open("keys.json"); err == nil {
		defer file.Close()
		var keys map[string]string
		json.NewDecoder(file).Decode(&keys)
		for _, mi := range modelInfos {
			if mi.APIKey != "" {
				continue // Already loaded
			}
			if key, ok := keys[mi.ID]; ok {
				mi.APIKey = key
			}
		}
	}
}

// GetForFamily retrieves the API key for a specific model family
func GetForFamily(familyID string) string {
	envVar, ok := familyEnvVars[familyID]
	if !ok {
		return ""
	}

	// Try environment variable
	if key := os.Getenv(envVar); key != "" {
		return key
	}

	// Try keys.json
	if file, err := os.Open("keys.json"); err == nil {
		defer file.Close()
		var keys map[string]string
		if json.NewDecoder(file).Decode(&keys) == nil {
			if key, ok := keys[familyID]; ok {
				return key
			}
		}
	}

	return ""
}
