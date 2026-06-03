// categories.js - Gestion des catégories pour FeedFusion

// Variables globales
let currentCategoryId = 0;
let allCategories = [];
let allFeeds = [];

/**
 * Initialise la page de gestion des catégories
 */
function initCategoriesPage(categories, feeds) {
    allCategories = categories || [];
    allFeeds = feeds || [];
    
    // Charger les associations flux-catégories
    loadCategoryFeeds();
    
    // Configurer les écouteurs d'événements
    setupCategoryEventListeners();
    updateColorValue();
}

/**
 * Configure tous les écouteurs d'événements pour la page catégories
 */
function setupCategoryEventListeners() {
    // Bouton ajouter catégorie
    const addCategoryBtn = document.getElementById('addCategoryBtn');
    if (addCategoryBtn) {
        addCategoryBtn.addEventListener('click', addCategory);
    }
    
    // Formulaire de catégorie
    const categoryForm = document.getElementById('categoryForm');
    if (categoryForm) {
        categoryForm.addEventListener('submit', saveCategory);
    }
    
    // Color picker
    const modalColor = document.getElementById('modalColor');
    if (modalColor) {
        modalColor.addEventListener('input', updateColorValue);
    }
    
    // Sélection flux/catégorie pour association
    const feedSelect = document.getElementById('feedSelect');
    const categorySelect = document.getElementById('categorySelect');
    if (feedSelect) {
        feedSelect.addEventListener('change', updateAssignButtons);
    }
    if (categorySelect) {
        categorySelect.addEventListener('change', updateAssignButtons);
    }
    
    // Boutons de modale
    const cancelCategoryBtn = document.getElementById('cancelCategoryBtn');
    if (cancelCategoryBtn) {
        cancelCategoryBtn.addEventListener('click', () => {
            const categoryModal = document.getElementById('categoryModal');
            if (categoryModal) categoryModal.style.display = 'none';
        });
    }
    
    const deleteCategoryCancelBtn = document.getElementById('deleteCategoryCancelBtn');
    if (deleteCategoryCancelBtn) {
        deleteCategoryCancelBtn.addEventListener('click', () => {
            const deleteCategoryModal = document.getElementById('deleteCategoryModal');
            if (deleteCategoryModal) deleteCategoryModal.style.display = 'none';
        });
    }
    
    // Fermer modales en cliquant à l'extérieur
    const categoryModal = document.getElementById('categoryModal');
    const deleteCategoryModal = document.getElementById('deleteCategoryModal');
    
    if (categoryModal) {
        categoryModal.addEventListener('click', (e) => {
            if (e.target.id === 'categoryModal') e.target.style.display = 'none';
        });
    }
    
    if (deleteCategoryModal) {
        deleteCategoryModal.addEventListener('click', (e) => {
            if (e.target.id === 'deleteCategoryModal') e.target.style.display = 'none';
        });
    }
    
    // Fermer avec Escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            if (categoryModal) categoryModal.style.display = 'none';
            if (deleteCategoryModal) deleteCategoryModal.style.display = 'none';
        }
    });
    
    // Bouton de suppression de catégorie
    const deleteCategoryConfirmBtn = document.getElementById('deleteCategoryConfirmBtn');
    if (deleteCategoryConfirmBtn) {
        deleteCategoryConfirmBtn.addEventListener('click', async () => {
            if (currentCategoryId === 0) return;
            
            try {
                const response = await fetch(`/api/categories/${currentCategoryId}`, {
                    method: 'DELETE'
                });
                
                if (!response.ok) {
                    const data = await response.json();
                    throw new Error(data.message || 'Erreur lors de la suppression');
                }
                
                showSuccess('Catégorie supprimée avec succès');
                if (deleteCategoryModal) deleteCategoryModal.style.display = 'none';
                location.reload();
                
            } catch (error) {
                showError(error.message);
                if (deleteCategoryModal) deleteCategoryModal.style.display = 'none';
            }
        });
    }
}

/**
 * Met à jour la valeur affichée du color picker
 */
function updateColorValue() {
    const colorInput = document.getElementById('modalColor');
    const colorValue = document.getElementById('colorValue');
    if (colorInput && colorValue) {
        colorValue.textContent = colorInput.value;
    }
}

