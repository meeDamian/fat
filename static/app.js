const questionInput = document.getElementById('questionInput');
const roundsSelect = document.getElementById('roundsSelect');
const submitBtn = document.getElementById('submitBtn');
const conversationBoard = document.getElementById('conversationBoard');
const toggleConfigLink = document.getElementById('toggleConfig');
const modelConfig = document.getElementById('modelConfig');
const controlPanel = document.querySelector('.control-panel');
const hero = document.querySelector('.hero');
const heroStage = document.getElementById('heroStage');
const galleryStage = document.getElementById('galleryStage');
const modelOrder = ['grok', 'gpt', 'claude', 'gemini', 'deepseek', 'mistral'];
let heroLayoutEnabled = false;
let currentHeroId = null;

const cardElements = {
    grok: document.getElementById('grok'),
    gpt: document.getElementById('gpt'),
    claude: document.getElementById('claude'),
    gemini: document.getElementById('gemini'),
    deepseek: document.getElementById('deepseek'),
    mistral: document.getElementById('mistral')
};

const statusIndicators = {
    grok: cardElements.grok?.querySelector('.model-status') || null,
    gpt: cardElements.gpt?.querySelector('.model-status') || null,
    claude: cardElements.claude?.querySelector('.model-status') || null,
    gemini: cardElements.gemini?.querySelector('.model-status') || null,
    deepseek: cardElements.deepseek?.querySelector('.model-status') || null,
    mistral: cardElements.mistral?.querySelector('.model-status') || null
};

const costIndicators = {
    grok: document.querySelector('.model-cost[data-model="grok"]'),
    gpt: document.querySelector('.model-cost[data-model="gpt"]'),
    claude: document.querySelector('.model-cost[data-model="claude"]'),
    gemini: document.querySelector('.model-cost[data-model="gemini"]'),
    deepseek: document.querySelector('.model-cost[data-model="deepseek"]'),
    mistral: document.querySelector('.model-cost[data-model="mistral"]')
};

// Track cumulative costs per model for current request
const modelCosts = {
    grok: 0,
    gpt: 0,
    claude: 0,
    gemini: 0,
    deepseek: 0,
    mistral: 0
};

function setCardStatus(model, icon = '') {
    const indicator = statusIndicators[model];
    if (!indicator) return;
    indicator.textContent = icon;
    indicator.classList.toggle('visible', Boolean(icon));
}

function formatCost(cost) {
    // Always show cost in cents with ¢ symbol
    const cents = cost * 100;
    // Remove trailing zeros after decimal point
    return `${cents.toFixed(4).replace(/\.?0+$/, '')}¢`;
}

function updateCostIndicator(model, additionalCost) {
    modelCosts[model] += additionalCost;
    const indicator = costIndicators[model];
    if (indicator) {
        indicator.textContent = formatCost(modelCosts[model]);
        indicator.classList.add('visible');
        updateCostColors();
    }
}

function updateCostColors() {
    // Get all non-zero costs
    const costs = Object.values(modelCosts).filter(c => c > 0);
    if (costs.length === 0) return;
    
    const minCost = Math.min(...costs);
    const maxCost = Math.max(...costs);
    const range = maxCost - minCost;
    
    // Apply gradient colors to each model
    for (const model in modelCosts) {
        const cost = modelCosts[model];
        if (cost === 0) continue;
        
        const indicator = costIndicators[model];
        if (!indicator) continue;
        
        // Calculate position in range (0 = cheapest, 1 = most expensive)
        const position = range === 0 ? 0 : (cost - minCost) / range;
        
        // Green -> Yellow -> Red gradient
        let r, g, b;
        if (position < 0.5) {
            // Green to Yellow (0 to 0.5)
            const t = position * 2; // 0 to 1
            r = Math.round(129 + (255 - 129) * t); // 129 to 255
            g = Math.round(199 + (235 - 199) * t); // 199 to 235
            b = Math.round(132 * (1 - t)); // 132 to 0
        } else {
            // Yellow to Red (0.5 to 1)
            const t = (position - 0.5) * 2; // 0 to 1
            r = 255;
            g = Math.round(235 * (1 - t)); // 235 to 0
            b = 0;
        }
        
        // Apply colors
        indicator.style.backgroundColor = `rgba(${r}, ${g}, ${b}, 0.2)`;
        indicator.style.color = `rgb(${r}, ${g}, ${b})`;
    }
}

