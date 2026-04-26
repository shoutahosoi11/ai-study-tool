(function (global) {
  var shared = global.KindleHighlightsMvp;
  if (!shared || !shared.utils) {
    return;
  }

  var utils = shared.utils;
  var stateKey = shared.STATE_KEY;
  var messageTypes = shared.MESSAGE_TYPES;
  var sessionStorageKey = 'kindle-highlights-mvp:session';

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
  var HIGHLIGHT_SELECTOR = '.kp-notebook-highlight, [id^="kp-notebook-annotated"], [data-annotation-id]';
  var GENERIC_BOOK_LABELS = {
    'メモとハイライト': true,
    'ノートとハイライト': true,
    'my notes and highlights': true,
    'your notes and highlights': true,
    'notes and highlights': true,
    'kindle': true,
    'サインアウト': true,
    'sign out': true
  };
  var NOTE_SELECTORS = [
    '.kp-notebook-note',
    '.kp-notebook-note-text',
    '.kp-notebook-note-content',
    '.kp-notebook-note-container',
    '.kp-notebook-annotation-note',
    '[data-annotation-note]',
    '[class*="note"]',
    '[id*="note"]'
  ];
  var HIGHLIGHT_TEXT_SELECTORS = [
    '#highlight',
    '.kp-notebook-highlight .a-text-bold',
    '[id^="highlight"]',
    '.kp-notebook-selected-text',
    '.kp-notebook-highlight-text'
  ];
  var METADATA_SELECTORS = [
    '.kp-notebook-metadata',
    '.kp-notebook-highlight-header',
    '.kp-notebook-highlight-meta',
    '[class*="metadata"]',
    '[class*="header"]'
  ];
  var COLOR_NAMES = ['yellow', 'blue', 'pink', 'orange'];

  var currentState = shared.buildInitialState();
  var isRunning = false;
  var lastResult = null;

  function initializeStoredState() {
    chrome.storage.local.get(stateKey, function (result) {
      if (!result || !result[stateKey]) {
        chrome.storage.local.set({ [stateKey]: currentState });
      } else {
        currentState = result[stateKey];
      }
    });
  }

  function setState(patch) {
    currentState = Object.assign({}, currentState, patch, {
      lastUpdatedAt: new Date().toISOString()
    });
    chrome.storage.local.set({ [stateKey]: currentState });
  }

  function resetState() {
    currentState = shared.buildInitialState();
    chrome.storage.local.set({ [stateKey]: currentState });
  }

  function readSession() {
    try {
      var raw = window.sessionStorage.getItem(sessionStorageKey);
      if (!raw) {
        return null;
      }
      return JSON.parse(raw);
    } catch (_) {
      return null;
    }
  }

  function writeSession(session) {
    try {
      window.sessionStorage.setItem(sessionStorageKey, JSON.stringify(session));
    } catch (error) {
      console.warn('[Kindle Highlights Scraper] Failed to persist session', error);
    }
  }

  function isLoginPage() {
    var href = window.location.href;
    for (var i = 0; i < SIGNIN_PATTERNS.length; i += 1) {
      if (href.indexOf(SIGNIN_PATTERNS[i]) !== -1) {
        return true;
      }
    }
    return false;
  }

  function isNotebookPage() {
    return shared.isNotebookUrl(window.location.href);
  }

  function parseASINPayload(value) {
    if (!value) {
      return '';
    }

    try {
      var parsed = JSON.parse(value);
      return utils.normalizeText(parsed && parsed.asin);
    } catch (_) {
      return '';
    }
  }

  function getASINFromURL() {
    try {
      return new URLSearchParams(window.location.search).get('asin') || '';
    } catch (_) {
      return '';
    }
  }

  function getASINFromNotebookURL(urlString) {
    if (!urlString) {
      return '';
    }

    try {
      return new URL(urlString, window.location.origin).searchParams.get('asin') || '';
    } catch (_) {
      return '';
    }
  }

  function buildNotebookURLFromASIN(asin) {
    var normalizedASIN = utils.normalizeText(asin);
    if (!normalizedASIN) {
      return '';
    }
    return 'https://' + window.location.hostname + '/notebook?asin=' + encodeURIComponent(normalizedASIN);
  }

  function getNotebookURLFromElement(element) {
    if (!element) {
      return '';
    }

    var href = element.getAttribute('href') || '';
    if (href && !/^(javascript:|data:|mailto:|#)/i.test(href)) {
      try {
        return new URL(href, window.location.origin).toString();
      } catch (_) {}
    }

    var anchor = element.querySelector('a[href]');
    if (!anchor) {
      return '';
    }

    var anchorHref = anchor.getAttribute('href') || '';
    if (!anchorHref || /^(javascript:|data:|mailto:|#)/i.test(anchorHref)) {
      return '';
    }

    try {
      return new URL(anchorHref, window.location.origin).toString();
    } catch (_) {
      return '';
    }
  }

  function getASINFromElement(element) {
    if (!element) {
      return '';
    }

    var direct = element.getAttribute('data-asin') || '';
    if (direct) {
      return utils.normalizeText(direct);
    }

    var payloadASIN = parseASINPayload(element.getAttribute('data-get-annotations-for-asin'));
    if (payloadASIN) {
      return payloadASIN;
    }

    var href = element.getAttribute('href') || '';
    if (href) {
      try {
        var url = new URL(href, window.location.origin);
        var asinFromHref = url.searchParams.get('asin') || '';
        if (asinFromHref) {
          return utils.normalizeText(asinFromHref);
        }
      } catch (_) {}
    }

    var nestedAnchor = element.querySelector('a[href*="asin="]');
    if (nestedAnchor) {
      try {
        var nestedURL = new URL(nestedAnchor.getAttribute('href') || '', window.location.origin);
        return utils.normalizeText(nestedURL.searchParams.get('asin') || '');
      } catch (_) {}
    }

    var nestedPayloadNode = element.querySelector('[data-get-annotations-for-asin]');
    if (nestedPayloadNode) {
      var nestedPayload = parseASINPayload(nestedPayloadNode.getAttribute('data-get-annotations-for-asin'));
      if (nestedPayload) {
        return nestedPayload;
      }
    }

    return '';
  }

  function resolveBookASIN(element) {
    var direct = getASINFromElement(element);
    if (direct) {
      return direct;
    }

    var notebookURL = getNotebookURLFromElement(element);
    return getASINFromNotebookURL(notebookURL);
  }

  function countNestedBookCandidates(element) {
    if (!element || !element.querySelectorAll) {
      return 0;
    }
    return element.querySelectorAll(BOOK_LINK_SELECTOR).length;
  }

  function hasSingleBookCandidate(element) {
    return countNestedBookCandidates(element) <= 1;
  }

  function chooseLeafBookRoot(element) {
    if (!element) {
      return null;
    }

    var liRoot = element.closest('li, article');
    if (liRoot && hasSingleBookCandidate(liRoot)) {
      return liRoot;
    }

    var anchorRoot = element.closest('a[href]');
    if (anchorRoot) {
      return anchorRoot;
    }

    return null;
  }

  function getBookRootElement(element) {
    if (!element) {
      return null;
    }

    if (element.matches && element.matches('[data-asin]')) {
      return element;
    }

    var root = element.closest(
      '[data-asin], .kp-notebook-library-each-book, .kp-notebook-booklist-item, .kp-notebook-library-list-item, .kp-notebook-library-item, .kp-notebook-library-book'
    );

    if (root) {
      if (hasSingleBookCandidate(root)) {
        return root;
      }

      var leafFromKnownRoot = chooseLeafBookRoot(element);
      if (leafFromKnownRoot) {
        return leafFromKnownRoot;
      }

      return root;
    }

    var leaf = chooseLeafBookRoot(element);
    if (leaf) {
      return leaf;
    }

    return element.closest('li, article, a, div') || element;
  }

  function hasKnownBookContainerClass(element) {
    if (!element) {
      return false;
    }

    var current = element;
    while (current && current !== document.body) {
      var className = typeof current.className === 'string' ? current.className : '';
      if (/kp-notebook-(library|book)/.test(className)) {
        return true;
      }
      current = current.parentElement;
    }

    return false;
  }

  function isGenericBookLabel(value) {
    var normalized = utils.normalizeComparisonText(value);
    return Boolean(normalized && GENERIC_BOOK_LABELS[normalized]);
  }

  function isLikelyBookTitle(value) {
    var normalized = utils.normalizeText(value);
    if (!normalized) {
      return false;
    }
    if (isGenericBookLabel(normalized)) {
      return false;
    }
    return normalized.length <= 160;
  }

  function isContainerTooLarge(element) {
    if (!element) {
      return false;
    }
    return utils.normalizeText(element.textContent).length > 300;
  }

  function extractBookTitleAndAuthor(element) {
    var titleElement = element.querySelector(
      '.kp-notebook-searchable, .kp-notebook-booktitle, h2, h3, [class*="title"], [title]'
    );
    var authorElement = element.querySelector(
      '.kp-notebook-bookauthor, [class*="author"], .a-size-base, .a-color-secondary'
    );

    var title = titleElement ? utils.normalizeText(titleElement.textContent || titleElement.getAttribute('title')) : '';
    var author = authorElement ? utils.normalizeText(authorElement.textContent) : '';

    if (!title) {
      var lines = utils.normalizeText(element.textContent)
        .split(/\s{2,}|\n/)
        .map(utils.normalizeText)
        .filter(Boolean);
      title = lines[0] || '';
      if (!author && lines.length > 1) {
        author = lines[1];
      }
    }

    if (author) {
      author = author.replace(/^著者[:：]?\s*/i, '').replace(/^by\s+/i, '');
      if (author === title || isGenericBookLabel(author)) {
        author = '';
      }
    }

    return {
      title: title,
      author: author
    };
  }

  function extractCoverImageUrl(element) {
    if (!element) {
      return '';
    }

    var image = element.querySelector('img[src]');
    return image ? utils.normalizeText(image.getAttribute('src')) : '';
  }

  function buildBookCandidateKey(asin, notebookURL, title, author, index) {
    var parts = [asin || '', notebookURL || '', title || '', author || '']
      .map(utils.normalizeText)
      .filter(Boolean);
    if (parts.length === 0) {
      return 'book:' + String(index);
    }
    return parts.join('::');
  }

  function shouldSkipBookCandidate(element, asin, title, author, notebookURL) {
    var effectiveASIN = asin || getASINFromNotebookURL(notebookURL);
    var knownBookContainer = hasKnownBookContainerClass(element);

    if (isGenericBookLabel(title) || isGenericBookLabel(author)) {
      return true;
    }
    if (isContainerTooLarge(element)) {
      return true;
    }
    if (!effectiveASIN && countNestedBookCandidates(element) > 1) {
      return true;
    }
    if (!effectiveASIN && !knownBookContainer && !title) {
      return true;
    }
    if (!isLikelyBookTitle(title)) {
      return true;
    }

    return false;
  }

  function collectBookElements() {
    var roots = [];

    function pushRoot(candidate) {
      if (!candidate) {
        return;
      }
      if (roots.indexOf(candidate) !== -1) {
        return;
      }
      roots.push(candidate);
    }

    function pushFromCandidate(candidate) {
      if (!candidate) {
        return;
      }

      var root = getBookRootElement(candidate);
      if (!root) {
        return;
      }

      if (!getASINFromElement(root) && countNestedBookCandidates(root) > 1) {
        var nestedLinks = root.querySelectorAll(BOOK_LINK_SELECTOR);
        for (var n = 0; n < nestedLinks.length; n += 1) {
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
    for (var i = 0; i < cards.length; i += 1) {
      pushFromCandidate(cards[i]);
    }

    var strongLinks = document.querySelectorAll('a[href*="asin="], [data-asin]');
    for (var j = 0; j < strongLinks.length; j += 1) {
      pushFromCandidate(strongLinks[j]);
    }

    var links = document.querySelectorAll(BOOK_LINK_SELECTOR);
    for (var k = 0; k < links.length; k += 1) {
      pushFromCandidate(links[k]);
    }

    return roots;
  }

  function getScrollableBookContainers() {
    var containers = [];
    var elements = document.querySelectorAll('div, section, main, aside, ul');

    for (var i = 0; i < elements.length; i += 1) {
      var element = elements[i];
      if (!element || !element.querySelector) {
        continue;
      }
      if (!element.querySelector(BOOK_LINK_SELECTOR) && !element.querySelector(BOOK_CARD_SELECTOR)) {
        continue;
      }
      if (element.scrollHeight <= element.clientHeight + 20) {
        continue;
      }

      var style = window.getComputedStyle(element);
      if (style.overflowY !== 'auto' && style.overflowY !== 'scroll') {
        continue;
      }

      containers.push(element);
    }

    return containers;
  }

  function nudgeBookListScrolling(step) {
    var amount = Math.max(200, Math.min(600, step * 40));
    var containers = getScrollableBookContainers();
    var moved = false;

    for (var i = 0; i < containers.length; i += 1) {
      var element = containers[i];
      var nextTop = Math.min(element.scrollTop + amount, Math.max(0, element.scrollHeight - element.clientHeight));
      if (nextTop !== element.scrollTop) {
        element.scrollTop = nextTop;
        moved = true;
      }
    }

    var scrollingElement = document.scrollingElement || document.documentElement || document.body;
    if (scrollingElement && typeof scrollingElement.scrollTop === 'number') {
      var nextPageTop = Math.min(
        scrollingElement.scrollTop + Math.floor(amount / 2),
        Math.max(0, scrollingElement.scrollHeight - scrollingElement.clientHeight)
      );
      if (nextPageTop !== scrollingElement.scrollTop) {
        scrollingElement.scrollTop = nextPageTop;
        moved = true;
      }
    }

    return moved;
  }

  function collectBookEntries() {
    var books = [];
    var seen = {};
    var bookElements = collectBookElements();

    for (var i = 0; i < bookElements.length; i += 1) {
      var root = getBookRootElement(bookElements[i]);
      if (!root) {
        continue;
      }

      var asin = getASINFromElement(root);
      var notebookURL = getNotebookURLFromElement(root);
      var urlASIN = getASINFromNotebookURL(notebookURL);
      if (!asin && urlASIN) {
        asin = urlASIN;
      }
      if (!notebookURL && asin) {
        notebookURL = buildNotebookURLFromASIN(asin);
      }

      var metadata = extractBookTitleAndAuthor(root);
      if (shouldSkipBookCandidate(root, asin, metadata.title, metadata.author, notebookURL)) {
        continue;
      }

      var bookID = buildBookCandidateKey(asin, notebookURL, metadata.title, metadata.author, i);
      if (seen[bookID]) {
        continue;
      }
      seen[bookID] = true;

      books.push({
        id: bookID,
        asin: asin,
        title: metadata.title,
        author: metadata.author,
        coverImageUrl: extractCoverImageUrl(root),
        notebookUrl: notebookURL,
        sidebarText: utils.normalizeText(root.textContent),
        sidebarHtml: root.outerHTML,
        sidebarAttributes: utils.serializeAttributes(root),
        sidebarDataset: utils.serializeDataset(root),
        _element: root
      });
    }

    return books;
  }

  function waitForBookEntries() {
    return new Promise(function (resolve) {
      var attempts = 0;
      var lastCount = -1;
      var stableCount = 0;
      var bestBooks = [];

      var timer = window.setInterval(function () {
        attempts += 1;
        nudgeBookListScrolling(attempts);

        var books = collectBookEntries();
        if (books.length > bestBooks.length) {
          bestBooks = books;
        }

        if (books.length === lastCount) {
          stableCount += 1;
        } else {
          stableCount = 0;
          lastCount = books.length;
        }

        if (books.length > 0 && stableCount >= 4) {
          window.clearInterval(timer);
          resolve(bestBooks);
          return;
        }

        if (attempts >= 70) {
          window.clearInterval(timer);
          resolve(bestBooks);
        }
      }, 400);
    });
  }

  function getBookInfo() {
    var titleElement = document.querySelector(
      '.kp-notebook-book-title, .kp-notebook-library-each-book h2, h3.kp-notebook-selectable, [data-testid="book-title"]'
    );
    var authorElement = document.querySelector(
      '.kp-notebook-book-author, p.kp-notebook-selectable, .a-color-secondary'
    );

    var title = titleElement ? utils.normalizeText(titleElement.textContent || titleElement.getAttribute('title')) : '';
    if (!title) {
      title = utils.normalizeText(document.title.split('|')[0]);
    }

    var author = authorElement ? utils.normalizeText(authorElement.textContent) : '';
    if (author) {
      author = author.replace(/^著者[:：]?\s*/i, '').replace(/^by\s+/i, '');
      if (author === title) {
        author = '';
      }
    }

    return {
      title: title,
      author: author
    };
  }

  function getCurrentHeaderElement() {
    var titleNode = document.querySelector(
      '.kp-notebook-book-title, h3.kp-notebook-selectable, [data-testid="book-title"], .kp-notebook-library-each-book h2'
    );
    return titleNode ? (titleNode.closest('header, section, article, div') || titleNode) : null;
  }

  function getCurrentBookHeaderInfo(fallbackEntry) {
    var currentInfo = getBookInfo();
    var headerElement = getCurrentHeaderElement();
    var coverImage = extractCoverImageUrl(headerElement) || extractCoverImageUrl(document.body);

    return {
      asin: utils.normalizeText(getASINFromURL()) || utils.normalizeText(fallbackEntry.asin || ''),
      title: currentInfo.title || fallbackEntry.title || '',
      author: currentInfo.author || fallbackEntry.author || '',
      coverImageUrl: coverImage || fallbackEntry.coverImageUrl || '',
      headerText: headerElement ? utils.normalizeText(headerElement.textContent) : '',
      headerHtml: headerElement ? headerElement.outerHTML : '',
      headerAttributes: headerElement ? utils.serializeAttributes(headerElement) : {},
      headerDataset: headerElement ? utils.serializeDataset(headerElement) : {}
    };
  }

  function isCurrentBookMatch(targetAsin, targetTitle, targetAuthor) {
    var normalizedTargetASIN = utils.normalizeText(targetAsin);
    if (normalizedTargetASIN) {
      var currentASIN = utils.normalizeText(getASINFromURL());
      if (currentASIN && currentASIN === normalizedTargetASIN) {
        return true;
      }
    }

    var currentInfo = getBookInfo();
    if (!currentInfo.title) {
      return false;
    }

    if (utils.normalizeComparisonText(currentInfo.title) !== utils.normalizeComparisonText(targetTitle)) {
      return false;
    }

    if (utils.normalizeText(targetAuthor)) {
      return utils.normalizeComparisonText(currentInfo.author) === utils.normalizeComparisonText(targetAuthor);
    }

    return true;
  }

  function findBookElementByASIN(asin) {
    var direct = document.querySelector('[data-asin="' + String(asin).replace(/"/g, '\\"') + '"]');
    if (direct) {
      return getBookRootElement(direct) || direct;
    }

    var anchor = document.querySelector('a[href*="asin=' + String(asin).replace(/"/g, '\\"') + '"]');
    if (anchor) {
      return getBookRootElement(anchor) || anchor;
    }

    var books = collectBookElements();
    for (var i = 0; i < books.length; i += 1) {
      var root = getBookRootElement(books[i]);
      if (resolveBookASIN(root) === asin) {
        return root;
      }
    }

    return null;
  }

  function findBookElementByTitle(title, author) {
    var normalizedTitle = utils.normalizeComparisonText(title);
    var normalizedAuthor = utils.normalizeComparisonText(author);
    if (!normalizedTitle) {
      return null;
    }

    var books = collectBookElements();
    for (var i = 0; i < books.length; i += 1) {
      var root = getBookRootElement(books[i]);
      if (!root) {
        continue;
      }

      var metadata = extractBookTitleAndAuthor(root);
      if (utils.normalizeComparisonText(metadata.title) !== normalizedTitle) {
        continue;
      }
      if (normalizedAuthor && metadata.author && utils.normalizeComparisonText(metadata.author) !== normalizedAuthor) {
        continue;
      }

      return root;
    }

    return null;
  }

  function openBookElement(element) {
    if (!element) {
      return false;
    }

    var anchor = element.querySelector('a[href*="/kp/notebook"], a[href*="asin="], a[href]');
    if (anchor && typeof anchor.click === 'function') {
      anchor.click();
      return true;
    }

    var button = element.querySelector('button, [role="button"], [role="link"]');
    if (button && typeof button.click === 'function') {
      button.click();
      return true;
    }

    if (typeof element.click === 'function') {
      element.click();
      return true;
    }

    return false;
  }

  async function findLiveBookElement(entry) {
    if (entry._element && entry._element.isConnected) {
      return entry._element;
    }

    for (var attempt = 0; attempt < 20; attempt += 1) {
      var byASIN = entry.asin ? findBookElementByASIN(entry.asin) : null;
      if (byASIN) {
        return byASIN;
      }

      var byTitle = entry.title ? findBookElementByTitle(entry.title, entry.author) : null;
      if (byTitle) {
        return byTitle;
      }

      nudgeBookListScrolling(attempt + 1);
      await utils.sleep(250);
    }

    return null;
  }

  function getHighlightNodes() {
    var nodes = document.querySelectorAll(HIGHLIGHT_SELECTOR);
    var items = [];
    var seen = [];

    for (var i = 0; i < nodes.length; i += 1) {
      var node = nodes[i];
      var root = node.closest('.kp-notebook-highlight, [id^="kp-notebook-annotated"], [data-annotation-id]') || node;
      if (seen.indexOf(root) !== -1) {
        continue;
      }
      seen.push(root);
      items.push(root);
    }

    return items;
  }

  function hasHighlightNodes() {
    return getHighlightNodes().length > 0;
  }

  function getScrollableHighlightContainers() {
    var containers = [];
    var elements = document.querySelectorAll('div, section, main, aside, article');

    for (var i = 0; i < elements.length; i += 1) {
      var element = elements[i];
      if (!element || !element.querySelector) {
        continue;
      }
      if (!element.querySelector(HIGHLIGHT_SELECTOR)) {
        continue;
      }
      if (element.scrollHeight <= element.clientHeight + 20) {
        continue;
      }

      var style = window.getComputedStyle(element);
      if (style.overflowY !== 'auto' && style.overflowY !== 'scroll') {
        continue;
      }

      containers.push(element);
    }

    return containers;
  }

  function scrollHighlights(step) {
    var amount = Math.max(320, Math.min(900, step * 55));
    var containers = getScrollableHighlightContainers();
    var moved = false;

    for (var i = 0; i < containers.length; i += 1) {
      var element = containers[i];
      var nextTop = Math.min(element.scrollTop + amount, Math.max(0, element.scrollHeight - element.clientHeight));
      if (nextTop !== element.scrollTop) {
        element.scrollTop = nextTop;
        moved = true;
      }
    }

    var scrollingElement = document.scrollingElement || document.documentElement || document.body;
    if (scrollingElement && typeof scrollingElement.scrollTop === 'number') {
      var nextPageTop = Math.min(
        scrollingElement.scrollTop + Math.floor(amount / 2),
        Math.max(0, scrollingElement.scrollHeight - scrollingElement.clientHeight)
      );
      if (nextPageTop !== scrollingElement.scrollTop) {
        scrollingElement.scrollTop = nextPageTop;
        moved = true;
      }
    }

    return moved;
  }

  async function loadAllHighlightsForCurrentBook() {
    var lastCount = -1;
    var stableCount = 0;

    for (var attempt = 0; attempt < 45; attempt += 1) {
      var currentCount = getHighlightNodes().length;
      if (currentCount === lastCount) {
        stableCount += 1;
      } else {
        stableCount = 0;
        lastCount = currentCount;
      }

      if (stableCount >= 4) {
        break;
      }

      var moved = scrollHighlights(attempt + 1);
      if (!moved && stableCount >= 2) {
        break;
      }

      await utils.sleep(350);
    }
  }

  function findText(root, selectors) {
    for (var i = 0; i < selectors.length; i += 1) {
      var element = root.querySelector(selectors[i]);
      if (!element) {
        continue;
      }
      var text = utils.normalizeText(element.textContent || element.getAttribute('title'));
      if (text) {
        return text;
      }
    }

    return '';
  }

  function getHighlightText(node) {
    var direct = findText(node, HIGHLIGHT_TEXT_SELECTORS);
    if (direct) {
      return direct;
    }

    var candidates = node.querySelectorAll('span, div, p');
    for (var i = 0; i < candidates.length; i += 1) {
      var text = utils.normalizeText(candidates[i].textContent);
      if (text.length > 10 && !/位置|location|page|メモ|note/i.test(text)) {
        return text;
      }
    }

    return '';
  }

  function extractMetadataText(node) {
    var direct = findText(node, METADATA_SELECTORS);
    if (direct) {
      return direct;
    }

    var candidates = node.querySelectorAll('div, span, p');
    for (var i = 0; i < candidates.length; i += 1) {
      var text = utils.normalizeText(candidates[i].textContent);
      if (!text) {
        continue;
      }
      if (/位置|location|page|ページ|highlighted|追加|作成|note/i.test(text)) {
        return text;
      }
    }

    return '';
  }

  function extractNoteInfo(node, highlightText, metadataText) {
    var seen = [];
    var selectors = NOTE_SELECTORS.slice();

    function pickCandidate(element) {
      if (!element || seen.indexOf(element) !== -1) {
        return null;
      }
      seen.push(element);

      var text = utils.normalizeText(element.textContent);
      if (!text) {
        return null;
      }
      if (text === highlightText || text === metadataText) {
        return null;
      }
      if (/位置|location|page|ページ/i.test(text) && text.length <= 40) {
        return null;
      }

      return {
        text: text,
        html: element.outerHTML
      };
    }

    for (var i = 0; i < selectors.length; i += 1) {
      var elements = node.querySelectorAll(selectors[i]);
      for (var j = 0; j < elements.length; j += 1) {
        var selected = pickCandidate(elements[j]);
        if (selected) {
          return selected;
        }
      }
    }

    var generic = node.querySelectorAll('div, p, span, textarea');
    for (var k = 0; k < generic.length; k += 1) {
      var text = utils.normalizeText(generic[k].textContent);
      if (!text || text === highlightText || text === metadataText) {
        continue;
      }
      if (text.length < 3) {
        continue;
      }
      if (/位置|location|page|ページ/i.test(text) && text.length <= 40) {
        continue;
      }
      if (highlightText && text.indexOf(highlightText) >= 0) {
        continue;
      }

      return {
        text: text,
        html: generic[k].outerHTML
      };
    }

    return {
      text: '',
      html: ''
    };
  }

  function extractColor(node) {
    var rawHints = [
      node.className || '',
      JSON.stringify(utils.serializeAttributes(node)),
      JSON.stringify(utils.serializeDataset(node))
    ].join(' ').toLowerCase();

    for (var i = 0; i < COLOR_NAMES.length; i += 1) {
      if (rawHints.indexOf(COLOR_NAMES[i]) !== -1) {
        return COLOR_NAMES[i];
      }
    }

    var style = window.getComputedStyle(node);
    var colorHint = (style.backgroundColor || '') + ' ' + (style.color || '');
    if (/255,\s*235,\s*59|255,\s*241,\s*118/i.test(colorHint)) {
      return 'yellow';
    }
    if (/66,\s*165,\s*245|144,\s*202,\s*249/i.test(colorHint)) {
      return 'blue';
    }
    if (/244,\s*143,\s*177|248,\s*187,\s*208/i.test(colorHint)) {
      return 'pink';
    }
    if (/255,\s*183,\s*77|255,\s*224,\s*178/i.test(colorHint)) {
      return 'orange';
    }

    return '';
  }

  function extractLocation(metadataText) {
    var patterns = [
      /位置(?:No\.?|番号)?[:：]?\s*([0-9,\-–~〜]+)/i,
      /Location[:\s#]*([0-9,\-–~]+)/i
    ];

    for (var i = 0; i < patterns.length; i += 1) {
      var match = metadataText.match(patterns[i]);
      if (match && match[1]) {
        return utils.normalizeText(match[1]);
      }
    }

    return '';
  }

  function extractPage(metadataText) {
    var patterns = [
      /ページ[:：]?\s*([0-9,\-–~〜]+)/i,
      /Page[:\s#]*([0-9,\-–~]+)/i
    ];

    for (var i = 0; i < patterns.length; i += 1) {
      var match = metadataText.match(patterns[i]);
      if (match && match[1]) {
        return utils.normalizeText(match[1]);
      }
    }

    return '';
  }

  function extractPossibleDate(metadataText) {
    var patterns = [
      /\d{4}[\/.-]\d{1,2}[\/.-]\d{1,2}(?:\s+\d{1,2}:\d{2}(?::\d{2})?)?/,
      /\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},\s+\d{4}(?:\s+\d{1,2}:\d{2}(?::\d{2})?)?/i
    ];

    for (var i = 0; i < patterns.length; i += 1) {
      var match = metadataText.match(patterns[i]);
      if (!match || !match[0]) {
        continue;
      }

      var parsed = Date.parse(match[0]);
      return {
        iso: Number.isNaN(parsed) ? null : new Date(parsed).toISOString(),
        raw: match[0]
      };
    }

    return {
      iso: null,
      raw: null
    };
  }

  function collectAnnotationIdCandidates(node) {
    var candidates = [];

    function pushCandidate(value) {
      var normalized = utils.normalizeText(value);
      if (!normalized) {
        return;
      }
      if (candidates.indexOf(normalized) !== -1) {
        return;
      }
      candidates.push(normalized);
    }

    pushCandidate(node.id);

    var attributes = utils.serializeAttributes(node);
    Object.keys(attributes).forEach(function (key) {
      if (/annotation|highlight|note|id/i.test(key)) {
        pushCandidate(attributes[key]);
      }
    });

    var dataset = utils.serializeDataset(node);
    Object.keys(dataset).forEach(function (key) {
      if (/annotation|highlight|note|id/i.test(key)) {
        pushCandidate(dataset[key]);
      }
    });

    var nested = node.querySelectorAll('[id], [data-annotation-id], [data-highlight-id], [data-note-id]');
    for (var i = 0; i < nested.length; i += 1) {
      pushCandidate(nested[i].id);
      var nestedAttrs = utils.serializeAttributes(nested[i]);
      Object.keys(nestedAttrs).forEach(function (key) {
        if (/annotation|highlight|note|id/i.test(key)) {
          pushCandidate(nestedAttrs[key]);
        }
      });
    }

    return candidates;
  }

  function collectLinks(node) {
    var links = [];
    var anchors = node.querySelectorAll('a[href]');
    for (var i = 0; i < anchors.length; i += 1) {
      var href = utils.normalizeText(anchors[i].getAttribute('href'));
      if (href && links.indexOf(href) === -1) {
        links.push(href);
      }
    }
    return links;
  }

  function collectImages(node) {
    var images = [];
    var imageNodes = node.querySelectorAll('img[src]');
    for (var i = 0; i < imageNodes.length; i += 1) {
      var src = utils.normalizeText(imageNodes[i].getAttribute('src'));
      if (src && images.indexOf(src) === -1) {
        images.push(src);
      }
    }
    return images;
  }

  function splitMetadataTokens(metadataText) {
    if (!metadataText) {
      return [];
    }

    return metadataText
      .split(/[|｜•●]/)
      .map(utils.normalizeText)
      .filter(Boolean);
  }

  function scrapeHighlight(node, bookInfo, index) {
    var highlightText = getHighlightText(node);
    var metadataText = extractMetadataText(node);
    var noteInfo = extractNoteInfo(node, highlightText, metadataText);
    var dateInfo = extractPossibleDate(metadataText);
    var annotationIdCandidates = collectAnnotationIdCandidates(node);

    return {
      amazonAnnotationId: annotationIdCandidates[0] || null,
      amazonAnnotationIdCandidates: annotationIdCandidates,
      highlightText: highlightText || null,
      note: noteInfo.text || null,
      noteHtml: noteInfo.html || null,
      color: extractColor(node) || null,
      location: extractLocation(metadataText) || null,
      page: extractPage(metadataText) || null,
      highlightedAt: dateInfo.iso,
      highlightedAtRaw: dateInfo.raw,
      metadataText: metadataText || null,
      metadataTokens: splitMetadataTokens(metadataText),
      domId: node.id || null,
      classNames: Array.from(node.classList || []),
      rawAttributes: utils.serializeAttributes(node),
      rawDataset: utils.serializeDataset(node),
      rawText: utils.normalizeText(node.textContent),
      rawHtml: node.outerHTML,
      links: collectLinks(node),
      images: collectImages(node),
      sortOrder: index + 1,
      bookTitle: bookInfo.title || null,
      bookAuthor: bookInfo.author || null,
      bookAsin: bookInfo.asin || null
    };
  }

  async function openBookEntry(entry) {
    if (isCurrentBookMatch(entry.asin, entry.title, entry.author)) {
      await utils.sleep(300);
      return;
    }

    var liveElement = await findLiveBookElement(entry);
    if (!liveElement) {
      throw new Error('BOOK_NOT_FOUND');
    }

    if (!openBookElement(liveElement)) {
      throw new Error('BOOK_OPEN_FAILED');
    }

    await utils.waitForCondition(function () {
      return isCurrentBookMatch(entry.asin, entry.title, entry.author);
    }, 15000, 350);

    await utils.sleep(500);
  }

  async function scrapeBook(entry) {
    await openBookEntry(entry);
    await loadAllHighlightsForCurrentBook();

    var bookInfo = getCurrentBookHeaderInfo(entry);
    var nodes = getHighlightNodes();
    var highlights = [];

    for (var i = 0; i < nodes.length; i += 1) {
      var highlight = scrapeHighlight(nodes[i], bookInfo, i);
      if (!utils.hasMeaningfulValue(highlight.highlightText) && !utils.hasMeaningfulValue(highlight.note)) {
        continue;
      }
      highlights.push(highlight);
    }

    return {
      asin: bookInfo.asin || entry.asin || null,
      title: bookInfo.title || entry.title || null,
      author: bookInfo.author || entry.author || null,
      coverImageUrl: bookInfo.coverImageUrl || entry.coverImageUrl || null,
      notebookUrl: entry.notebookUrl || buildNotebookURLFromASIN(bookInfo.asin || entry.asin) || null,
      sidebarText: entry.sidebarText || null,
      sidebarHtml: entry.sidebarHtml || null,
      sidebarAttributes: entry.sidebarAttributes || {},
      sidebarDataset: entry.sidebarDataset || {},
      headerText: bookInfo.headerText || null,
      headerHtml: bookInfo.headerHtml || null,
      headerAttributes: bookInfo.headerAttributes || {},
      headerDataset: bookInfo.headerDataset || {},
      highlights: highlights
    };
  }

  function formatColorDistribution(colorDistribution) {
    return Object.keys(colorDistribution || {})
      .map(function (key) {
        return key + '=' + colorDistribution[key];
      })
      .join(', ');
  }

  function logScrapeOutputs(result) {
    var books = result.books || [];
    var highlights = [];
    for (var i = 0; i < books.length; i += 1) {
      highlights = highlights.concat(books[i].highlights || []);
    }

    console.log('=== Sample Highlight (Full Properties) ===');
    console.log(highlights[0] || null);

    console.log('=== Summary ===');
    console.log('Total books: ' + result.summary.totalBooks);
    console.log('Total highlights: ' + result.summary.totalHighlights);
    console.log('Highlights with notes: ' + result.summary.highlightsWithNotes);
    console.log('Color distribution: ' + formatColorDistribution(result.colorDistribution));

    console.log('=== Per-book counts ===');
    for (var j = 0; j < result.perBookCounts.length; j += 1) {
      var item = result.perBookCounts[j];
      console.log('"' + item.title + '" (' + (item.asin || 'NO-ASIN') + '): ' + item.count + ' highlights');
    }

    console.log('=== Detected Fields ===');
    var detectedKeys = Object.keys(result.detectedFields || {}).sort();
    for (var k = 0; k < detectedKeys.length; k += 1) {
      var key = detectedKeys[k];
      var stat = result.detectedFields[key];
      console.log(
        key + ': ' + stat.count + '/' + stat.total + ' (' + (stat.ratio * 100).toFixed(1) + '%)'
      );
    }

    if (Array.isArray(result.errors) && result.errors.length > 0) {
      console.log('=== Errors ===');
      console.log(result.errors);
    }
  }

  function buildResult(books, errors) {
    var summaryBundle = utils.buildSummary({ books: books });
    return {
      syncedAt: new Date().toISOString(),
      amazonDomain: window.location.hostname,
      summary: {
        totalBooks: summaryBundle.totalBooks,
        totalHighlights: summaryBundle.totalHighlights,
        highlightsWithNotes: summaryBundle.highlightsWithNotes
      },
      colorDistribution: summaryBundle.colorDistribution,
      perBookCounts: summaryBundle.perBookCounts,
      detectedFields: summaryBundle.detectedFields,
      errors: errors,
      books: books
    };
  }

  function createSerializableBookEntry(entry) {
    return {
      id: entry.id,
      asin: entry.asin,
      title: entry.title,
      author: entry.author,
      coverImageUrl: entry.coverImageUrl,
      notebookUrl: entry.notebookUrl,
      sidebarText: entry.sidebarText,
      sidebarHtml: entry.sidebarHtml,
      sidebarAttributes: entry.sidebarAttributes,
      sidebarDataset: entry.sidebarDataset
    };
  }

  async function continueSession(session) {
    var bookEntries = Array.isArray(session.books) ? session.books : [];
    var books = Array.isArray(session.results) ? session.results : [];
    var errors = Array.isArray(session.errors) ? session.errors : [];

    for (var i = session.currentIndex || 0; i < bookEntries.length; i += 1) {
      var entry = bookEntries[i];
      session.currentIndex = i;
      session.results = books;
      session.errors = errors;
      writeSession(session);

      setState({
        status: 'running',
        message: 'ハイライトを収集中です...',
        totalBooks: bookEntries.length,
        currentBookIndex: i + 1,
        currentBookTitle: entry.title || entry.asin || 'タイトル不明',
        errors: errors
      });

      try {
        var book = await scrapeBook(entry);
        books.push(book);
      } catch (error) {
        console.error('[Kindle Highlights Scraper] Failed to scrape book', entry, error);
        errors.push({
          asin: entry.asin || null,
          title: entry.title || null,
          error: error && error.message ? error.message : String(error)
        });
      }

      session.currentIndex = i + 1;
      session.results = books;
      session.errors = errors;
      writeSession(session);

      await utils.sleep(utils.randomBetween(300, 500));
    }

    var result = buildResult(books, errors);
    lastResult = result;
    session.active = false;
    session.result = result;
    writeSession(session);

    logScrapeOutputs(result);

    setState({
      status: 'done',
      message: errors.length > 0
        ? '同期完了。一部の書籍で失敗がありますが、JSON はダウンロードできます。'
        : '同期完了。JSON をダウンロードできます。',
      totalBooks: bookEntries.length,
      currentBookIndex: bookEntries.length,
      currentBookTitle: '',
      completedAt: result.syncedAt,
      summary: Object.assign({}, result.summary, {
        colorDistribution: result.colorDistribution
      }),
      errors: errors,
      downloadReady: true
    });
  }

  async function runScrape() {
    if (isLoginPage()) {
      throw new Error('Amazon にログインした状態で Kindle Notebook を開いてください');
    }
    if (!isNotebookPage()) {
      throw new Error('Kindle Notebook ページで実行してください');
    }

    resetState();
    setState({
      status: 'running',
      message: '書籍一覧を読み取っています...',
      startedAt: new Date().toISOString(),
      amazonDomain: window.location.hostname,
      downloadReady: false
    });

    var bookEntries = await waitForBookEntries();
    if (!bookEntries.length) {
      throw new Error('書籍一覧を取得できませんでした');
    }

    var books = [];
    var session = {
      active: true,
      startedAt: currentState.startedAt,
      amazonDomain: window.location.hostname,
      currentIndex: 0,
      books: bookEntries.map(createSerializableBookEntry),
      results: books,
      errors: []
    };
    writeSession(session);

    setState({
      status: 'running',
      message: '全書籍の巡回を開始します...',
      totalBooks: bookEntries.length,
      currentBookIndex: 0,
      currentBookTitle: '',
      errors: []
    });

    await continueSession(session);
  }

  function handleStartSync(sendResponse) {
    if (isRunning) {
      sendResponse({
        ok: false,
        error: 'すでに同期中です'
      });
      return;
    }

    isRunning = true;
    lastResult = null;
    runScrape()
      .catch(function (error) {
        console.error('[Kindle Highlights Scraper] Fatal scrape error', error);
        setState({
          status: 'error',
          message: error && error.message ? error.message : '同期に失敗しました',
          completedAt: new Date().toISOString(),
          downloadReady: false
        });
      })
      .finally(function () {
        isRunning = false;
      });

    sendResponse({ ok: true });
  }

  chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
    if (!message || message.namespace !== shared.NAMESPACE) {
      return undefined;
    }

    if (message.type === messageTypes.START_SYNC) {
      handleStartSync(sendResponse);
      return true;
    }

    if (message.type === messageTypes.GET_STATE) {
      sendResponse({ ok: true, state: currentState });
      return undefined;
    }

    if (message.type === messageTypes.GET_RESULT) {
      if (!lastResult) {
        var session = readSession();
        if (session && session.result) {
          lastResult = session.result;
        }
      }
      sendResponse({ ok: true, result: lastResult });
      return undefined;
    }

    return undefined;
  });

  initializeStoredState();

  var resumableSession = readSession();
  if (resumableSession && resumableSession.active && !isRunning && isNotebookPage()) {
    isRunning = true;
    setState({
      status: 'running',
      message: '前回の同期を再開しています...',
      totalBooks: Array.isArray(resumableSession.books) ? resumableSession.books.length : 0,
      currentBookIndex: resumableSession.currentIndex || 0,
      currentBookTitle: '',
      startedAt: resumableSession.startedAt || new Date().toISOString(),
      amazonDomain: resumableSession.amazonDomain || window.location.hostname,
      downloadReady: false
    });

    continueSession(resumableSession)
      .catch(function (error) {
        console.error('[Kindle Highlights Scraper] Resume failed', error);
        setState({
          status: 'error',
          message: error && error.message ? error.message : '同期の再開に失敗しました',
          completedAt: new Date().toISOString(),
          downloadReady: false
        });
      })
      .finally(function () {
        isRunning = false;
      });
  } else if (resumableSession && resumableSession.result && !lastResult) {
    lastResult = resumableSession.result;
  }
})(globalThis);
