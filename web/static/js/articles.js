// articles.js - Gestion des articles RSS

// Mark article as read
function markAsRead(itemId, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    fetch('/item/' + itemId + '/read?view=' + window.viewMode + '&filter=' + window.filterMode + '&showRead=' + window.showRead, {
        method: 'GET',
        credentials: 'same-origin'
    }).then(function () {
        var article = document.querySelector('article[data-id="' + itemId + '"]');
        if (article) {
            article.classList.remove('unread');
            article.classList.add('read');
            var readBtn = article.querySelector('.mark-read');
            var unreadBtn = article.querySelector('.mark-unread');
            if (readBtn) readBtn.style.display = 'none';
            if (unreadBtn) unreadBtn.style.display = 'inline-flex';
            var titleLink = article.querySelector('.item-title a');
            if (titleLink) {
                titleLink.style.color = 'var(--text-secondary)';
            }
        }
        window.location.reload();
    });
    return false;
}

// Open article URL and mark as read
function openAndMarkRead(itemId, url, event) {
    if (event) {
        event.preventDefault();
    }

    var decodedUrl = decodeURIComponent(url);

    // Mark as read
    fetch('/item/' + itemId + '/read', {
        method: 'GET',
        credentials: 'same-origin'
    }).then(function () {
        var article = document.querySelector('article[data-id="' + itemId + '"]');
        if (article) {
            article.classList.remove('unread');
            article.classList.add('read');
            var readBtn = article.querySelector('.mark-read');
            var unreadBtn = article.querySelector('.mark-unread');
            if (readBtn) readBtn.style.display = 'none';
            if (unreadBtn) unreadBtn.style.display = 'inline-flex';
            var titleLink = article.querySelector('.item-title a');
            if (titleLink) {
                titleLink.style.color = 'var(--text-secondary)';
            }
        }
    });

    window.open(decodedUrl, '_blank', 'noopener,noreferrer');
    return false;
}

// Mark article as unread
function markAsUnread(itemId, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    fetch('/item/' + itemId + '/unread?view=' + window.viewMode + '&filter=' + window.filterMode + '&showRead=' + window.showRead, {
        method: 'GET',
        credentials: 'same-origin'
    }).then(function () {
        var article = document.querySelector('article[data-id="' + itemId + '"]');
        if (article) {
            article.classList.remove('read');
            article.classList.add('unread');
            var readBtn = article.querySelector('.mark-read');
            var unreadBtn = article.querySelector('.mark-unread');
            if (readBtn) readBtn.style.display = 'inline-flex';
            if (unreadBtn) unreadBtn.style.display = 'none';
            var titleLink = article.querySelector('.item-title a');
            if (titleLink) {
                titleLink.style.color = 'var(--text)';
            }
        }
        window.location.reload();
    });
    return false;
}