function resetCosts() {
    for (const model in modelCosts) {
        modelCosts[model] = 0;
        const indicator = costIndicators[model];
        if (indicator) {
            indicator.textContent = '';
            indicator.classList.remove('visible');
            indicator.style.backgroundColor = '';
            indicator.style.color = '';
        }
    }
}

const outputs = {
    grok: document.getElementById('grok-output'),
    gpt: document.getElementById('gpt-output'),
    claude: document.getElementById('claude-output'),
    gemini: document.getElementById('gemini-output'),
    deepseek: document.getElementById('deepseek-output'),
    mistral: document.getElementById('mistral-output')
};

const selectors = {
    grok: document.getElementById('grok-selector'),
    gpt: document.getElementById('gpt-selector'),
    claude: document.getElementById('claude-selector'),
    gemini: document.getElementById('gemini-selector'),
    deepseek: document.getElementById('deepseek-selector'),
    mistral: document.getElementById('mistral-selector')
};

// Fetch random question from backend
async function fetchRandomQuestion() {
    try {
        const response = await fetch('/question/random');
        const data = await response.json();
        return data.question || "Explain the concept of emergence in complex systems.";
    } catch (error) {
        console.error('Failed to fetch random question:', error);
        return "Explain the concept of emergence in complex systems.";
    }
}

let ws;
let lastTotalRounds = parseInt(roundsSelect.value, 10) || 3;
const modelState = {
    grok: createEmptyModelState(),
    gpt: createEmptyModelState(),
    claude: createEmptyModelState(),
    gemini: createEmptyModelState(),
    deepseek: createEmptyModelState(),
    mistral: createEmptyModelState()
};

function createEmptyModelState() {
    return {
        totalRounds: lastTotalRounds,
        responses: [],
        rationales: [],
        discussions: [],
        dots: [],
        displayedRound: null
    };
}

function resetModelState(model, totalRounds) {
    const state = modelState[model];
    if (state) {
        state.totalRounds = totalRounds;
        state.responses = new Array(totalRounds).fill(null);
        state.rationales = new Array(totalRounds).fill(null);
        state.discussions = new Array(totalRounds).fill(null);
        state.displayedRound = null;
        renderRoundDots(model);
        setCardStatus(model, '');
    }
}

function resetModelStates(totalRounds) {
    lastTotalRounds = totalRounds;
    Object.keys(modelState).forEach(model => resetModelState(model, totalRounds));
}

function ensureRounds(totalRounds) {
    if (totalRounds !== lastTotalRounds) {
        resetModelStates(totalRounds);
    }
}

function renderRoundDots(model) {
    const container = document.querySelector(`.round-progress[data-model="${model}"]`);
    if (!container) return;

    container.innerHTML = '';
    const state = modelState[model];
    state.dots = [];

    for (let i = 0; i < state.totalRounds; i++) {
        const dot = document.createElement('span');
        dot.classList.add('round-dot');
        dot.dataset.round = i + 1;
        dot.addEventListener('click', (e) => {
            e.stopPropagation();
            if (!dot.classList.contains('completed')) return;
            showRoundResponse(model, i + 1);
            setActiveDot(model, i + 1);
        });
        container.appendChild(dot);
        state.dots.push(dot);
    }
}

function markRoundCompleted(model, round, responseText, rationaleText, discussionData) {
    const state = modelState[model];
    if (!state) return;
    state.responses[round - 1] = responseText;
    state.rationales[round - 1] = rationaleText || '';
    state.discussions[round - 1] = discussionData || {};
    const dot = state.dots[round - 1];
    if (dot) {
        dot.classList.add('completed');
    }
    state.displayedRound = round;
}

function setActiveDot(model, round) {
    const state = modelState[model];
    if (!state) return;
    state.dots.forEach(dot => dot.classList.remove('active'));
    const targetDot = state.dots[round - 1];
    if (targetDot) {
        targetDot.classList.add('active');
    }
    state.displayedRound = round;
}

function highlightCurrentRound(model, round) {
    const state = modelState[model];
    if (!state) return;
    setActiveDot(model, round);
}

