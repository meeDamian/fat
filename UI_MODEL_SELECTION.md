# UI Model Selection Feature

## Overview
Users can now select specific model variants for each AI family (Grok, GPT, Claude, Gemini) directly from the UI before starting a discussion.

## How It Works

### Frontend
1. **Model Dropdowns**: Each model card has a dropdown showing all available variants for that family
2. **Auto-populated**: Dropdowns are populated from `/models` API endpoint on page load
3. **Default Selection**: Each dropdown defaults to the model from `DefaultModels` in `models.go`
4. **Locked During Processing**: Dropdowns are disabled once "Launch Discussion" is clicked and re-enabled when complete

### Backend
1. **`/models` Endpoint**: Returns all available model families and their variants
   ```json
   {
     "grok": {
       "id": "grok",
       "provider": "xAI",
       "variants": [
         {"key": "Grok4Fast", "name": "grok-4-fast", "maxTok": 2000000},
         ...
       ],
       "active": "Grok4Fast"
     },
     ...
   }
   ```

2. **WebSocket Message**: Client sends selected models with question
   ```json
   {
     "type": "question",
     "question": "...",
     "rounds": 3,
     "models": {
       "grok": "Grok4Fast",
       "gpt": "GPT5Mini",
       "claude": "Claude35Haiku",
       "gemini": "Gemini25Flash"
     }
   }
   ```

3. **Dynamic Model Building**: `handleQuestionWS` builds `activeModels` array based on selected variants
   - Falls back to `DefaultModels` if no selection provided
   - Loads appropriate API keys per family
   - Validates variant exists in `ModelFamilies`

## User Experience

1. User opens the app
2. Dropdowns show all available models for each family (e.g., "grok-4-fast (2000K)")
3. User can change any dropdown to select different model variants
4. User enters question and clicks "Launch Discussion"
5. Dropdowns lock (disabled) during processing
6. Models process the question using selected variants
7. Dropdowns unlock when complete, ready for next question

## Benefits

✅ **Flexibility**: Choose different models per session without code changes  
✅ **Experimentation**: Easy A/B testing of model variants  
✅ **Cost Control**: Select cheaper/faster models when needed  
✅ **Capability**: Use most powerful models for complex questions  
✅ **No Restart**: Change models without restarting the server  

## Code Changes

### Files Modified
- `static/index.html` - Added model selector dropdowns to each card
- `static/style.css` - Styled dropdowns to match UI theme
- `static/app.js` - Load models, populate dropdowns, send selections, lock/unlock
- `cmd/fat/main.go` - Added `/models` endpoint, dynamic model building in WebSocket handler

### Key Functions
- `loadModels()` - Fetches and populates dropdowns
- `getSelectedModels()` - Extracts current dropdown selections
- `setSelectorsEnabled(bool)` - Locks/unlocks dropdowns
- `getAPIKeyForFamily(familyID)` - Retrieves API key for a family
- `/models` endpoint - Returns available model families and variants
- `handleQuestionWS` - Builds activeModels from selections

## Configuration

Models are still configured in `internal/models/models.go`:

```go
// Add new variants to ModelFamilies
ModelFamilies = map[string]types.ModelFamily{
    Grok: {
        Variants: map[string]types.ModelVariant{
            GrokNewModel: {Name: "grok-new-model", MaxTok: 200_000},
        },
    },
}

// Set defaults in DefaultModels (used as fallback)
var DefaultModels = map[string]string{
    Grok: Grok4Fast,
}
```

New variants automatically appear in UI dropdowns!
