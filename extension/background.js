var pendingSyncs = {};
var BRIDGE_STORAGE_KEY = 'aiStudyToolBridgeMessage';
var WEBAPP_PORT_NAME = 'ai-study-tool-webapp';
var webappPorts = {};
var NOTEBOOK_TAB_TIMEOUT_MS = 45000;

console.log('[kindle-sync][background] service worker loaded');

chrome.runtime.onConnect.addListener(function (port) {
  if (!port || port.name !== WEBAPP_PORT_NAME) return;

  var tabId = port.sender && port.sender.tab && typeof port.sender.tab.id === 'number'
    ? port.sender.tab.id
    : null;
  if (tabId === null) return;

  console.log('[kindle-sync][background] port connected', { tabId: tabId });
  webappPorts[tabId] = port;

  port.onDisconnect.addListener(function () {
    console.log('[kindle-sync][background] port disconnected', { tabId: tabId });
    if (webappPorts[tabId] === port) {
      delete webappPorts[tabId];
    }
  });

  port.onMessage.addListener(function (message) {
    if (!message || !message.type) return;
    console.log('[kindle-sync][background] port message', {
      tabId: tabId,
      type: message.type,
      requestId: message.requestId || '',
    });

    if (message.type === 'LIST_BOOKS_REQUEST') {
      handleListBooksRequest(message, { tab: { id: tabId } });
      return;
    }

    if (message.type === 'SYNC_BOOK_REQUEST') {
      handleSyncBookRequest(message, { tab: { id: tabId } });
    }
  });
});

function getBridgeStorageKey(message) {
  if (!message || !message.requestId) return BRIDGE_STORAGE_KEY;
  return BRIDGE_STORAGE_KEY + ':' + String(message.requestId);
}

chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
  if (!message || !message.type) return;
  console.log('[kindle-sync][background] runtime message', {
    type: message.type,
    requestId: message.requestId || '',
    senderTabId: sender && sender.tab ? sender.tab.id : null,
  });

  if (message.type === 'LIST_BOOKS_REQUEST') {
    handleListBooksRequest(message, sender);
    if (typeof sendResponse === 'function') {
      sendResponse({ ok: true, stage: 'background_received' });
    }
    return;
  }
  if (message.type === 'SYNC_BOOK_REQUEST') {
    handleSyncBookRequest(message, sender);
    if (typeof sendResponse === 'function') {
      sendResponse({ ok: true, stage: 'background_received' });
    }
    return;
  }
  if (message.type === 'NOTEBOOK_BOOK_LIST') {
    handleNotebookBookList(message, sender);
    return;
  }
  if (message.type === 'NOTEBOOK_PROGRESS') {
    handleNotebookProgress(message, sender);
    return;
  }
  if (message.type === 'NOTEBOOK_HIGHLIGHT_DATA') {
    handleNotebookHighlightData(message, sender);
    return;
  }
  if (message.type === 'NOTEBOOK_ERROR') {
    handleNotebookError(message, sender);
  }
});

function handleListBooksRequest(message, sender) {
  var webappTabId = sender.tab && typeof sender.tab.id === 'number' ? sender.tab.id : null;
  if (webappTabId === null) return;

  console.log('[kindle-sync][background] handle list books request', {
    requestId: message.requestId || '',
    webappTabId: webappTabId,
  });

  sendToWebapp(webappTabId, {
    type: 'LIST_BOOKS_PROGRESS',
    requestId: message.requestId,
    stage: 'background_received',
  });

  chrome.tabs.create({ url: 'https://read.amazon.co.jp/notebook#mode=list', active: false }, function (tab) {
    console.log('[kindle-sync][background] tabs.create callback', {
      requestId: message.requestId || '',
      hasError: Boolean(chrome.runtime.lastError),
      error: chrome.runtime.lastError ? chrome.runtime.lastError.message : '',
      createdTabId: tab && typeof tab.id === 'number' ? tab.id : null,
    });
    if (chrome.runtime.lastError || !tab || typeof tab.id !== 'number') {
      sendToWebapp(webappTabId, {
        type: 'LIST_BOOKS_RESULT',
        requestId: message.requestId,
        books: [],
        error: 'TAB_CREATE_FAILED',
      });
      return;
    }
    pendingSyncs[tab.id] = {
      mode: 'list',
      requestId: message.requestId,
      webappTabId: webappTabId,
      timeoutId: setTimeout(function () {
        handleNotebookTabTimeout(tab.id);
      }, NOTEBOOK_TAB_TIMEOUT_MS),
    };
    sendToWebapp(webappTabId, {
      type: 'LIST_BOOKS_PROGRESS',
      requestId: message.requestId,
      stage: 'tab_opened',
    });
  });
}

