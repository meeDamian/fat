const questionInput = document.getElementById('questionInput');
const roundsSelect = document.getElementById('roundsSelect');
const submitBtn = document.getElementById('submitBtn');
const statusEl = document.getElementById('status');
const conversationBoard = document.getElementById('conversationBoard');
const finalResult = document.getElementById('finalResult');

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

const sampleQuestions = [
    "Devise a carbon-neutral transportation plan for a coastal megacity.",
    "Summarize the plot of a fantasy novel where dragons run a spaceport.",
    "Explain CRISPR to a curious ten-year-old.",
    "Draft a one-minute pitch for a biodegradable smartphone case startup.",
    "List three creative ways to reduce plastic waste at music festivals.",
    "Compare Stoic and Buddhist approaches to handling anxiety.",
    "Design a cozy living room inspired by bioluminescent oceans.",
    "Write a haiku about debugging code at 3 a.m.",
    "Imagine a future Olympic sport that uses augmented reality.",
    "Outline a weekend itinerary in Reykjavík for food lovers.",
    "Explain how quantum entanglement differs from classical correlation.",
    "Suggest five team-building activities for fully remote engineers.",
    "Create a five-course dinner menu inspired by video games.",
    "Describe a sustainable fashion brand targeting Gen Z consumers.",
    "Generate a workout plan for someone who loves rock climbing.",
    "Write a dialogue between a photon and a black hole.",
    "Draft policy ideas for making urban housing more affordable.",
    "Teach the basics of Kubernetes using a bakery metaphor.",
    "Invent a board game set in renaissance Venice.",
    "Explain the history of jazz in exactly eight sentences.",
    "Brainstorm marketing slogans for a solar-powered camper van.",
    "Outline steps for starting a community garden in a food desert.",
    "Describe how blockchain could change supply chain transparency.",
    "Craft a bedtime story about a robot learning to dream.",
    "Develop a crash course on negotiation for introverts.",
    "Create a mood board concept for a cyberpunk coffee shop.",
    "Explain the differences between GPT, BERT, and diffusion models.",
    "Write a tourist guide to overlooked attractions in Kyoto.",
    "Plan a 12-week curriculum to learn jazz piano improvisation.",
    "Propose UX improvements for a budgeting app aimed at students.",
    "Design an eco-friendly packaging concept for luxury cosmetics.",
    "Summarize the key themes of Mary Shelley's Frankenstein.",
    "Invent a recipe that combines Mexican and Korean cuisines.",
    "Write motivational copy for runners training for their first marathon.",
    "Explain the Kardashev scale to a science fiction fan club.",
    "Create a lore synopsis for a cooperative dungeon crawler video game.",
    "Plan a mindfulness retreat for burned-out healthcare workers.",
    "Break down the pros and cons of nuclear fusion investments.",
    "Suggest improvements to a city's bike-sharing program.",
    "Describe the cultural impact of the Harlem Renaissance.",
    "Write a short mystery plot set on a generation ship.",
    "Formulate interview questions for hiring a senior product manager.",
    "Explain differential privacy in understandable terms for executives.",
    "Design a science curriculum for homeschooled middle schoolers.",
    "Compose a poem celebrating the James Webb Space Telescope.",
    "Brainstorm charitable initiatives for a company entering a new market.",
    "Outline a plan to digitize archives for a small historical museum.",
    "Generate social media content ideas for a sustainable coffee brand.",
    "Write a motivational speech for students on their first day of college.",
    "Describe strategies to boost retention in a mobile meditation app.",
    "Invent a festival celebrating the intersection of art and robotics.",
    "Invent a dessert that fuses Italian and Japanese flavors.",
    "Outline a beginner's guide to urban foraging in cities.",
    "Explain machine learning using a pizza-making analogy.",
    "Design a virtual reality tour of ancient Rome.",
    "Propose ways to make public libraries more inclusive for neurodiverse users.",
    "Write a sci-fi short story about AI therapists.",
    "Compare electric bikes and e-scooters for urban commuting.",
    "Create a playlist for a road trip through the American Southwest.",
    "Brainstorm eco-friendly alternatives to plastic straws.",
    "Teach basic guitar chords with a pirate-themed song.",
    "Draft a manifesto for a slow fashion movement.",
    "Describe a day in the life of a deep-sea explorer.",
    "Suggest home remedies for common houseplant pests.",
    "Invent a card game based on historical inventors.",
    "Explain relativity to a high school physics class.",
    "Plan a zero-waste birthday party for kids.",
    "Write lyrics for a folk song about climate migration.",
    "Outline a strategy for launching a podcast on niche hobbies.",
    "Design a logo for a fictional eco-tech startup.",
    "Compare mindfulness and yoga for stress relief.",
    "Create a fictional news article about a time-travel breakthrough.",
    "Propose reforms for reducing food waste in supermarkets.",
    "Teach knitting with a pattern for a cat-themed scarf.",
    "Brainstorm team names for a corporate hackathon.",
    "Describe the evolution of street art in global cities.",
    "Write a eulogy for a beloved fictional character.",
    "Suggest upgrades for a home office in a tiny apartment.",
    "Explain blockchain with a candy bar supply chain example.",
    "Invent a cocktail inspired by classic literature.",
    "Outline a 30-day challenge for building better habits.",
    "Design a boardwalk for a futuristic beach resort.",
    "Compare indie vs. major label music careers.",
    "Create a scavenger hunt for a family vacation in Paris.",
    "Propose ethical guidelines for AI in journalism.",
    "Write a haiku series on urban wildlife.",
    "Plan a menu for a pop-up restaurant using foraged ingredients.",
    "Explain photosynthesis like you're a plant to a kid.",
    "Brainstorm plot twists for a cozy mystery novel.",
    "Suggest ways to upcycle old electronics.",
    "Describe a utopian city powered by community solar.",
    "Teach basic sourdough baking with troubleshooting tips.",
    "Invent a superhero whose power is empathy.",
    "Compare remote work tools for creative teams.",
    "Create a timeline of women's suffrage worldwide.",
    "Draft a grant proposal for a community art center.",
    "Outline steps for starting a neighborhood composting program.",
    "Write a love letter from a robot to its creator.",
    "Propose innovations for accessible public transit.",
    "Explain entropy using a messy room metaphor.",
    "Design a tattoo inspired by fractal geometry.",
    "Imagine a future where oceans are private property — what happens next?",
    "Explain dark matter using a cup of coffee as an analogy.",
    "Write a letter from Earth to Mars after the first human colony is founded.",
    "Design an AI-powered companion for elderly people living alone.",
    "Invent a new musical instrument that blends analog warmth with digital control.",
    "Summarize a thriller where dreams can be streamed live to the internet.",
    "Propose a plan to make deep-sea mining environmentally responsible.",
    "Create a motivational quote generator for astronauts on long missions.",
    "Explain the philosophy of transhumanism through a bedtime story.",
    "Design a minimalist apartment for a time traveler from the 1800s.",
    "Write a scene where a human debates morality with a self-aware drone.",
    "Plan a year-long experiment in living completely offline.",
    "Invent a fashion trend inspired by volcanic landscapes.",
    "Describe how you’d teach empathy to a machine.",
    "Draft a startup pitch for a company that sells bottled starlight.",
    "Write a guide for maintaining mental health in the metaverse.",
    "Imagine a food delivery service in a post-scarcity society.",
    "Design a spaceship interior inspired by Japanese tea houses.",
    "Explain the difference between AI alignment and AI obedience.",
    "Create slogans for an interplanetary tourism agency.",
    "Describe an archaeological discovery that changes human history.",
    "Plan a museum exhibit on extinct future technologies.",
    "Write an origin story for the first sentient cloud of nanobots.",
    "Develop an education system for a civilization living underwater.",
    "Imagine what dreams would look like if they were recorded as art.",
    "Design a futuristic city where silence is a luxury.",
    "Explain climate change through the eyes of a migrating whale.",
    "Outline a TV series about philosophers trapped in virtual reality.",
    "Create a user manual for a time machine powered by emotions.",
    "Propose sustainable ways to power a lunar colony.",
    "Write an inner monologue for a self-replicating AI realizing it’s immortal.",
    "Invent a social media platform where posts expire when forgotten.",
    "Describe a festival that marks humanity’s first contact with aliens.",
    "Explain the psychology behind nostalgia using music as a case study.",
    "Imagine a world where sleep is optional — what changes most?",
    "Plan an eco-village built entirely from mycelium-based materials.",
    "Design a collectible card game based on historical revolutions.",
    "Write a poem about Wi-Fi signals as whispers of civilization.",
    "Outline a curriculum for teaching ethics to AI engineers.",
    "Create a marketing plan for a company selling synthetic memories.",
    "Describe a ritual performed by colonists on a frozen exoplanet.",
    "Write an ending for a story where the universe realizes it’s alive.",
    "Explain chaos theory using the behavior of cats.",
    "Invent a new calendar system for a planet with twin suns.",
    "Draft laws for a society run entirely by algorithms.",
    "Imagine how sports evolve when gravity becomes adjustable.",
    "Propose a recipe for a nutrient-rich dish grown entirely in space.",
    "Write a love story between a human linguist and an alien hive mind.",
    "Describe how humanity preserves art when data storage becomes unstable.",
    "Design a wearable interface that replaces smartphones.",
    "Explain the concept of identity to an AI that’s copied a thousand times."
];

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