function showRoundResponse(model, round) {
    const state = modelState[model];
    if (!state) return;
    const response = state.responses[round - 1];
    const rationale = state.rationales[round - 1];
    
    const output = outputs[model];
    output.className = 'model-output';
    output.innerHTML = '';
    
    // If there's a response, show it
    if (response) {
        const answerDiv = document.createElement('div');
        answerDiv.className = 'answer-text';
        answerDiv.textContent = response;
        output.appendChild(answerDiv);
    }
    
    // Show rationale if present
    if (rationale) {
        const rationaleDiv = document.createElement('div');
        rationaleDiv.className = 'rationale-text';
        rationaleDiv.textContent = rationale;
        output.appendChild(rationaleDiv);
    }
}

function showLatestResponse(model) {
    const state = modelState[model];
    if (!state) return;
    for (let i = state.totalRounds - 1; i >= 0; i--) {
        if (state.responses[i]) {
            showRoundResponse(model, i + 1);
            setActiveDot(model, i + 1);
            return;
        }
    }
}

async function prefillRandomQuestion(force = false) {
    if (force || questionInput.value.trim() === '') {
        const question = await fetchRandomQuestion();
        questionInput.value = question;
    }
}

const connectionStatus = document.getElementById('connectionStatus');

function updateConnectionStatus(status) {
    connectionStatus.className = 'connection-status ' + status;
}

function initWebSocket() {
    updateConnectionStatus('connecting');
    ws = new WebSocket('ws://localhost:4444/ws');

    ws.onopen = function(event) {
        console.log('WebSocket connected');
        updateConnectionStatus('connected');
    };

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        if (data.type === 'clear') {
            const total = parseInt(roundsSelect.value, 10) || 1;
            resetModelStates(total);
            resetCosts();
            prefillRandomQuestion();
            Object.entries(outputs).forEach(([model, output]) => {
                output.innerHTML = '<p class="placeholder">Responses will appear here once the collaboration begins.</p>';
                output.className = 'model-output';
                cardElements[model].className = 'model-card';
                setCardStatus(model, '');
            });
            conversationBoard.classList.remove('hidden');
            document.getElementById('discussionsSection')?.classList.add('hidden');
            activeDiscussionFilter = null;
            submitBtn.textContent = 'Starting...';
            resetHeroLayout();
        } else if (data.type === 'round_start') {
            submitBtn.textContent = `Round ${data.round}/${data.total}`;
            Object.values(cardElements).forEach(card => card.classList.add('loading'));
            ensureRounds(data.total);
            Object.keys(modelState).forEach(model => highlightCurrentRound(model, data.round));
        } else if (data.type === 'response') {
            const output = outputs[data.model];
            if (output) {
                cardElements[data.model].classList.remove('loading', 'error', 'winner');
                setCardStatus(data.model, '');
                markRoundCompleted(data.model, data.round, data.response, data.rationale, data.discussion);
                showRoundResponse(data.model, data.round);
                setActiveDot(data.model, data.round);
                
                // Update cost if provided
                if (data.cost !== undefined) {
                    updateCostIndicator(data.model, data.cost);
                }
                
                // Update discussions section if there are any discussions
                if (data.discussion && Object.keys(data.discussion).length > 0) {
                    buildDiscussionsSection();
                }
            }
        } else if (data.type === 'error') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'model-output error-text';
                cardElements[data.model].classList.remove('loading');
                cardElements[data.model].classList.add('error');
                setCardStatus(data.model, '');
                output.textContent = `Error: ${data.error}`;
            }
        } else if (data.type === 'loading') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'model-output loading-text';
                cardElements[data.model].classList.add('loading');
                setCardStatus(data.model, '');
                output.textContent = 'Processing...';
            }
        } else if (data.type === 'ranking_start') {
            submitBtn.textContent = 'Ranking...';
        } else if (data.type === 'winner') {
            Object.values(cardElements).forEach(card => card.classList.remove('loading'));

            const winnerId = data.model;
            const runnerUpId = data.runner_up;

            Object.keys(statusIndicators).forEach(model => setCardStatus(model, ''));

            if (winnerId && cardElements[winnerId]) {
                cardElements[winnerId].classList.add('winner');
                currentHeroId = winnerId;
                setCardStatus(winnerId, '🏆');
            }

            if (runnerUpId && cardElements[runnerUpId]) {
                cardElements[runnerUpId].classList.add('runner-up');
                setCardStatus(runnerUpId, '🥈');
            }

            buildHeroLayout(winnerId, runnerUpId);
            
            // Build and show discussions
            buildDiscussionsSection();

            submitBtn.textContent = '✓ Complete';
            submitBtn.disabled = false;
            setSelectorsEnabled(true);
        }
    };

    ws.onclose = function(event) {
        console.log('WebSocket closed, reconnecting...');
        updateConnectionStatus('disconnected');
        setTimeout(initWebSocket, 1000);
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        updateConnectionStatus('disconnected');
    };
}

