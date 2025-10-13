package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/meedamian/fat/internal/constants"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/types"
)

// Log appends a log entry to the model's log file
func Log(modelName, prompt, response string) {
	ts := time.Now().Unix()
	filename := fmt.Sprintf("answers/%d_%s.log", ts, modelName)
	os.MkdirAll("answers", 0755)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()
	entry := fmt.Sprintf("[%d] Prompt: %s\nResponse: %s\n\n", ts, prompt, response)
	file.WriteString(entry)
}

// EstTokens estimates token count (rough: len/4 + buffer)
func EstTokens(text string) int64 {
	return int64(len(text)/4 + constants.EstTokensBuffer)
}

// SummarizeChunk calls Grok to summarize a chunk of text
func SummarizeChunk(chunk string) (string, error) {
	// Use CallModel for grok
	mi := models.ModelMap["A"] // Assuming "A" is grok
	resp, _, _, err := models.CallModel(context.Background(), mi, "Summarize this text concisely: "+chunk, []string{})
	if err != nil {
		return "", fmt.Errorf("summarize failed: %w", err)
	}
	return resp.Refined, nil
}

// BuildContext builds the context string from question and history
func BuildContext(question string, history types.History) string {
	context := "Question: " + question + "\n\nHistory:\n"
	for id, responses := range history {
		context += fmt.Sprintf("Model %s:\n", id)
		for _, resp := range responses {
			context += fmt.Sprintf("Refined: %s\nSuggestions: %v\n", resp.Refined, resp.Suggestions)
		}
	}
	return context
}

// CapContext caps the context at 80% max tokens, summarizing if needed
func CapContext(context string, maxTokens int64) string {
	est := EstTokens(context)
	if est < int64(float64(maxTokens)*constants.ContextCapRatio) {
		return context
	}
	// Summarize older parts
	lines := strings.Split(context, "\n")
	half := len(lines) / 2
	old := strings.Join(lines[:half], "\n")
	new := strings.Join(lines[half:], "\n")
	summary, _ := SummarizeChunk(old)
	return summary + "\n" + new
}

// FetchRates fetches rates from HTTP and saves to file
func FetchRates(ctx context.Context) (map[string]types.Rate, error) {
	rates := make(map[string]types.Rate)
	pricingURLs := map[string]string{
		"grok-4-fast":      "https://x.ai/api",
		"gpt-5-mini":       "https://openai.com/pricing",
		"claude-3.5-haiku": "https://www.anthropic.com/pricing/api",
		"gemini-2.5-flash": "https://ai.google.dev/pricing",
	}
	client := &http.Client{Timeout: constants.HTTPTimeoutSeconds * time.Second}
	for name, url := range pricingURLs {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		text := string(body)
		var in, out float64
		// Simple regex parse (stub, need specific parsing)
		re := regexp.MustCompile(`input.*?([\d.]+).*?output.*?([\d.]+)`)
		matches := re.FindStringSubmatch(text)
		if len(matches) == 3 {
			in, _ = strconv.ParseFloat(matches[1], 64)
			out, _ = strconv.ParseFloat(matches[2], 64)
		} else {
			// Fallback to defaults
			switch name {
			case "grok-4-fast":
				in, out = 0.20, 0.50
			case "gpt-5-mini":
				in, out = 0.25, 2.00
			case "claude-3.5-haiku":
				in, out = 0.80, 4.00
			case "gemini-2.5-flash":
				in, out = 0.35, 1.05
			}
		}
		rates[name] = types.Rate{TS: time.Now().Unix(), In: in, Out: out}
	}
	// Save to rates.json
	data, _ := json.Marshal(rates)
	os.WriteFile("rates.json", data, 0644)
	return rates, nil
}

// LoadRates loads rates from file if <7 days, else fetch
func LoadRates(ctx context.Context) map[string]types.Rate {
	file, err := os.Open("rates.json")
	if err != nil {
		rates, _ := FetchRates(ctx)
		return rates
	}
	defer file.Close()
	var rates map[string]types.Rate
	json.NewDecoder(file).Decode(&rates)
	now := time.Now().Unix()
	for _, rate := range rates {
		if now-rate.TS > constants.RateCacheDays {
			rates, _ := FetchRates(ctx)
			return rates
		}
	}
	return rates
}