/**
 * Active/désactive les boutons d'association en fonction des sélections
 */
function updateAssignButtons() {
    const feedId = document.getElementById('feedSelect')?.value;
    const categoryId = document.getElementById('categorySelect')?.value;
    const assignBtn = document.getElementById('assignBtn');
    const removeBtn = document.getElementById('removeBtn');
    
    if (assignBtn) assignBtn.disabled = !feedId || !categoryId;
    if (removeBtn) removeBtn.disabled = !feedId;
}

/**
 * Charge et affiche les flux par catégorie
 */
async function loadCategoryFeeds() {
    try {
        const container = document.getElementById('categoryFeedsContainer');
        
        if (!container) return;
        
        if (allCategories.length === 0) {
            container.innerHTML = '<p style="color: var(--text-secondary);">Aucune catégorie disponible</p>';
            return;
        }
        
        let html = '';
        for (const category of allCategories) {
            // Filtrer les flux qui appartiennent à cette catégorie
            const categoryFeeds = allFeeds.filter(f => f.CategoryID === category.ID);
            
            html += `
                <div class="category-feeds-item" data-category-id="${category.ID}">
                    <h5>
                        <span class="color-preview" style="background-color: ${category.Color}; width: 20px; height: 20px; border-radius: 50%; display: inline-block;"></span>
                        ${escapeHtml(category.Name)} (${categoryFeeds.length})
                    </h5>
                    <div class="feed-badges-container">
                        ${categoryFeeds.map(f => `
                            <span class="feed-badge">
                                ${escapeHtml(f.Title)}
                                <span class="remove-feed" onclick="removeFeedFromCategoryByFeedId(${f.ID})" title="Retirer de cette catégorie">
                                    <i class="fas fa-times"></i>
                                </span>
                            </span>
                        `).join('')}
                    </div>
                </div>
            `;
        }
        
        container.innerHTML = html;
        
        // Mettre à jour les comptes dans le tableau
        for (const category of allCategories) {
            const categoryFeeds = allFeeds.filter(f => f.CategoryID === category.ID);
            const countEl = document.getElementById(`feedCount_${category.ID}`);
            if (countEl) {
                countEl.textContent = categoryFeeds.length;
            }
        }
        
    } catch (error) {
        showError('Erreur lors du chargement: ' + error.message);
    }
}

/**
 * Échappe le HTML pour éviter les injections XSS
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * Affiche un message d'erreur
 */
function showError(message) {
    const el = document.getElementById('errorMessage');
    if (el) {
        el.textContent = message;
        setTimeout(() => el.textContent = '', 5000);
    }
}

/**
 * Affiche un message de succès
 */
function showSuccess(message) {
    const el = document.getElementById('successMessage');
    if (el) {
        el.textContent = message;
        setTimeout(() => el.textContent = '', 5000);
    }
}

/**
 * Ouvre la modale pour ajouter une nouvelle catégorie
 */
function addCategory() {
    currentCategoryId = 0;
    const modalTitle = document.getElementById('modalTitle');
    const categoryId = document.getElementById('categoryId');
    const modalName = document.getElementById('modalName');
    const modalColor = document.getElementById('modalColor');
    const colorValue = document.getElementById('colorValue');
    const categoryModal = document.getElementById('categoryModal');
    
    if (modalTitle) modalTitle.textContent = 'Ajouter une catégorie';
    if (categoryId) categoryId.value = '';
    if (modalName) modalName.value = '';
    if (modalColor) modalColor.value = '#6B8EBA';
    if (colorValue) colorValue.textContent = '#6B8EBA';
    if (categoryModal) {
        categoryModal.style.display = 'flex';
        if (modalName) modalName.focus();
    }
}

/**
 * Ouvre la modale pour modifier une catégorie existante
 */
function editCategory(id) {
    currentCategoryId = id;
    const category = allCategories.find(c => c.ID === id);
    if (!category) return;
    
    const modalTitle = document.getElementById('modalTitle');
    const categoryId = document.getElementById('categoryId');
    const modalName = document.getElementById('modalName');
    const modalColor = document.getElementById('modalColor');
    const colorValue = document.getElementById('colorValue');
    const categoryModal = document.getElementById('categoryModal');
    
    if (modalTitle) modalTitle.textContent = 'Modifier la catégorie';
    if (categoryId) categoryId.value = category.ID;
    if (modalName) modalName.value = category.Name;
    if (modalColor) modalColor.value = category.Color || '#6B8EBA';
    if (colorValue) colorValue.textContent = category.Color || '#6B8EBA';
    if (categoryModal) {
        categoryModal.style.display = 'flex';
        if (modalName) modalName.focus();
    }
}