submitBtn.addEventListener('click', async function() {
    const question = questionInput.value.trim();
    if (!question) return;

    // Transition to compact mode
    controlPanel.classList.remove('initial');
    hero.classList.add('compact');
    if (modelConfig) {
        modelConfig.classList.add('hidden');
    }
    if (toggleConfigLink) {
        toggleConfigLink.textContent = '⚙️ Configure';
    }

    conversationBoard.classList.remove('hidden');
    Object.entries(outputs).forEach(([model, output]) => {
        output.innerHTML = '<p class="placeholder">Awaiting model response...</p>';
        output.className = 'model-output loading-text';
        cardElements[model].classList.remove('winner', 'runner-up', 'error');
        cardElements[model].classList.add('loading');
        setCardStatus(model, '');
        renderRoundDots(model);
    });

    submitBtn.disabled = true;
    submitBtn.textContent = 'Processing...';
    
    // Lock model selectors
    setSelectorsEnabled(false);

    try {
        // Get selected models
        const selectedModels = getSelectedModels();
        
        // Send question via WebSocket with selected models
        ws.send(JSON.stringify({
            type: "question",
            question: question,
            rounds: parseInt(roundsSelect.value),
            models: selectedModels
        }));

    } catch (error) {
        console.error('Error sending question:', error);
        Object.values(outputs).forEach(output => {
            output.className = 'output error';
            output.textContent = 'Failed to send question';
        });
        submitBtn.disabled = false;
        submitBtn.textContent = 'Launch Discussion';
        setSelectorsEnabled(true);
    }
});

questionInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter') {
        if (e.shiftKey) {
            // allow newline
            return;
        }
        e.preventDefault();
        submitBtn.click();
    }
});

// Clean up WebSocket on page unload to cancel ongoing requests
window.addEventListener('beforeunload', function() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
    }
});

// Load available models and populate dropdowns
async function loadModels() {
    try {
        const response = await fetch('/models');
        const families = await response.json();
        
        Object.entries(families).forEach(([familyID, familyData]) => {
            const selector = selectors[familyID];
            if (!selector) return;
            
            // Clear loading option
            selector.innerHTML = '';
            
            // Sort variants by name for consistent ordering
            const sortedVariants = familyData.variants.sort((a, b) => a.name.localeCompare(b.name));
            
            // Add options with pricing
            sortedVariants.forEach(variant => {
                const option = document.createElement('option');
                option.value = variant.key;
                // Format: model-name ($X/$Y)
                const priceIn = variant.rate_in ? `$${variant.rate_in.toFixed(2)}` : '$0.00';
                const priceOut = variant.rate_out ? `$${variant.rate_out.toFixed(2)}` : '$0.00';
                option.textContent = `${variant.name} (${priceIn}/${priceOut})`;
                selector.appendChild(option);
            });
            
            // Set default to active model
            if (familyData.active) {
                selector.value = familyData.active;
            }
        });
    } catch (error) {
        console.error('Failed to load models:', error);
        Object.values(selectors).forEach(selector => {
            selector.innerHTML = '<option value="">Error loading models</option>';
        });
    }
}

// Lock/unlock model selectors
function setSelectorsEnabled(enabled) {
    Object.values(selectors).forEach(selector => {
        selector.disabled = !enabled;
    });
}

function resetHeroLayout() {
    heroLayoutEnabled = false;
    currentHeroId = null;
    heroStage.classList.remove('active');
    heroStage.innerHTML = '';
    galleryStage.classList.remove('interactive', 'compact');
    galleryStage.innerHTML = '';
    // Re-append cards in original order
    modelOrder.forEach(id => {
        const card = cardElements[id];
        if (card) {
            card.className = 'model-card';
            galleryStage.appendChild(card);
        }
    });
}

