const questionInput = document.getElementById('questionInput');
const roundsSelect = document.getElementById('roundsSelect');
const submitBtn = document.getElementById('submitBtn');
const conversationBoard = document.getElementById('conversationBoard');
const finalResult = document.getElementById('finalResult');
const toggleConfigLink = document.getElementById('toggleConfig');
const modelConfig = document.getElementById('modelConfig');
const controlPanel = document.querySelector('.control-panel');
const hero = document.querySelector('.hero');

const cardElements = {
    grok: document.getElementById('grok'),
    gpt: document.getElementById('gpt'),
    claude: document.getElementById('claude'),
    gemini: document.getElementById('gemini')
};

const outputs = {
    grok: document.getElementById('grok-output'),
    gpt: document.getElementById('gpt-output'),
    claude: document.getElementById('claude-output'),
    gemini: document.getElementById('gemini-output')
};

const selectors = {
    grok: document.getElementById('grok-selector'),
    gpt: document.getElementById('gpt-selector'),
    claude: document.getElementById('claude-selector'),
    gemini: document.getElementById('gemini-selector')
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
    gemini: createEmptyModelState()
};

function createEmptyModelState() {
    return {
        totalRounds: lastTotalRounds,
        responses: [],
        dots: [],
        displayedRound: null
    };
}

function resetModelStates(totalRounds) {
    lastTotalRounds = totalRounds;
    Object.keys(modelState).forEach(model => {
        const state = modelState[model];
        state.totalRounds = totalRounds;
        state.responses = new Array(totalRounds).fill(null);
        state.displayedRound = null;
        renderRoundDots(model);
    });
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
        dot.addEventListener('click', () => {
            if (!dot.classList.contains('completed')) return;
            showRoundResponse(model, i + 1);
            setActiveDot(model, i + 1);
        });
        container.appendChild(dot);
        state.dots.push(dot);
    }
}

function markRoundCompleted(model, round, responseText) {
    const state = modelState[model];
    if (!state) return;
    state.responses[round - 1] = responseText;
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
    if (response !== null && response !== undefined) {
        outputs[model].className = 'model-output';
        outputs[model].textContent = response;
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

function initWebSocket() {
    ws = new WebSocket('ws://localhost:4444/ws');

    ws.onopen = function(event) {
        console.log('WebSocket connected');
    };

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        if (data.type === 'clear') {
            const total = parseInt(roundsSelect.value, 10) || 1;
            resetModelStates(total);
            prefillRandomQuestion();
            Object.entries(outputs).forEach(([model, output]) => {
                output.innerHTML = '<p class="placeholder">Responses will appear here once the collaboration begins.</p>';
                output.className = 'model-output';
                cardElements[model].classList.remove('winner', 'runner-up', 'loading', 'error');
            });
            conversationBoard.classList.remove('hidden');
            finalResult.classList.add('hidden');
            finalResult.textContent = '';
            submitBtn.textContent = 'Starting...';
        } else if (data.type === 'round_start') {
            submitBtn.textContent = `Round ${data.round}/${data.total}`;
            Object.values(cardElements).forEach(card => card.classList.add('loading'));
            ensureRounds(data.total);
            Object.keys(modelState).forEach(model => highlightCurrentRound(model, data.round));
        } else if (data.type === 'response') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'model-output';
                cardElements[data.model].classList.remove('loading', 'error', 'winner');
                output.textContent = data.response;
                markRoundCompleted(data.model, data.round, data.response);
                setActiveDot(data.model, data.round);
            }
        } else if (data.type === 'error') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'model-output error-text';
                cardElements[data.model].classList.remove('loading');
                cardElements[data.model].classList.add('error');
                output.textContent = `Error: ${data.error}`;
            }
        } else if (data.type === 'loading') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'model-output loading-text';
                cardElements[data.model].classList.add('loading');
                output.textContent = 'Processing...';
            }
        } else if (data.type === 'ranking_start') {
            submitBtn.textContent = 'Ranking...';
        } else if (data.type === 'winner') {
            Object.values(cardElements).forEach(card => card.classList.remove('loading'));
            
            // Handle winner
            const winnerElement = data.model ? cardElements[data.model] : null;
            if (winnerElement) {
                winnerElement.classList.add('winner');
            }
            
            // Handle runner-up
            const runnerUpElement = data.runner_up ? cardElements[data.runner_up] : null;
            if (runnerUpElement) {
                runnerUpElement.classList.add('runner-up');
            }
            
            submitBtn.textContent = '✓ Complete';
            submitBtn.disabled = false;
            setSelectorsEnabled(true);
            finalResult.classList.remove('hidden');
            
            let resultHTML = `<strong>🏆 Winner:</strong> ${winnerElement ? winnerElement.querySelector('.model-name').textContent : data.model}`;
            if (runnerUpElement) {
                resultHTML += ` &nbsp;|&nbsp; <strong>🥈 Runner-up:</strong> ${runnerUpElement.querySelector('.model-name').textContent}`;
            }
            finalResult.innerHTML = resultHTML;
        }
    };

    ws.onclose = function(event) {
        console.log('WebSocket closed, reconnecting...');
        setTimeout(initWebSocket, 1000);
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}

submitBtn.addEventListener('click', async function() {
    const question = questionInput.value.trim();
    if (!question) return;

    // Transition to compact mode
    controlPanel.classList.remove('initial');
    hero.classList.add('compact');
    modelConfig.classList.add('hidden');
    toggleConfigLink.textContent = '⚙️ Configure';

    conversationBoard.classList.remove('hidden');
    finalResult.classList.add('hidden');
    finalResult.textContent = '';
    Object.entries(outputs).forEach(([model, output]) => {
        output.innerHTML = '<p class="placeholder">Awaiting model response...</p>';
        output.className = 'model-output loading-text';
        cardElements[model].classList.remove('winner', 'runner-up', 'error');
        cardElements[model].classList.add('loading');
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
    // Cmd/Ctrl + Enter to submit
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
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
            
            // Add options
            sortedVariants.forEach(variant => {
                const option = document.createElement('option');
                option.value = variant.key;
                option.textContent = variant.name;
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
toggleConfigLink.addEventListener('click', function(e) {
    e.preventDefault();
    modelConfig.classList.toggle('hidden');
    toggleConfigLink.textContent = modelConfig.classList.contains('hidden') 
        ? '⚙️ Configure' 
        : '✕ Close';
});

// Set initial state
controlPanel.classList.add('initial');

// Rounds slider update
const roundsSlider = document.getElementById('roundsSelect');
const roundsValueDisplay = document.getElementById('roundsValue');

roundsSlider.addEventListener('input', function() {
    roundsValueDisplay.textContent = this.value;
});

// Initialize WebSocket connection
prefillRandomQuestion(true);
loadModels();
initWebSocket();