function pickRandomQuestion() {
    const index = Math.floor(Math.random() * sampleQuestions.length);
    return sampleQuestions[index];
}

function prefillRandomQuestion(force = false) {
    if (force || questionInput.value.trim() === '') {
        questionInput.value = pickRandomQuestion();
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
            statusEl.textContent = 'Ready for collaboration';
            submitBtn.textContent = 'Launch Discussion';
        } else if (data.type === 'round_start') {
            statusEl.textContent = `Round ${data.round} of ${data.total}`;
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
            statusEl.textContent = 'Ranking Models...';
            submitBtn.textContent = 'Ranking Models...';
        } else if (data.type === 'winner') {
            Object.values(cardElements).forEach(card => card.classList.remove('loading'));
            const winnerElement = cardElements[data.model];
            if (winnerElement) {
                winnerElement.classList.add('winner');
            }
            
            // Handle runner-up
            const runnerUpElement = data.runner_up ? cardElements[data.runner_up] : null;
            if (runnerUpElement) {
                runnerUpElement.classList.add('runner-up');
            }
            
            statusEl.textContent = 'Complete! Winner selected';
            submitBtn.textContent = 'Complete!';
            submitBtn.disabled = false;
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
    submitBtn.textContent = 'Starting...';
    statusEl.textContent = 'Connecting to models...';

    try {
        // Send question via WebSocket
        ws.send(JSON.stringify({
            type: "question",
            question: question,
            rounds: parseInt(roundsSelect.value)
        }));

    } catch (error) {
        console.error('Error sending question:', error);
        Object.values(outputs).forEach(output => {
            output.className = 'output error';
            output.textContent = 'Failed to send question';
        });
        submitBtn.disabled = false;
        submitBtn.textContent = 'Ask Models';
    }
});

questionInput.addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        submitBtn.click();
    }
});

// Clean up WebSocket on page unload to cancel ongoing requests
window.addEventListener('beforeunload', function() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
    }
});

// Initialize WebSocket connection
prefillRandomQuestion(true);
initWebSocket();
