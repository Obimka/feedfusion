// admin.js - Fonctions communes pour la gestion des droits administrateur

/**
 * Vérifie si l'utilisateur est admin et affiche/masque les éléments appropriés
 * Cette fonction vérifie en appelant /api/users - si la réponse est OK, l'utilisateur est admin
 */
async function checkAdminStatus() {
    try {
        const response = await fetch('/api/users');
        if (response.ok) {
            // Si on peut accéder à /api/users, c'est qu'on est admin
            const usersLink = document.getElementById('usersLink');
            if (usersLink) {
                usersLink.style.display = 'block';
            }
        }
    } catch (error) {
        console.log('Not admin or error:', error);
    }
}

// Exporter pour pouvoir être appelé depuis d'autres scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { checkAdminStatus };
}