function handleSyncBookRequest(message, sender) {
  var webappTabId = sender.tab && typeof sender.tab.id === 'number' ? sender.tab.id : null;
  if (webappTabId === null || !message.bookId || !message.token || !message.appOrigin) return;

  sendToWebapp(webappTabId, {
    type: 'SYNC_BOOK_PROGRESS',
    requestId: message.requestId,
    bookId: message.bookId,
    stage: 'background_received',
  });

  if (!message.asin) {
    sendToWebapp(webappTabId, {
      type: 'SYNC_BOOK_RESULT',
      requestId: message.requestId,
      bookId: message.bookId,
      result: null,
      error: 'BOOK_IDENTIFIER_MISSING',
    });
    return;
  }

  var url = buildSyncURL(message);
  if (!url) {
    sendToWebapp(webappTabId, {
      type: 'SYNC_BOOK_RESULT',
      requestId: message.requestId,
      bookId: message.bookId,
      result: null,
      error: 'BOOK_TARGET_UNAVAILABLE',
    });
    return;
  }

  console.log('[kindle-sync][background] open sync tab', {
    requestId: message.requestId || '',
    bookId: message.bookId,
    asin: message.asin || '',
    notebookUrl: message.notebookUrl || '',
    resolvedUrl: url,
  });

  chrome.tabs.create({ url: url, active: false }, function (tab) {
    if (chrome.runtime.lastError || !tab || typeof tab.id !== 'number') {
      var createErrorMessage = chrome.runtime.lastError ? chrome.runtime.lastError.message : '';
      sendToWebapp(webappTabId, {
        type: 'SYNC_BOOK_RESULT',
        requestId: message.requestId,
        bookId: message.bookId,
        result: null,
        error: createErrorMessage ? 'TAB_CREATE_FAILED: ' + createErrorMessage : 'TAB_CREATE_FAILED',
      });
      return;
    }

    sendToWebapp(webappTabId, {
      type: 'SYNC_BOOK_PROGRESS',
      requestId: message.requestId,
      bookId: message.bookId,
      stage: 'tab_opened',
    });

    pendingSyncs[tab.id] = {
      mode: 'sync',
      requestId: message.requestId,
      bookId: message.bookId,
      asin: message.asin,
      token: message.token,
      appOrigin: message.appOrigin,
      webappTabId: webappTabId,
      timeoutId: setTimeout(function () {
        handleNotebookTabTimeout(tab.id);
      }, NOTEBOOK_TAB_TIMEOUT_MS),
    };
  });
}

function clearPendingSync(tabId) {
  if (tabId === null || !pendingSyncs[tabId]) return null;

  var sync = pendingSyncs[tabId];
  delete pendingSyncs[tabId];
  if (sync.timeoutId) {
    clearTimeout(sync.timeoutId);
  }
  return sync;
}

function handleNotebookTabTimeout(tabId) {
  var sync = clearPendingSync(tabId);
  if (!sync) return;

  closeTab(tabId);

  if (sync.mode === 'list') {
    sendToWebapp(sync.webappTabId, {
      type: 'LIST_BOOKS_RESULT',
      requestId: sync.requestId,
      books: [],
      error: 'NOTEBOOK_TIMEOUT',
    });
    return;
  }

  sendToWebapp(sync.webappTabId, {
    type: 'SYNC_BOOK_RESULT',
    requestId: sync.requestId,
    bookId: sync.bookId,
    result: null,
    error: 'NOTEBOOK_TIMEOUT',
  });
}

function handleNotebookBookList(message, sender) {
  var tabId = sender.tab && typeof sender.tab.id === 'number' ? sender.tab.id : null;
  if (tabId === null || !pendingSyncs[tabId] || pendingSyncs[tabId].mode !== 'list') return;

  var sync = clearPendingSync(tabId);
  closeTab(tabId);

  sendToWebapp(sync.webappTabId, {
    type: 'LIST_BOOKS_RESULT',
    requestId: sync.requestId,
    books: Array.isArray(message.books) ? message.books : [],
    error: null,
  });
}

