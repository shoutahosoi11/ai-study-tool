document.documentElement.setAttribute('data-kindle-sync-extension', 'installed');

var APP_ORIGIN = window.location.origin;
var REQUEST_EVENT = 'ai-study-tool:kindle-request';
var RESPONSE_EVENT = 'ai-study-tool:kindle-response';
var BRIDGE_STORAGE_KEY = 'aiStudyToolBridgeMessage';
var WEBAPP_PORT_NAME = 'ai-study-tool-webapp';
var bridgePollers = {};
var runtimePort = null;
var pendingPortRequests = {};
var recentlyHandledRequests = {};

function emitToPage(message) {
  window.postMessage(message, APP_ORIGIN);
  document.dispatchEvent(
    new CustomEvent(RESPONSE_EVENT, {
      detail: JSON.stringify(message),
    })
  );
}

function parseMessageDetail(detail) {
  if (!detail) return null;
  if (typeof detail === 'string') {
    try {
      return JSON.parse(detail);
    } catch (_) {
      return null;
    }
  }
  return detail;
}

function getBridgeStorageKey(requestId) {
  return BRIDGE_STORAGE_KEY + ':' + String(requestId);
}

function stopBridgePolling(requestId) {
  var poller = bridgePollers[requestId];
  if (!poller) return;
  window.clearInterval(poller.timer);
  delete bridgePollers[requestId];
}

function clearPendingPortRequest(requestId) {
  var pending = pendingPortRequests[requestId];
  if (!pending) return;
  window.clearTimeout(pending.timer);
  delete pendingPortRequests[requestId];
}

function failPendingPortRequest(requestId, error) {
  var pending = pendingPortRequests[requestId];
  if (!pending) return;

  clearPendingPortRequest(requestId);

  if (pending.kind === 'list') {
    emitToPage({
      type: 'LIST_BOOKS_RESULT',
      requestId: requestId,
      books: [],
      error: error,
    });
    return;
  }

  emitToPage({
    type: 'SYNC_BOOK_RESULT',
    requestId: requestId,
    bookId: pending.bookId,
    result: null,
    error: error,
  });
}

function shouldHandleRequest(message) {
  if (!message || !message.type || !message.requestId) return true;

  var key = message.type + ':' + String(message.requestId);
  var now = Date.now();
  var lastHandledAt = recentlyHandledRequests[key] || 0;
  recentlyHandledRequests[key] = now;

  return now-lastHandledAt > 1000;
}

function trackPendingPortRequest(requestId, kind, bookId) {
  clearPendingPortRequest(requestId);
  pendingPortRequests[requestId] = {
    kind: kind,
    bookId: bookId || '',
    timer: window.setTimeout(function () {
      failPendingPortRequest(requestId, 'BACKGROUND_NO_ACK');
    }, 3000),
  };
}

function startBridgePolling(requestId) {
  if (!requestId || !chrome.storage || !chrome.storage.local) return;
  stopBridgePolling(requestId);

  bridgePollers[requestId] = {
    lastBridgeId: '',
    timer: window.setInterval(function () {
      chrome.storage.local.get(getBridgeStorageKey(requestId), function (result) {
        var entry = result && result[getBridgeStorageKey(requestId)];
        if (!entry || !entry.payload || !entry.bridgeId) return;

        var poller = bridgePollers[requestId];
        if (!poller || poller.lastBridgeId === entry.bridgeId) return;
        poller.lastBridgeId = entry.bridgeId;

        emitToPage(entry.payload);

        if (entry.payload.type === 'LIST_BOOKS_RESULT' || entry.payload.type === 'SYNC_BOOK_RESULT') {
          stopBridgePolling(requestId);
        }
      });
    }, 500),
  };
}

function sendRuntimeMessage(message, onSuccess, onError) {
  if (!chrome || !chrome.runtime || typeof chrome.runtime.sendMessage !== 'function') {
    if (typeof onError === 'function') onError('RUNTIME_UNAVAILABLE');
    return;
  }

  var settled = false;
  var timer = window.setTimeout(function () {
    if (settled) return;
    settled = true;
    if (typeof onError === 'function') onError('BACKGROUND_NO_ACK');
  }, 3000);

  try {
    chrome.runtime.sendMessage(message, function (response) {
      if (settled) return;
      settled = true;
      window.clearTimeout(timer);

      if (chrome.runtime.lastError && typeof onError === 'function') {
        onError(chrome.runtime.lastError.message || 'BACKGROUND_UNREACHABLE');
        return;
      }

      if (typeof onSuccess === 'function') {
        onSuccess(response);
      }
    });
  } catch (_) {
    if (settled) return;
    settled = true;
    window.clearTimeout(timer);
    if (typeof onError === 'function') onError('BACKGROUND_UNREACHABLE');
  }
}

