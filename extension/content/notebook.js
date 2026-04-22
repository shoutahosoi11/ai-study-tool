(function () {
  var SIGNIN_PATTERNS = ['/signin', '/ap/signin', '/gp/sign-in'];
  var BOOK_CARD_SELECTOR = [
    '[data-asin]',
    '[data-get-annotations-for-asin]',
    '[data-action="get-annotations-for-asin"]',
    '.kp-notebook-library-each-book',
    '.kp-notebook-booklist-item',
    '.kp-notebook-library-list-item',
    '.kp-notebook-library-item',
    '.kp-notebook-library-book'
  ].join(', ');
  var BOOK_LINK_SELECTOR = 'a[href*="asin="], a[href*="/notebook"], a[href*="/kp/notebook"], [data-asin], [data-get-annotations-for-asin], [data-action="get-annotations-for-asin"]';
  var HIGHLIGHT_SELECTOR = '.kp-notebook-highlight, [id^="kp-notebook-annotated"]';
  var GENERIC_BOOK_LABELS = {
    'メモとハイライト': true,
    'ノートとハイライト': true,
    'my notes and highlights': true,
    'your notes and highlights': true,
    'notes and highlights': true,
    'kindle': true,
    'サインアウト': true,
    'sign out': true,
  };

  function isLoginPage() {
    var href = window.location.href;
    for (var i = 0; i < SIGNIN_PATTERNS.length; i++) {
      if (href.indexOf(SIGNIN_PATTERNS[i]) !== -1) return true;
    }
    return false;
  }

  function isNotebookPage() {
    return window.location.pathname.indexOf('/kp/notebook') !== -1 || window.location.pathname.indexOf('/notebook') !== -1;
  }

  function waitForNotebookReady(callback) {
    var attempts = 0;
    var timer = window.setInterval(function () {
      attempts += 1;

      if (isNotebookPage()) {
        window.clearInterval(timer);
        callback();
        return;
      }

      if (attempts >= 180) {
        window.clearInterval(timer);
        chrome.runtime.sendMessage({
          type: 'NOTEBOOK_ERROR',
          error: isLoginPage() ? 'NOT_LOGGED_IN' : 'NOTEBOOK_NOT_REACHED',
        });
      }
    }, 1000);
  }

  function getASINFromURL() {
    return new URLSearchParams(window.location.search).get('asin') || '';
  }

  function getTargetASINFromHash() {
    var hash = window.location.hash.replace(/^#/, '');
    return new URLSearchParams(hash).get('asin') || '';
  }

  function getTargetBookTitleFromHash() {
    var hash = window.location.hash.replace(/^#/, '');
    return new URLSearchParams(hash).get('book_title') || '';
  }

  function getTargetBookAuthorFromHash() {
    var hash = window.location.hash.replace(/^#/, '');
    return new URLSearchParams(hash).get('book_author') || '';
  }

  function getModeFromHash() {
    var hash = window.location.hash.replace(/^#/, '');
    return new URLSearchParams(hash).get('mode') || '';
  }

  function getNotebookURLFromElement(el) {
    if (!el) return '';

    var href = el.getAttribute('href') || '';
    if (href) {
      if (/^(javascript:|data:|mailto:|#)/i.test(href)) {
        href = '';
      }
    }
    if (href) {
      try {
        var resolved = new URL(href, window.location.origin).toString();
        if (getASINFromNotebookURL(resolved)) return resolved;
      } catch (_) {}
    }

    var anchor = el.querySelector('a[href]');
    if (anchor) {
      var anchorHref = anchor.getAttribute('href') || '';
      if (/^(javascript:|data:|mailto:|#)/i.test(anchorHref)) {
        anchorHref = '';
      }
      if (!anchorHref) return '';
      try {
        var resolvedAnchor = new URL(anchorHref, window.location.origin).toString();
        if (getASINFromNotebookURL(resolvedAnchor)) return resolvedAnchor;
      } catch (_) {}
    }

    return '';
  }

  function getASINFromNotebookURL(urlString) {
    if (!urlString) return '';
    try {
      return new URL(urlString, window.location.origin).searchParams.get('asin') || '';
    } catch (_) {
      return '';
    }
  }

  function resolveBookASIN(el) {
    if (!el) return '';

    var directASIN = getASINFromElement(el);
    if (directASIN) return directASIN;

    var notebookURL = getNotebookURLFromElement(el);
    return getASINFromNotebookURL(notebookURL);
  }

  function normalizeText(value) {
    return (value || '').replace(/\s+/g, ' ').trim();
  }

  function normalizeComparisonText(value) {
    return normalizeText(value).toLowerCase();
  }

  function buildNotebookURLFromASIN(asin) {
    var normalizedASIN = normalizeText(asin);
    if (!normalizedASIN) return '';
    return 'https://read.amazon.co.jp/notebook?asin=' + encodeURIComponent(normalizedASIN);
  }

  function parseASINPayload(value) {
    if (!value) return '';
    try {
      var parsed = JSON.parse(value);
      return normalizeText(parsed && parsed.asin);
    } catch (_) {
      return '';
    }
  }

  function getASINFromElement(el) {
    if (!el) return '';

    var direct = el.getAttribute('data-asin') || '';
    if (direct) return direct;

    var payloadASIN = parseASINPayload(el.getAttribute('data-get-annotations-for-asin'));
    if (payloadASIN) return payloadASIN;

    var href = el.getAttribute('href') || '';
    if (href) {
      try {
        var url = new URL(href, window.location.origin);
        var asin = url.searchParams.get('asin') || '';
        if (asin) return asin;
      } catch (_) {}
    }

    var anchor = el.querySelector('a[href*="asin="]');
    if (anchor) {
      try {
        var anchorURL = new URL(anchor.getAttribute('href') || '', window.location.origin);
        return anchorURL.searchParams.get('asin') || '';
      } catch (_) {}
    }

    var payloadNode = el.querySelector('[data-get-annotations-for-asin]');
    if (payloadNode) {
      var nestedPayloadASIN = parseASINPayload(payloadNode.getAttribute('data-get-annotations-for-asin'));
      if (nestedPayloadASIN) return nestedPayloadASIN;
    }

    return '';
  }

  function escapeAttrValue(value) {
    return String(value || '').replace(/\\/g, '\\\\').replace(/"/g, '\\"');
  }

  function hasHighlightNodes() {
    return document.querySelectorAll(HIGHLIGHT_SELECTOR).length > 0;
  }

  function isGenericBookLabel(value) {
    var normalized = normalizeText(value).toLowerCase();
    return Boolean(normalized && GENERIC_BOOK_LABELS[normalized]);
  }

  function countNestedBookCandidates(el) {
    if (!el || !el.querySelectorAll) return 0;
    return el.querySelectorAll(BOOK_LINK_SELECTOR).length;
  }

  function hasSingleBookCandidate(el) {
    return countNestedBookCandidates(el) <= 1;
  }

  function chooseLeafBookRoot(el) {
    if (!el) return null;

    var liRoot = el.closest('li, article');
    if (liRoot && hasSingleBookCandidate(liRoot)) {
      return liRoot;
    }

    var anchorRoot = el.closest('a[href]');
    if (anchorRoot) {
      return anchorRoot;
    }

    return null;
  }

  function getBookRootElement(el) {
    if (!el) return null;
    if (el.matches && el.matches('[data-asin]')) return el;

    var root = el.closest(
      '[data-asin], .kp-notebook-library-each-book, .kp-notebook-booklist-item, .kp-notebook-library-list-item, .kp-notebook-library-item, .kp-notebook-library-book'
    );

    if (root) {
      if (hasSingleBookCandidate(root)) {
        return root;
      }

      var leafFromKnownRoot = chooseLeafBookRoot(el);
      if (leafFromKnownRoot) {
        return leafFromKnownRoot;
      }

      return root;
    }

    var leaf = chooseLeafBookRoot(el);
    if (leaf) {
      return leaf;
    }

    return el.closest('li, article, a, div') || el;
  }

  function hasKnownBookContainerClass(el) {
    if (!el) return false;

    var current = el;
    while (current && current !== document.body) {
      var className = typeof current.className === 'string' ? current.className : '';
      if (/kp-notebook-(library|book)/.test(className)) {
        return true;
      }
      current = current.parentElement;
    }

    return false;
  }

  function isLikelyBookTitle(value) {
    var normalized = normalizeText(value);
    if (!normalized) return false;
    if (isGenericBookLabel(normalized)) return false;
    if (normalized.length > 160) return false;
    return true;
  }

  function isContainerTooLarge(el) {
    if (!el) return false;
    var text = normalizeText(el.textContent);
    return text.length > 300;
  }

  function buildBookCandidateKey(asin, notebookURL, title, author, index) {
    var parts = [asin || '', notebookURL || '', title || '', author || '']
      .map(normalizeText)
      .filter(Boolean);
    if (parts.length === 0) {
      return 'book:' + String(index);
    }
    return parts.join('::');
  }

  function extractBookTitleAndAuthor(el) {
    var titleEl = el.querySelector(
      '.kp-notebook-searchable, .kp-notebook-booktitle, h2, h3, [class*="title"], [title]'
    );
    var authorEl = el.querySelector(
      '.kp-notebook-bookauthor, [class*="author"], .a-size-base, .a-color-secondary'
    );

    var title = titleEl ? normalizeText(titleEl.textContent || titleEl.getAttribute('title')) : '';
    var author = authorEl ? normalizeText(authorEl.textContent) : '';

    if (!title) {
      var text = normalizeText(el.textContent);
      if (text) {
        var lines = text.split(/\s{2,}|\n/).map(normalizeText).filter(Boolean);
        title = lines[0] || '';
        if (!author && lines.length > 1) author = lines[1];
      }
    }

    if (author && isGenericBookLabel(author)) {
      author = '';
    }

    return {
      title: title,
      author: author,
    };
  }

  function getScrollableBookContainers() {
    var containers = [];
    var elements = document.querySelectorAll('div, section, main, aside, ul');

    for (var i = 0; i < elements.length; i++) {
      var el = elements[i];
      if (!el || !el.querySelector) continue;
      if (!el.querySelector(BOOK_LINK_SELECTOR) && !el.querySelector(BOOK_CARD_SELECTOR)) continue;
      if (el.scrollHeight <= el.clientHeight + 20) continue;

      var style = window.getComputedStyle(el);
      if (style.overflowY !== 'auto' && style.overflowY !== 'scroll') continue;

      containers.push(el);
    }

    return containers;
  }

  function nudgeBookListScrolling(step) {
    var amount = Math.max(200, Math.min(600, step * 40));
    var containers = getScrollableBookContainers();

    for (var i = 0; i < containers.length; i++) {
      var el = containers[i];
      var nextTop = Math.min(el.scrollTop + amount, Math.max(0, el.scrollHeight - el.clientHeight));
      if (nextTop !== el.scrollTop) {
        el.scrollTop = nextTop;
      }
    }

    var scrollingElement = document.scrollingElement || document.documentElement || document.body;
    if (scrollingElement && typeof scrollingElement.scrollTop === 'number') {
      var nextPageTop = Math.min(
        scrollingElement.scrollTop + Math.floor(amount / 2),
        Math.max(0, scrollingElement.scrollHeight - scrollingElement.clientHeight)
      );
      scrollingElement.scrollTop = nextPageTop;
    }
  }

  function collectBookElements() {
    var roots = [];
    function pushRoot(candidate) {
      if (!candidate) return;
      if (roots.indexOf(candidate) !== -1) return;
      roots.push(candidate);
    }
    function pushFromCandidate(candidate) {
      if (!candidate) return;

      var root = getBookRootElement(candidate);
      if (!root) return;

      if (!getASINFromElement(root) && countNestedBookCandidates(root) > 1) {
        var nestedLinks = root.querySelectorAll(BOOK_LINK_SELECTOR);
        for (var n = 0; n < nestedLinks.length; n++) {
          var nestedRoot = getBookRootElement(nestedLinks[n]);
          if (nestedRoot && nestedRoot !== root) {
            pushRoot(nestedRoot);
          }
        }
        return;
      }

      pushRoot(root);
    }

    var cards = document.querySelectorAll(BOOK_CARD_SELECTOR);
    for (var i = 0; i < cards.length; i++) {
      pushFromCandidate(cards[i]);
    }

    var strongLinks = document.querySelectorAll('a[href*="asin="], [data-asin]');
    for (var k = 0; k < strongLinks.length; k++) {
      pushFromCandidate(strongLinks[k]);
    }

    var links = document.querySelectorAll(BOOK_LINK_SELECTOR);
    for (var j = 0; j < links.length; j++) {
      pushFromCandidate(links[j]);
    }

    return roots;
  }

  function shouldSkipBookCandidate(el, asin, title, author, notebookURL) {
    var effectiveASIN = asin || getASINFromNotebookURL(notebookURL);
    var effectiveNotebookURL = notebookURL || '';
    var knownBookContainer = hasKnownBookContainerClass(el);

    if (isGenericBookLabel(title) || isGenericBookLabel(author)) {
      return true;
    }

    if (isContainerTooLarge(el)) {
      return true;
    }

    if (!effectiveASIN && countNestedBookCandidates(el) > 1) {
      return true;
    }

    if (!effectiveASIN && !knownBookContainer) {
      return true;
    }

    if (effectiveNotebookURL && effectiveNotebookURL.indexOf('asin=') === -1 && !knownBookContainer) {
      return true;
    }

    if (!effectiveASIN && !effectiveNotebookURL && !title) {
      return true;
    }

    if (!isLikelyBookTitle(title)) {
      return true;
    }

    return false;
  }

  function sendProgress(stage, count) {
    chrome.runtime.sendMessage({
      type: 'NOTEBOOK_PROGRESS',
      stage: stage,
      count: typeof count === 'number' ? count : undefined,
    });
  }

  // ===== 本一覧モード =====

  function scrapeBookList() {
    var books = [];
    var seen = {};
    var bookEls = collectBookElements();

    for (var i = 0; i < bookEls.length; i++) {
      var el = getBookRootElement(bookEls[i]);
      var asin = getASINFromElement(el);
      var notebookURL = getNotebookURLFromElement(el);
      var urlASIN = getASINFromNotebookURL(notebookURL);
      if (!asin && urlASIN) asin = urlASIN;
      if (!notebookURL && asin) {
        notebookURL = buildNotebookURLFromASIN(asin);
      }

      var metadata = extractBookTitleAndAuthor(el);
      var title = metadata.title;
      var author = metadata.author;

      if (shouldSkipBookCandidate(el, asin, title, author, notebookURL)) {
        continue;
      }

      var bookID = buildBookCandidateKey(asin, notebookURL, title, author, i);
      if (seen[bookID]) continue;
      seen[bookID] = true;

      books.push({
        id: bookID,
        asin: asin,
        book_title: title,
        book_author: author,
        notebook_url: notebookURL,
      });
    }

    return books;
  }

  function waitForBookList(callback) {
    var attempts = 0;
    var lastCount = -1;
    var stableCount = 0;
    var bestBooks = [];
    var timer = window.setInterval(function () {
      attempts += 1;
      nudgeBookListScrolling(attempts);
      var books = scrapeBookList();
      if (books.length > bestBooks.length) {
        bestBooks = books;
      }

      if (books.length === lastCount) {
        stableCount += 1;
      } else {
        stableCount = 0;
        lastCount = books.length;
      }

      if (books.length > 0 && stableCount >= 3) {
        window.clearInterval(timer);
        callback(bestBooks);
        return;
      }

      if (attempts >= 60) {
        window.clearInterval(timer);
        callback(bestBooks);
      }
    }, 500);
  }

  // ===== ハイライト取得モード =====

  function findText(root, selectors) {
    for (var i = 0; i < selectors.length; i++) {
      var el = root.querySelector(selectors[i]);
      if (el) {
        var t = normalizeText(el.textContent);
        if (t) return t;
      }
    }
    return '';
  }

  function getHighlightText(el) {
    var text = findText(el, [
      '#highlight',
      '.kp-notebook-highlight .a-text-bold',
      '[id^="highlight"]',
      '.kp-notebook-selected-text',
    ]);
    if (text) return text;

    var candidates = el.querySelectorAll('span, div');
    for (var i = 0; i < candidates.length; i++) {
      var t = normalizeText(candidates[i].textContent);
      if (t.length > 10) return t;
    }
    return '';
  }

  function getLocation(el) {
    return findText(el, ['.kp-notebook-metadata', '.kp-notebook-highlight-header']);
  }

  function getBookInfo() {
    var title = findText(document, [
      '.kp-notebook-book-title',
      '.kp-notebook-library-each-book h2',
      'h3.kp-notebook-selectable',
      '[data-testid="book-title"]',
    ]);
    if (!title) title = normalizeText(document.title.split('|')[0]);

    var author = findText(document, [
      '.kp-notebook-book-author',
      'p.kp-notebook-selectable',
      '.a-color-secondary',
    ]);
    if (author) {
      author = author.replace(/^著者[:：]?\s*/i, '').replace(/^by\s+/i, '');
      if (author === title) author = '';
    }

    return { book_title: title, book_author: author };
  }

  function isCurrentBookMatch(targetAsin, targetTitle, targetAuthor) {
    var normalizedTargetAsin = normalizeText(targetAsin);
    if (normalizedTargetAsin) {
      if (getASINFromURL() !== normalizedTargetAsin) {
        return false;
      }
    }

    var normalizedTargetTitle = normalizeComparisonText(targetTitle);
    if (!normalizedTargetTitle) {
      return Boolean(normalizedTargetAsin);
    }

    var currentBook = getBookInfo();
    if (normalizeComparisonText(currentBook.book_title) !== normalizedTargetTitle) {
      return false;
    }

    var normalizedTargetAuthor = normalizeComparisonText(targetAuthor);
    if (!normalizedTargetAuthor) {
      return true;
    }

    return normalizeComparisonText(currentBook.book_author) === normalizedTargetAuthor;
  }

  function scrapeHighlights(asin) {
    var bookInfo = getBookInfo();
    var nodes = document.querySelectorAll(HIGHLIGHT_SELECTOR);
    var seen = [];
    var highlights = [];

    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      if (seen.indexOf(el) !== -1) continue;
      seen.push(el);

      var content = getHighlightText(el);
      var location = getLocation(el);
      if (!content && !location) continue;

      highlights.push({
        asin: asin,
        book_title: bookInfo.book_title,
        book_author: bookInfo.book_author,
        content: content,
        location: location,
        highlighted_at: null,
      });
    }

    return highlights;
  }

  function waitForHighlights(callback) {
    var attempts = 0;
    var timer = window.setInterval(function () {
      attempts += 1;
      var els = document.querySelectorAll(HIGHLIGHT_SELECTOR);
      if (els.length > 0 || attempts >= 20) {
        window.clearInterval(timer);
        callback();
      }
    }, 500);
  }

  function findBookElementByASIN(asin) {
    var direct = document.querySelector('[data-asin="' + escapeAttrValue(asin) + '"]');
    if (direct) return direct;

    var anchor = document.querySelector('a[href*="asin=' + escapeAttrValue(asin) + '"]');
    if (anchor) {
      return getBookRootElement(anchor) || anchor;
    }

    var bookEls = collectBookElements();
    for (var i = 0; i < bookEls.length; i++) {
      var el = bookEls[i];
      if (getASINFromElement(el) === asin) {
        return el;
      }

      var nested = el.querySelector('[data-asin="' + escapeAttrValue(asin) + '"]');
      if (nested) {
        return getBookRootElement(nested) || nested;
      }

      var nestedAnchor = el.querySelector('a[href*="asin=' + escapeAttrValue(asin) + '"]');
      if (nestedAnchor) {
        return getBookRootElement(nestedAnchor) || nestedAnchor;
      }
    }

    return null;
  }

  function findBookElementByTitle(title, author) {
    var normalizedTitle = normalizeComparisonText(title);
    var normalizedAuthor = normalizeComparisonText(author);
    if (!normalizedTitle) return null;

    var bookEls = collectBookElements();
    for (var i = 0; i < bookEls.length; i++) {
      var el = getBookRootElement(bookEls[i]);
      if (!el) continue;

      var metadata = extractBookTitleAndAuthor(el);
      if (normalizeComparisonText(metadata.title) !== normalizedTitle) {
        continue;
      }

      if (normalizedAuthor && metadata.author && normalizeComparisonText(metadata.author) !== normalizedAuthor) {
        continue;
      }

      return el;
    }

    return null;
  }

  function openBookElement(el) {
    if (!el) return false;

    var anchor = el.querySelector('a[href*="/kp/notebook"], a[href*="asin="], a[href]');
    if (anchor && typeof anchor.click === 'function') {
      anchor.click();
      return true;
    }

    var button = el.querySelector('button, [role="button"], [role="link"]');
    if (button && typeof button.click === 'function') {
      button.click();
      return true;
    }

    if (typeof el.click === 'function') {
      el.click();
      return true;
    }

    return false;
  }

  function navigateToBook(notebookURL, targetAsin, targetTitle, targetAuthor) {
    var nextURL = appendHashParams(notebookURL, {
      mode: 'sync',
      asin: targetAsin || '',
      book_title: targetTitle || '',
      book_author: targetAuthor || '',
    });

    if (!nextURL) return false;

    if (window.location.href !== nextURL) {
      window.location.href = nextURL;
    }

    return true;
  }

  function sendHighlights(targetAsin) {
    var effectiveASIN = targetAsin || getASINFromURL() || '';
    var highlights = scrapeHighlights(effectiveASIN);
    sendProgress('highlight_data_found', highlights.length);
    chrome.runtime.sendMessage({
      type: 'NOTEBOOK_HIGHLIGHT_DATA',
      highlights: highlights,
    });
  }

  function waitForOpenedBook(targetAsin, targetTitle, targetAuthor) {
    var attempts = 0;
    var timer = window.setInterval(function () {
      attempts += 1;
      if (isCurrentBookMatch(targetAsin, targetTitle, targetAuthor) && hasHighlightNodes()) {
        window.clearInterval(timer);
        sendProgress('book_opened');
        waitForHighlights(function () {
          sendHighlights(targetAsin);
        });
        return;
      }

      if (attempts >= 30) {
        window.clearInterval(timer);
        chrome.runtime.sendMessage({ type: 'NOTEBOOK_ERROR', error: 'BOOK_OPEN_FAILED' });
      }
    }, 500);
  }

  function syncBookFromNotebook(targetAsin, targetTitle, targetAuthor) {
    if (!targetAsin) {
      chrome.runtime.sendMessage({ type: 'NOTEBOOK_ERROR', error: 'BOOK_NOT_FOUND' });
      return;
    }

    if (getASINFromURL() === normalizeText(targetAsin)) {
      sendProgress('book_ready');
      waitForHighlights(function () {
        sendHighlights(targetAsin);
      });
      return;
    }

    waitForBookList(function () {
      var bookEl = findBookElementByASIN(targetAsin);
      var resolvedTargetASIN = targetAsin || resolveBookASIN(bookEl);
      var resolvedNotebookURL = getNotebookURLFromElement(bookEl) || buildNotebookURLFromASIN(resolvedTargetASIN);
      if (!bookEl) {
        chrome.runtime.sendMessage({ type: 'NOTEBOOK_ERROR', error: 'BOOK_NOT_FOUND' });
        return;
      }

      if (navigateToBook(resolvedNotebookURL, resolvedTargetASIN, targetTitle, targetAuthor)) {
        waitForOpenedBook(resolvedTargetASIN, targetTitle, targetAuthor);
        return;
      }

      if (!openBookElement(bookEl)) {
        chrome.runtime.sendMessage({ type: 'NOTEBOOK_ERROR', error: 'BOOK_OPEN_FAILED' });
        return;
      }

      waitForOpenedBook(resolvedTargetASIN, targetTitle, targetAuthor);
    });
  }

  // ===== エントリポイント =====

  function runNotebookFlow() {
    if (!isNotebookPage()) {
      waitForNotebookReady(runNotebookFlow);
      return;
    }

    var mode = getModeFromHash();
    var targetAsin = getTargetASINFromHash();
    var targetBookTitle = getTargetBookTitleFromHash();
    var targetBookAuthor = getTargetBookAuthorFromHash();
    var asin = getASINFromURL();

    if (mode === 'list') {
      sendProgress('notebook_ready');
      waitForBookList(function (books) {
        if (!books || books.length === 0) {
          sendProgress('book_list_empty', 0);
          chrome.runtime.sendMessage({
            type: 'NOTEBOOK_ERROR',
            error: 'BOOK_LIST_EMPTY',
          });
          return;
        }
        sendProgress('book_list_found', books.length);
        chrome.runtime.sendMessage({
          type: 'NOTEBOOK_BOOK_LIST',
          books: books,
        });
      });
      return;
    }

    if (mode === 'sync' && (targetAsin || targetBookTitle)) {
      sendProgress('notebook_ready');
      syncBookFromNotebook(targetAsin, targetBookTitle, targetBookAuthor);
      return;
    }

    if (mode === 'sync') {
      waitForHighlights(function () {
        sendHighlights(getASINFromURL() || asin);
      });
      return;
    }

    if (asin) {
      waitForHighlights(function () {
        sendHighlights(asin);
      });
    } else {
      waitForBookList(function () {
        chrome.runtime.sendMessage({
          type: 'NOTEBOOK_BOOK_LIST',
          books: scrapeBookList(),
        });
      });
    }
  }

  if (isLoginPage()) {
    waitForNotebookReady(runNotebookFlow);
    return;
  }

  runNotebookFlow();
})();