function handleNotebookProgress(message, sender) {
  var tabId = sender.tab && typeof sender.tab.id === 'number' ? sender.tab.id : null;
  if (tabId === null || !pendingSyncs[tabId]) return;

  var sync = pendingSyncs[tabId];
  if (sync.mode === 'list') {
    sendToWebapp(sync.webappTabId, {
      type: 'LIST_BOOKS_PROGRESS',
      requestId: sync.requestId,
      stage: message.stage || 'unknown',
      count: typeof message.count === 'number' ? message.count : undefined,
    });
    return;
  }

  if (sync.mode === 'sync') {
    sendToWebapp(sync.webappTabId, {
      type: 'SYNC_BOOK_PROGRESS',
      requestId: sync.requestId,
      bookId: sync.bookId,
      stage: message.stage || 'unknown',
      count: typeof message.count === 'number' ? message.count : undefined,
    });
  }
}

function handleNotebookHighlightData(message, sender) {
  var tabId = sender.tab && typeof sender.tab.id === 'number' ? sender.tab.id : null;
  if (tabId === null || !pendingSyncs[tabId] || pendingSyncs[tabId].mode !== 'sync') return;

  var sync = pendingSyncs[tabId];
  sendToWebapp(sync.webappTabId, {
    type: 'SYNC_BOOK_PROGRESS',
    requestId: sync.requestId,
    bookId: sync.bookId,
    stage: 'highlight_data_received',
  });
  var token = sync.token;
  sync = clearPendingSync(tabId);
  closeTab(tabId);

  var highlights = Array.isArray(message.highlights) ? message.highlights : [];
  if (highlights.length === 0) {
    sendToWebapp(sync.webappTabId, {
      type: 'SYNC_BOOK_RESULT',
      requestId: sync.requestId,
      bookId: sync.bookId,
      result: null,
      error: 'NO_HIGHLIGHTS',
    });
    return;
  }

  postImport(sync.appOrigin, token, highlights)
    .then(function (result) {
      sendToWebapp(sync.webappTabId, {
        type: 'SYNC_BOOK_RESULT',
        requestId: sync.requestId,
        bookId: sync.bookId,
        result: result,
        error: null,
      });
    })
    .catch(function (err) {
      sendToWebapp(sync.webappTabId, {
        type: 'SYNC_BOOK_RESULT',
        requestId: sync.requestId,
        bookId: sync.bookId,
        result: null,
        error: typeof err === 'string' ? err : 'IMPORT_FAILED',
      });
    });
}

function handleNotebookError(message, sender) {
  var tabId = sender.tab && typeof sender.tab.id === 'number' ? sender.tab.id : null;
  if (tabId === null || !pendingSyncs[tabId]) return;

  var sync = clearPendingSync(tabId);
  closeTab(tabId);

  if (sync.mode === 'list') {
    sendToWebapp(sync.webappTabId, {
      type: 'LIST_BOOKS_RESULT',
      requestId: sync.requestId,
      books: [],
      error: message.error || 'NOTEBOOK_ERROR',
    });
  } else {
    sendToWebapp(sync.webappTabId, {
      type: 'SYNC_BOOK_RESULT',
      requestId: sync.requestId,
      bookId: sync.bookId,
      result: null,
      error: message.error || 'NOTEBOOK_ERROR',
    });
  }
}

function sendToWebapp(webappTabId, message) {
  if (typeof webappTabId !== 'number') return;
  console.log('[kindle-sync][background] send to webapp', {
    webappTabId: webappTabId,
    type: message.type,
    requestId: message.requestId || '',
    stage: message.stage || '',
    error: message.error || '',
  });

  var port = webappPorts[webappTabId];
  if (port) {
    try {
      port.postMessage(message);
    } catch (_) {
      delete webappPorts[webappTabId];
    }
  }

  chrome.tabs.sendMessage(webappTabId, message, function () {
    void chrome.runtime.lastError;
  });

  if (!chrome.scripting || typeof chrome.scripting.executeScript !== 'function') {
    return;
  }

  chrome.scripting.executeScript(
    {
      target: { tabId: webappTabId },
      func: function (payload) {
        try {
          window.postMessage(payload, window.location.origin);
        } catch (_) {}

        try {
          document.dispatchEvent(
            new CustomEvent('ai-study-tool:kindle-response', {
              detail: JSON.stringify(payload),
            })
          );
        } catch (_) {}
      },
      args: [message],
    },
    function () {
      void chrome.runtime.lastError;
    }
  );

  if (chrome.storage && chrome.storage.local && typeof chrome.storage.local.set === 'function') {
    chrome.storage.local.set({
      [getBridgeStorageKey(message)]: {
        bridgeId: String(Date.now()) + ':' + String(Math.random()),
        targetTabId: webappTabId,
        payload: message,
      },
    }, function () {
      void chrome.runtime.lastError;
    });
  }
}