/**
 * Ouvre la modale de confirmation de suppression d'une catégorie
 */
function deleteCategory(id, name) {
    currentCategoryId = id;
    const deleteCategoryConfirmText = document.getElementById('deleteCategoryConfirmText');
    const deleteCategoryModal = document.getElementById('deleteCategoryModal');
    
    if (deleteCategoryConfirmText) {
        deleteCategoryConfirmText.innerHTML = `
            <i class="fas fa-exclamation-circle"></i> 
            Êtes-vous sûr de vouloir supprimer la catégorie <strong>${escapeHtml(name)}</strong> ?
        `;
    }
    if (deleteCategoryModal) {
        deleteCategoryModal.style.display = 'flex';
    }
}

/**
 * Sauvegarde une catégorie (création ou mise à jour)
 */
async function saveCategory(e) {
    if (e) e.preventDefault();
    
    const categoryId = document.getElementById('categoryId')?.value;
    const name = document.getElementById('modalName')?.value?.trim();
    const color = document.getElementById('modalColor')?.value;
    
    if (!name) {
        showError('Le nom de la catégorie est obligatoire');
        return;
    }
    
    try {
        let url, method;
        let body = { name, color };
        
        if (categoryId) {
            url = `/api/categories/${categoryId}`;
            method = 'PUT';
        } else {
            url = '/api/categories';
            method = 'POST';
        }
        
        const response = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        
        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.message || 'Erreur lors de la sauvegarde');
        }
        
        const savedCategory = await response.json();
        
        showSuccess(categoryId ? 'Catégorie mise à jour avec succès' : 'Catégorie créée avec succès');
        closeCategoryModal();
        
        // Recharger la page pour voir les changements
        location.reload();
        
    } catch (error) {
        showError(error.message);
    }
}

/**
 * Ferme la modale de catégorie
 */
function closeCategoryModal() {
    const categoryModal = document.getElementById('categoryModal');
    if (categoryModal) categoryModal.style.display = 'none';
}

/**
 * Associe un flux à une catégorie
 */
async function assignFeedToCategory() {
    const feedId = document.getElementById('feedSelect')?.value;
    const categoryId = document.getElementById('categorySelect')?.value;
    
    if (!feedId || !categoryId) return;
    
    try {
        const response = await fetch(`/api/feed/${feedId}/category/${categoryId}`, {
            method: 'POST'
        });
        
        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.message || "Erreur lors de l'association");
        }
        
        showSuccess('Flux associé à la catégorie avec succès');
        // Recharger la page pour voir les changements
        location.reload();
        
    } catch (error) {
        showError(error.message);
    }
}

/**
 * Retire un flux de sa catégorie (via les sélecteurs)
 */
async function removeFeedFromCategory() {
    const feedId = document.getElementById('feedSelect')?.value;
    
    if (!feedId) return;
    
    try {
        const response = await fetch(`/api/feed/${feedId}/category`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.message || 'Erreur lors du retrait');
        }
        
        showSuccess('Flux retiré de sa catégorie avec succès');
        // Recharger la page pour voir les changements
        location.reload();
        
    } catch (error) {
        showError(error.message);
    }
}

/**
 * Retire un flux de sa catégorie (via le badge dans la liste)
 */
async function removeFeedFromCategoryByFeedId(feedId) {
    try {
        const response = await fetch(`/api/feed/${feedId}/category`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.message || 'Erreur lors du retrait');
        }
        
        showSuccess('Flux retiré de sa catégorie');
        // Recharger la page pour voir les changements
        location.reload();
        
    } catch (error) {
        showError(error.message);
    }
}

// Initialisation automatique si la page contient les éléments nécessaires
if (document.getElementById('categoriesTableBody') || document.getElementById('categoryFeedsContainer')) {
    // La page catégories est chargée, initialiser automatiquement
    // Les données seront passées via initCategoriesPage depuis le template
}