function ensureRuntimePort() {
  if (runtimePort) return runtimePort;
  if (!chrome || !chrome.runtime || typeof chrome.runtime.connect !== 'function') return null;

  try {
    runtimePort = chrome.runtime.connect({ name: WEBAPP_PORT_NAME });
  } catch (_) {
    runtimePort = null;
    return null;
  }

  runtimePort.onMessage.addListener(function (message) {
    if (!message || !message.type) return;
    if (message.requestId) {
      clearPendingPortRequest(message.requestId);
    }
    emitToPage(message);
  });

  runtimePort.onDisconnect.addListener(function () {
    Object.keys(pendingPortRequests).forEach(function (requestId) {
      failPendingPortRequest(requestId, 'BACKGROUND_UNREACHABLE');
    });
    runtimePort = null;
  });

  return runtimePort;
}

function handlePageMessage(message) {
  if (!message || !message.type) return;
  if (!shouldHandleRequest(message)) return;

  var type = message.type;

  if (type === 'LIST_BOOKS_REQUEST') {
    startBridgePolling(message.requestId);
    emitToPage({
      type: 'LIST_BOOKS_PROGRESS',
      requestId: message.requestId,
      stage: 'bridge_request_sent',
    });

    var port = ensureRuntimePort();
    if (port) {
      try {
        trackPendingPortRequest(message.requestId, 'list');
        port.postMessage({
          type: 'LIST_BOOKS_REQUEST',
          requestId: message.requestId,
        });
        return;
      } catch (_) {
        runtimePort = null;
      }
    }

    sendRuntimeMessage(
      {
        type: 'LIST_BOOKS_REQUEST',
        requestId: message.requestId,
      },
      function (response) {
        emitToPage({
          type: 'LIST_BOOKS_PROGRESS',
          requestId: message.requestId,
          stage: response && response.stage ? response.stage : 'background_received',
        });
      },
      function (error) {
        emitToPage({
          type: 'LIST_BOOKS_RESULT',
          requestId: message.requestId,
          books: [],
          error: error,
        });
      }
    );
    return;
  }

  if (type === 'SYNC_BOOK_REQUEST') {
    startBridgePolling(message.requestId);

    var syncPort = ensureRuntimePort();
    if (syncPort) {
      try {
        trackPendingPortRequest(message.requestId, 'sync', message.bookId);
        syncPort.postMessage({
          type: 'SYNC_BOOK_REQUEST',
          requestId: message.requestId,
          bookId: message.bookId,
          asin: message.asin,
          bookTitle: message.bookTitle,
          bookAuthor: message.bookAuthor,
          notebookUrl: message.notebookUrl,
          token: message.token,
          appOrigin: message.appOrigin,
        });
        return;
      } catch (_) {
        runtimePort = null;
      }
    }

    sendRuntimeMessage(
      {
        type: 'SYNC_BOOK_REQUEST',
        requestId: message.requestId,
        bookId: message.bookId,
        asin: message.asin,
        bookTitle: message.bookTitle,
        bookAuthor: message.bookAuthor,
        notebookUrl: message.notebookUrl,
        token: message.token,
        appOrigin: message.appOrigin,
      },
      function () {},
      function (error) {
        emitToPage({
          type: 'SYNC_BOOK_RESULT',
          requestId: message.requestId,
          bookId: message.bookId,
          result: null,
          error: error,
        });
      }
    );
  }
}

window.addEventListener('message', function (event) {
  if (event.origin !== APP_ORIGIN) return;
  handlePageMessage(event.data);
});

document.addEventListener(REQUEST_EVENT, function (event) {
  handlePageMessage(parseMessageDetail(event.detail));
});

chrome.runtime.onMessage.addListener(function (message) {
  if (!message || !message.type) return;
  if (
    message.type === 'LIST_BOOKS_RESULT' ||
    message.type === 'LIST_BOOKS_PROGRESS' ||
    message.type === 'SYNC_BOOK_PROGRESS' ||
    message.type === 'SYNC_BOOK_RESULT'
  ) {
    emitToPage(message);
  }
});

if (chrome.storage && chrome.storage.onChanged) {
  chrome.storage.onChanged.addListener(function (changes, areaName) {
    if (areaName !== 'local') return;
    Object.keys(changes).forEach(function (key) {
      if (key !== BRIDGE_STORAGE_KEY && key.indexOf(BRIDGE_STORAGE_KEY + ':') !== 0) return;
      var change = changes[key];
      if (!change || !change.newValue || !change.newValue.payload) return;
      if (change.newValue.payload.requestId) {
        clearPendingPortRequest(change.newValue.payload.requestId);
      }
      emitToPage(change.newValue.payload);
    });
  });
}
