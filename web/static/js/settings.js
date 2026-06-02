// settings.js - Gestion des paramètres TTS

// Save TTS settings to localStorage
async function saveTTSSettings() {
    const voice = document.getElementById('ttsVoice').value;
    const voiceFR = document.getElementById('ttsVoiceFR').value;
    const voiceEN = document.getElementById('ttsVoiceEN').value;
    const model = document.getElementById('ttsModel').value;
    const autoPlay = document.getElementById('ttsAutoPlay').checked;

    const settings = {
        voice: voice,
        voiceFR: voiceFR,
        voiceEN: voiceEN,
        model: model,
        autoPlay: autoPlay
    };

    localStorage.setItem('ttsSettings', JSON.stringify(settings));
    alert('Paramètres TTS enregistrés !');
}

// Load TTS settings from localStorage and populate form
async function loadTTSSettings() {
    // Load voices and models from API
    await loadTTSVoices();
    await loadTTSModels();

    // Try to load saved settings from localStorage
    const savedSettings = localStorage.getItem('ttsSettings');
    if (savedSettings) {
        try {
            const settings = JSON.parse(savedSettings);
            const selects = {
                voice: 'ttsVoice',
                voiceFR: 'ttsVoiceFR',
                voiceEN: 'ttsVoiceEN',
                model: 'ttsModel'
            };
            
            for (const [key, selectId] of Object.entries(selects)) {
                if (settings[key]) {
                    const select = document.getElementById(selectId);
                    if (select) {
                        const option = select.querySelector(`option[value="${settings[key]}"]`);
                        if (option) select.value = settings[key];
                    }
                }
            }
            const autoPlayCheckbox = document.getElementById('ttsAutoPlay');
            if (autoPlayCheckbox) {
                autoPlayCheckbox.checked = settings.autoPlay || false;
            }
        } catch (e) {
            console.error('Failed to load TTS settings:', e);
        }
    }
}

// Load TTS voices from API
async function loadTTSVoices() {
    try {
        const response = await fetch('/api/tts/voices');
        if (response.ok) {
            const voices = await response.json();
            const selects = ['ttsVoice', 'ttsVoiceFR', 'ttsVoiceEN'];
            
            selects.forEach(selectId => {
                const select = document.getElementById(selectId);
                if (!select) return;
                
                select.innerHTML = '';
                voices.forEach(voice => {
                    const option = document.createElement('option');
                    option.value = voice.id;
                    option.textContent = `${voice.name} (${voice.languages?.join(', ') || 'unknown'})`;
                    select.appendChild(option);
                });
            });
        }
    } catch (e) {
        console.error('Failed to load TTS voices:', e);
        // Fallback to default voices
        const fallbackVoices = `
            <option value="5a271406-039d-46fe-835b-fbbb00eaf08d">Marie - Neutral (fr_fr)</option>
            <option value="c69964a6-ab8b-4f8a-9465-ec0925096ec8">Paul - Neutral (en_us)</option>
            <option value="e3596645-b1af-469e-b857-f18ddedc7652">Oliver - Neutral (en_gb)</option>
        `;
        ['ttsVoice', 'ttsVoiceFR', 'ttsVoiceEN'].forEach(id => {
            const select = document.getElementById(id);
            if (select && select.options.length === 0) {
                select.innerHTML = fallbackVoices;
            }
        });
    }
}

// Load TTS models from API
async function loadTTSModels() {
    try {
        const response = await fetch('/api/tts/models');
        if (response.ok) {
            const models = await response.json();
            const select = document.getElementById('ttsModel');
            if (!select) return;
            
            select.innerHTML = '';
            models.forEach(model => {
                const option = document.createElement('option');
                option.value = model;
                option.textContent = model;
                select.appendChild(option);
            });
        }
    } catch (e) {
        console.error('Failed to load TTS models:', e);
    }
}

// Test TTS with selected voice and model
async function testTTS() {
    const voiceSelect = document.getElementById('ttsVoice');
    const modelSelect = document.getElementById('ttsModel');
    
    if (!voiceSelect || voiceSelect.options.length === 0 || !modelSelect || modelSelect.options.length === 0) {
        alert('Veuillez attendre, les voix et modèles TTS sont en cours de chargement...');
        return;
    }
    
    const text = document.getElementById('ttsTestText').value || 'Test de synthèse vocale';
    const voice = voiceSelect.value;
    const model = modelSelect.value;

    await playTTSFromSettings(text, voice, model);
}

// Initialize settings page
function initSettingsPage() {
    checkTTSSatus();
    loadTTSSettings();
}
