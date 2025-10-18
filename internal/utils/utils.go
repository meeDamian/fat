package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/meedamian/fat/internal/types"
)

var startTS int64

func SetStartTS(ts int64) {
	startTS = ts
}

// Log writes a conversation entry to a log file
func Log(questionTS int64, logType, modelName, prompt, response string) {
	diff := time.Now().Unix() - questionTS
	diffStr := fmt.Sprintf("%04d", diff)
	filename := fmt.Sprintf("answers/%d_%s_%s_%s.log", questionTS, diffStr, logType, modelName)
	os.MkdirAll("answers", 0755)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()
	entry := fmt.Sprintf("=== PROMPT ===\n\n%s\n\n=== AGENT RESPONSE ===\n\n%s\n\n", prompt, response)
	file.WriteString(entry)
}

// LoadRates loads rates from file if <7 days, else fetch defaults
func LoadRates(ctx context.Context) map[string]types.Rate {
	file, err := os.Open("rates.json")
	if err != nil {
		return getDefaultRates()
	}
	defer file.Close()
	var rates map[string]types.Rate
	json.NewDecoder(file).Decode(&rates)
	now := time.Now().Unix()
	for _, rate := range rates {
		if now-rate.TS > 7*24*3600 { // 7 days
			return getDefaultRates()
		}
	}
	return rates
}

// getDefaultRates returns hardcoded default rates
func getDefaultRates() map[string]types.Rate {
	now := time.Now().Unix()
	return map[string]types.Rate{
		"grok-4-fast":      {TS: now, In: 0.20, Out: 0.50},
		"gpt-5-mini":       {TS: now, In: 0.25, Out: 2.00},
		"claude-3.5-haiku": {TS: now, In: 0.80, Out: 4.00},
		"gemini-2.5-flash": {TS: now, In: 0.35, Out: 1.05},
	}
}
