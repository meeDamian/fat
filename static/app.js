const questionInput = document.getElementById('questionInput');
const roundsSelect = document.getElementById('roundsSelect');
const submitBtn = document.getElementById('submitBtn');
const statusEl = document.getElementById('status');
const outputs = {
    grok: document.getElementById('grok-output'),
    gpt: document.getElementById('gpt-output'),
    claude: document.getElementById('claude-output'),
    gemini: document.getElementById('gemini-output')
};

let ws;

function initWebSocket() {
    ws = new WebSocket('ws://localhost:4444/ws');

    ws.onopen = function(event) {
        console.log('WebSocket connected');
    };

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        if (data.type === 'clear') {
            // Clear all outputs and remove winner styling
            Object.values(outputs).forEach(output => {
                output.textContent = 'Waiting for response...';
                output.className = 'output';
            });
            document.querySelectorAll('.grid-item').forEach(item => {
                item.classList.remove('winner');
            });
            statusEl.textContent = 'Ready';
            submitBtn.textContent = 'Ask Models';
        } else if (data.type === 'round_start') {
            statusEl.textContent = `Round ${data.round} of ${data.total}`;
            submitBtn.textContent = `Round ${data.round}/${data.total}`;
        } else if (data.type === 'response') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'output';
                const roundIndicator = data.round ? ` [Round ${data.round}]` : '';
                output.textContent = `${data.response}${roundIndicator}`;
            }
        } else if (data.type === 'error') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'output error';
                const roundIndicator = data.round ? ` [Round ${data.round}]` : '';
                output.textContent = `Error${roundIndicator}: ${data.error}`;
            }
        } else if (data.type === 'loading') {
            const output = outputs[data.model];
            if (output) {
                output.className = 'output loading';
                output.textContent = 'Processing...';
            }
        } else if (data.type === 'ranking_start') {
            statusEl.textContent = 'Ranking Models...';
            submitBtn.textContent = 'Ranking Models...';
        } else if (data.type === 'winner') {
            // Highlight the winning model
            const winnerElement = document.getElementById(data.model);
            if (winnerElement) {
                winnerElement.classList.add('winner');
            }
            statusEl.textContent = 'Complete! Winner selected.';
            submitBtn.textContent = 'Complete!';
            submitBtn.disabled = false;
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

    // Clear previous outputs and styling
    Object.values(outputs).forEach(output => {
        output.textContent = 'Waiting for response...';
        output.className = 'output';
    });
    document.querySelectorAll('.grid-item').forEach(item => {
        item.classList.remove('winner');
    });

    submitBtn.disabled = true;
    submitBtn.textContent = 'Starting...';

    try {
        const response = await fetch('/question', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ 
                question: question,
                rounds: parseInt(roundsSelect.value)
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        console.log('Question submitted:', result);

    } catch (error) {
        console.error('Error submitting question:', error);
        Object.values(outputs).forEach(output => {
            output.className = 'output error';
            output.textContent = 'Failed to submit question';
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

// Initialize WebSocket connection
initWebSocket();