function buildHeroLayout(winnerId, runnerUpId) {
    heroLayoutEnabled = true;
    heroStage.innerHTML = '';
    galleryStage.innerHTML = '';

    heroStage.classList.add('active');
    galleryStage.classList.add('interactive');

    const orderedIds = [...modelOrder];
    // Winner first
    if (winnerId) {
        moveCardToHero(winnerId, true);
    }

    // Populate gallery with remaining cards
    orderedIds.filter(id => id !== winnerId).forEach(id => {
        const card = cardElements[id];
        if (!card) return;
        card.classList.remove('active-card');
        card.classList.add('compact');
        card.removeEventListener('click', handleGalleryCardClick);
        card.addEventListener('click', handleGalleryCardClick);
        galleryStage.appendChild(card);
    });

    if (runnerUpId && cardElements[runnerUpId]) {
        galleryStage.classList.add('compact');
    }
}

function moveCardToHero(cardId, isInitial = false) {
    const card = cardElements[cardId];
    if (!card) return;

    currentHeroId = cardId;
    heroStage.innerHTML = '';
    card.classList.remove('compact');
    card.classList.add('hero-card', 'active-card');
    heroStage.appendChild(card);

    if (!isInitial) {
        // Rebuild gallery with remaining cards
        const remaining = modelOrder.filter(id => id !== cardId);
        galleryStage.innerHTML = '';
        remaining.forEach(id => {
            const galleryCard = cardElements[id];
            if (!galleryCard) return;
            galleryCard.classList.remove('hero-card', 'active-card');
            galleryCard.classList.add('compact');
            galleryCard.removeEventListener('click', handleGalleryCardClick);
            galleryCard.addEventListener('click', handleGalleryCardClick);
            galleryStage.appendChild(galleryCard);
        });
    }
}

function handleGalleryCardClick(event) {
    const card = event.currentTarget;
    const id = card.dataset.model;
    if (!id || id === currentHeroId) return;
    moveCardToHero(id);
}

// Get selected models
function getSelectedModels() {
    const selected = {};
    Object.entries(selectors).forEach(([family, selector]) => {
        if (selector.value) {
            selected[family] = selector.value;
        }
    });
    return selected;
}

// Toggle configuration panel
if (toggleConfigLink && modelConfig) {
    toggleConfigLink.addEventListener('click', function(e) {
        e.preventDefault();
        modelConfig.classList.toggle('hidden');
        toggleConfigLink.textContent = modelConfig.classList.contains('hidden') 
            ? '⚙️ Configure' 
            : '✕ Close';
    });
}

// Set initial state
controlPanel.classList.add('initial');

// Rounds slider update
const roundsSlider = document.getElementById('roundsSelect');
const roundsValueDisplay = document.getElementById('roundsValue');

function updateRoundsSliderUI(value) {
    if (!roundsSlider) return;
    const displayValue = Number.isFinite(value) ? value : parseInt(roundsSlider.value, 10) || 3;
    if (roundsValueDisplay) {
        roundsValueDisplay.textContent = displayValue;
    }
    const size = Math.min(18 + (displayValue - 3) * 2, 30);
    roundsSlider.style.setProperty('--thumb-size', `${size}px`);
}

if (roundsSlider) {
    roundsSlider.addEventListener('input', function() {
        const value = parseInt(this.value, 10);
        updateRoundsSliderUI(value);
    });
    updateRoundsSliderUI(parseInt(roundsSlider.value, 10));
}

// Track active discussion filter
let activeDiscussionFilter = null;