function closeTab(tabId) {
  chrome.tabs.remove(tabId, function () {
    void chrome.runtime.lastError;
  });
}

function buildImportURL(appOrigin) {
  try {
    return new URL('/api/highlights/import', appOrigin).toString();
  } catch (_) {
    return '';
  }
}

function normalizeAmazonNotebookURL(rawURL) {
  if (!rawURL) return '';

  var value = String(rawURL).trim();
  if (!value) return '';

  try {
    var parsed = new URL(value);
    var protocol = (parsed.protocol || '').toLowerCase();
    var hostname = (parsed.hostname || '').toLowerCase();
    if (protocol !== 'http:' && protocol !== 'https:') {
      return '';
    }
    if (
      hostname !== 'read.amazon.co.jp' &&
      hostname !== 'www.amazon.co.jp' &&
      hostname !== 'amazon.co.jp'
    ) {
      return '';
    }
    return parsed.toString();
  } catch (_) {}

  if (value.indexOf('//') === 0) {
    return 'https:' + value;
  }

  if (value.charAt(0) === '#') {
    return 'https://read.amazon.co.jp/notebook' + value;
  }

  if (value.charAt(0) === '/') {
    return 'https://read.amazon.co.jp' + value;
  }

  if (value.indexOf('read.amazon.co.jp') === 0 || value.indexOf('www.amazon.co.jp') === 0 || value.indexOf('amazon.co.jp') === 0) {
    return 'https://' + value;
  }

  if (value.indexOf('notebook') !== -1) {
    return 'https://read.amazon.co.jp/' + value.replace(/^\/+/, '');
  }

  return '';
}

function appendHashParams(baseURL, params) {
  try {
    var normalizedBaseURL = normalizeAmazonNotebookURL(baseURL);
    if (!normalizedBaseURL) return '';

    var url = new URL(normalizedBaseURL);
    var hash = url.hash ? url.hash.replace(/^#/, '') : '';
    var search = new URLSearchParams(hash);
    Object.keys(params).forEach(function (key) {
      if (params[key]) search.set(key, params[key]);
    });
    url.hash = search.toString();
    return url.toString();
  } catch (_) {
    return '';
  }
}

function buildSyncURL(message) {
  var asin = message && message.asin ? String(message.asin).trim() : '';
  var bookTitle = message && message.bookTitle ? String(message.bookTitle).trim() : '';
  var bookAuthor = message && message.bookAuthor ? String(message.bookAuthor).trim() : '';

  if (!asin) return '';

  return 'https://read.amazon.co.jp/notebook?asin='
    + encodeURIComponent(asin)
    + '#mode=sync&asin='
    + encodeURIComponent(asin)
    + '&book_title='
    + encodeURIComponent(bookTitle)
    + '&book_author='
    + encodeURIComponent(bookAuthor);
}

function postImport(appOrigin, token, highlights) {
  var importURL = buildImportURL(appOrigin);
  if (!importURL) {
    return Promise.reject('IMPORT_URL_INVALID');
  }

  return fetch(importURL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + token,
    },
    body: JSON.stringify({ highlights: highlights }),
  }).then(function (res) {
    return res.text().then(function (text) {
      var body = {};
      if (text) {
        try { body = JSON.parse(text); } catch (_) {}
      }
      if (res.ok) return body;
      if (res.status === 422) throw (body && body.message ? body.message : 'ALL_COPY_PROTECTED');
      throw 'IMPORT_FAILED';
    });
  }).catch(function (err) {
    if (typeof err === 'string') throw err;
    throw 'NETWORK_ERROR';
  });
}
