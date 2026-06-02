// tts.js - Synthèse vocale (Text-to-Speech)

let ttsEnabled = false;
let audioContext = null;
let audioElement = null;
let currentAudio = null;
let isPlaying = false;

// Saved TTS settings
let savedVoice = 'c69964a6-ab8b-4f8a-9465-ec0925096ec8';
let savedVoiceFR = '5a271406-039d-46fe-835b-fbbb00eaf08d';
let savedVoiceEN = 'c69964a6-ab8b-4f8a-9465-ec0925096ec8';
let savedModel = 'voxtral-mini-tts-2603';

// Variables for playing all articles
let isPlayingAll = false;
let playAllResolve = null;

// Check TTS status
async function checkTTSSatus() {
    try {
        const response = await fetch('/api/tts/status');
        if (response.ok) {
            const data = await response.json();
            ttsEnabled = data.enabled;
        }
    } catch (e) {
        ttsEnabled = false;
    }
}

// Load saved TTS settings from localStorage
function loadTTSSettings() {
    const savedSettings = localStorage.getItem('ttsSettings');
    if (savedSettings) {
        try {
            const settings = JSON.parse(savedSettings);
            if (settings.voice) savedVoice = settings.voice;
            if (settings.voiceFR) savedVoiceFR = settings.voiceFR;
            if (settings.voiceEN) savedVoiceEN = settings.voiceEN;
            if (settings.model) savedModel = settings.model;
        } catch (e) {
            console.error('Failed to load TTS settings:', e);
        }
    }
}

// Language detection for TTS voice selection
function detectLanguage(text) {
    const frWords = ['le ', 'la ', 'les ', 'de ', 'des ', 'un ', 'une ', 'et ', 'en ', 'à ', 'est ', 'pas ', 'je ', 'tu ', 'il ', 'elle '];
    const enWords = ['the ', 'a ', 'an ', 'and ', 'in ', 'to ', 'of ', 'is ', 'it ', 'that ', 'this ', 'with ', 'for '];

    const textLower = text.toLowerCase();
    const frCount = frWords.reduce((c, w) => c + (textLower.split(w).length - 1), 0);
    const enCount = enWords.reduce((c, w) => c + (textLower.split(w).length - 1), 0);

    return frCount > enCount ? 'fr' : 'en';
}

// Get voice for language
function getVoiceForLang(lang) {
    const voiceMap = {
        fr: savedVoiceFR,
        en: savedVoiceEN
    };
    return voiceMap[lang] || savedVoice;
}

// Initialize TTS player
function initTTSPlayer() {
    audioElement = document.getElementById('ttsAudio');
    if (!audioElement) return;

    audioElement.addEventListener('play', function() {
        isPlaying = true;
        updatePlayIcon();
    });

    audioElement.addEventListener('pause', function() {
        isPlaying = false;
        updatePlayIcon();
    });

    audioElement.addEventListener('ended', function() {
        isPlaying = false;
        updatePlayIcon();
        resetProgress();
    });

    audioElement.addEventListener('timeupdate', function() {
        updateProgress();
        updateTimeDisplay();
    });

    audioElement.addEventListener('loadedmetadata', function() {
        updateDurationDisplay();
    });

    // Allow progress bar click to seek
    const progressContainer = document.getElementById('ttsProgress');
    if (progressContainer) {
        progressContainer.parentElement.addEventListener('click', function(e) {
            if (!audioElement || !audioElement.duration) return;

            const rect = this.getBoundingClientRect();
            const pos = (e.clientX - rect.left) / rect.width;
            audioElement.currentTime = pos * audioElement.duration;
        });
    }
}

// Play TTS for a specific item
async function playTTS(itemId, text, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    // Check if we're already playing the same item
    const currentText = document.getElementById('ttsAudio').getAttribute('data-text');
    if (currentText === text && !audioElement.paused) {
        togglePlayPause();
        return;
    }

    // Stop current playback
    stopTTS();

    // Fetch audio from TTS API
    try {
        if (!ttsEnabled) {
            alert('TTS is not available. Please check server configuration.');
            return;
        }

        // Detect language and use appropriate voice
        const detectedLang = detectLanguage(text);
        const voiceToUse = getVoiceForLang(detectedLang);

        const response = await fetch('/api/tts/speak', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                text: text,
                voice: voiceToUse,
                model: savedModel
            })
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(error || 'Failed to generate speech');
        }

        // Get audio data (WAV format)
        const audioBlob = await response.blob();
        const audioUrl = URL.createObjectURL(audioBlob);

        // Set audio source and metadata
        audioElement.src = audioUrl;
        audioElement.setAttribute('data-text', text);
        audioElement.setAttribute('data-item-id', itemId);

        // Play automatically
        await audioElement.play();

        // Highlight the current playing item
        highlightCurrentItem(itemId);

    } catch (error) {
        console.error('TTS Error:', error);
        alert('Error: ' + error.message);
    }
}