// Build discussions section from collected discussion data
function buildDiscussionsSection() {
    const discussionsSection = document.getElementById('discussionsSection');
    const discussionsContainer = document.getElementById('discussionsContainer');
    const filtersContainer = document.getElementById('discussionFilters');
    
    if (!discussionsSection || !discussionsContainer || !filtersContainer) return;
    
    // Helper to normalize agent name to model ID
    const normalizeToModelId = (agentName) => {
        // If it's already a model ID, return it
        if (modelState[agentName]) return agentName;
        
        // Try to extract model ID from full name or partial match
        const lowerName = agentName.toLowerCase();
        for (const modelId of Object.keys(modelState)) {
            if (lowerName.includes(modelId)) return modelId;
        }
        
        // Fallback: return as-is
        return agentName;
    };
    
    // Collect all discussions grouped by agent pairs
    const pairConversations = {};
    
    Object.keys(modelState).forEach(fromModel => {
        const state = modelState[fromModel];
        state.discussions.forEach((discussionData, roundIndex) => {
            if (!discussionData || Object.keys(discussionData).length === 0) return;
            
            Object.entries(discussionData).forEach(([toAgent, message]) => {
                // Normalize both agents to model IDs
                const fromId = normalizeToModelId(fromModel);
                const toId = normalizeToModelId(toAgent);
                
                // Skip if we couldn't normalize
                if (!modelState[fromId] || !modelState[toId]) return;
                
                // Create a normalized pair key (alphabetically sorted)
                const pair = [fromId, toId].sort().join('-');
                
                if (!pairConversations[pair]) {
                    pairConversations[pair] = [];
                }
                
                // Add message to the conversation
                pairConversations[pair].push({
                    round: roundIndex + 1,
                    from: fromId,
                    to: toId,
                    message: message
                });
            });
        });
    });
    
    // Clear container
    discussionsContainer.innerHTML = '';
    
    // If no discussions, hide section
    if (Object.keys(pairConversations).length === 0) {
        discussionsSection.classList.add('hidden');
        return;
    }
    
    // Show section and build UI
    discussionsSection.classList.remove('hidden');
    
    // Build filter chips
    filtersContainer.innerHTML = '';
    const allModels = Object.keys(modelState);
    
    // Add "All" chip
    const allChip = document.createElement('button');
    allChip.className = 'discussion-filter-chip' + (activeDiscussionFilter === null ? ' active' : '');
    allChip.textContent = 'All';
    allChip.addEventListener('click', () => {
        activeDiscussionFilter = null;
        buildDiscussionsSection();
    });
    filtersContainer.appendChild(allChip);
    
    // Add chip for each model
    allModels.forEach(modelId => {
        const chip = document.createElement('button');
        chip.className = 'discussion-filter-chip' + (activeDiscussionFilter === modelId ? ' active' : '');
        const modelName = cardElements[modelId]?.querySelector('.model-name')?.textContent || modelId;
        chip.textContent = modelName;
        chip.addEventListener('click', () => {
            activeDiscussionFilter = modelId;
            buildDiscussionsSection();
        });
        filtersContainer.appendChild(chip);
    });
    
    // Sort pairs alphabetically
    const sortedPairs = Object.keys(pairConversations).sort();
    
    // Filter pairs based on active filter
    const filteredPairs = activeDiscussionFilter 
        ? sortedPairs.filter(pair => pair.includes(activeDiscussionFilter))
        : sortedPairs;
    
    filteredPairs.forEach(pair => {
        const messages = pairConversations[pair];
        const [model1, model2] = pair.split('-');
        
        // Sort messages chronologically (by round, then maintain order)
        messages.sort((a, b) => a.round - b.round);
        
        const pairDiv = document.createElement('div');
        pairDiv.className = 'discussion-pair';
        
        const headerDiv = document.createElement('div');
        headerDiv.className = 'discussion-pair-header';
        
        const model1Name = cardElements[model1]?.querySelector('.model-name')?.textContent || model1;
        const model2Name = cardElements[model2]?.querySelector('.model-name')?.textContent || model2;
        
        headerDiv.textContent = `${model1Name} ↔ ${model2Name}`;
        pairDiv.appendChild(headerDiv);
        
        const conversationDiv = document.createElement('div');
        conversationDiv.className = 'discussion-conversation';
        
        // Display all messages in chronological order
        messages.forEach(msg => {
            const msgDiv = document.createElement('div');
            msgDiv.className = 'discussion-message';
            
            const fromName = cardElements[msg.from]?.querySelector('.model-name')?.textContent || msg.from;
            const toName = cardElements[msg.to]?.querySelector('.model-name')?.textContent || msg.to;
            
            const metaSpan = document.createElement('span');
            metaSpan.className = 'discussion-meta';
            metaSpan.textContent = `Round ${msg.round} • ${fromName} to ${toName}`;
            
            const textDiv = document.createElement('div');
            textDiv.className = 'discussion-text';
            textDiv.textContent = msg.message;
            
            msgDiv.appendChild(metaSpan);
            msgDiv.appendChild(textDiv);
            conversationDiv.appendChild(msgDiv);
        });
        
        pairDiv.appendChild(conversationDiv);
        discussionsContainer.appendChild(pairDiv);
    });
}

// Initialize WebSocket connection
prefillRandomQuestion(true);
loadModels();
initWebSocket();