// Play TTS from settings page
async function playTTSFromSettings(text, voice, model) {
    if (!ttsEnabled) {
        alert('TTS is not available. Please check server configuration.');
        return;
    }

    try {
        const response = await fetch('/api/tts/speak', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                text: text,
                voice: voice,
                model: model
            })
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(error || 'Échec de la génération vocale');
        }

        const audioBlob = await response.blob();
        const audioUrl = URL.createObjectURL(audioBlob);
        const audio = new Audio(audioUrl);
        audio.play();

    } catch (error) {
        console.error('TTS Error:', error);
        alert('Erreur: ' + error.message);
    }
}

// Toggle play/pause
function togglePlayPause() {
    if (!audioElement) return;

    if (audioElement.paused) {
        audioElement.play();
    } else {
        audioElement.pause();
    }
}

// Stop TTS playback
function stopTTS() {
    if (!audioElement) return;

    audioElement.pause();
    audioElement.currentTime = 0;
    resetProgress();
    removeCurrentItemHighlight();
}

// Stop playing all articles
function stopAllTTS() {
    if (isPlayingAll) {
        isPlayingAll = false;
        stopTTS();
        if (playAllResolve) {
            playAllResolve();
            playAllResolve = null;
        }
        const playAllBtn = document.getElementById('playAllBtn');
        const stopAllBtn = document.getElementById('stopAllBtn');
        if (playAllBtn) playAllBtn.style.display = 'inline-block';
        if (stopAllBtn) stopAllBtn.style.display = 'none';
    }
}

// Play all articles in sequence with 2s pause between each
async function playAllArticlesTTS() {
    if (isPlayingAll) return;

    isPlayingAll = true;
    const playAllBtn = document.getElementById('playAllBtn');
    const stopAllBtn = document.getElementById('stopAllBtn');
    if (playAllBtn) playAllBtn.style.display = 'none';
    if (stopAllBtn) stopAllBtn.style.display = 'inline-block';

    const articles = document.querySelectorAll('article[data-id]');

    for (let i = 0; i < articles.length && isPlayingAll; i++) {
        const article = articles[i];
        const itemId = article.getAttribute('data-id');
        const title = article.querySelector('.item-title')?.textContent || '';
        const description = article.querySelector('.item-description')?.textContent || '';
        const text = `${title} ${description}`;

        if (text.trim()) {
            try {
                highlightCurrentItem(itemId);
                await playTTS(itemId, text, null);

                // Wait for audio to finish
                await new Promise(resolve => {
                    const checkEnded = () => {
                        if (!audioElement || isPlayingAll === false) {
                            resolve();
                            return;
                        }
                        if (audioElement.ended) {
                            resolve();
                        } else {
                            setTimeout(checkEnded, 100);
                        }
                    };
                    checkEnded();
                });

                // Wait 2 seconds AFTER the audio ends (unless stopped)
                if (i < articles.length - 1 && isPlayingAll) {
                    await new Promise(resolve => {
                        playAllResolve = resolve;
                        setTimeout(() => {
                            if (isPlayingAll) resolve();
                        }, 1000);
                    });
                    playAllResolve = null;
                }
            } catch (error) {
                console.error('Error playing article:', error);
            }
        }
    }

    isPlayingAll = false;
    playAllResolve = null;
    if (playAllBtn) playAllBtn.style.display = 'inline-block';
    if (stopAllBtn) stopAllBtn.style.display = 'none';
}

// Update play/pause icon
function updatePlayIcon() {
    const icon = document.getElementById('ttsPlayIcon');
    if (!icon) return;

    if (isPlaying) {
        icon.className = 'fas fa-pause';
    } else {
        icon.className = 'fas fa-play';
    }
}

// Update progress bar
function updateProgress() {
    if (!audioElement) return;

    const progressBar = document.getElementById('ttsProgress');
    if (!progressBar) return;

    const percent = (audioElement.currentTime / audioElement.duration) * 100;
    progressBar.style.width = percent + '%';
}

// Reset progress bar
function resetProgress() {
    const progressBar = document.getElementById('ttsProgress');
    if (progressBar) {
        progressBar.style.width = '0%';
    }
}

// Update time display
function updateTimeDisplay() {
    const currentTimeEl = document.getElementById('ttsCurrentTime');
    const durationEl = document.getElementById('ttsDuration');

    if (currentTimeEl && audioElement) {
        currentTimeEl.textContent = formatTime(audioElement.currentTime);
    }
}

// Update duration display
function updateDurationDisplay() {
    const durationEl = document.getElementById('ttsDuration');
    if (durationEl && audioElement) {
        durationEl.textContent = formatTime(audioElement.duration);
    }
}

// Highlight the current playing item
function highlightCurrentItem(itemId) {
    removeCurrentItemHighlight();

    const item = document.querySelector(`article[data-id="${itemId}"]`);
    if (item) {
        item.classList.add('tts-playing');
        item.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
}

// Remove highlight from current item
function removeCurrentItemHighlight() {
    const playingItems = document.querySelectorAll('article.tts-playing');
    playingItems.forEach(item => {
        item.classList.remove('tts-playing');
    });
}

// Initialize TTS on page load
function initTTS() {
    checkTTSSatus();
    loadTTSSettings();
    initTTSPlayer();
}
