(function (global) {
  var shared = global.KindleHighlightsMvp;
  if (!shared || !shared.utils) {
    return;
  }

  var utils = shared.utils;
  var stateKey = shared.STATE_KEY;
  var messageTypes = shared.MESSAGE_TYPES;

  var statusText = document.getElementById('statusText');
  var progressText = document.getElementById('progressText');
  var startButton = document.getElementById('startButton');
  var downloadButton = document.getElementById('downloadButton');
  var summaryCard = document.getElementById('summaryCard');
  var summaryList = document.getElementById('summaryList');
  var summaryMeta = document.getElementById('summaryMeta');

  function queryTabs(queryInfo) {
    return new Promise(function (resolve) {
      chrome.tabs.query(queryInfo, function (tabs) {
        resolve(tabs || []);
      });
    });
  }

  function sendMessageToTab(tabId, payload) {
    return new Promise(function (resolve, reject) {
      chrome.tabs.sendMessage(tabId, payload, function (response) {
        if (chrome.runtime.lastError) {
          reject(new Error(chrome.runtime.lastError.message));
          return;
        }
        resolve(response || null);
      });
    });
  }

  async function findNotebookTab() {
    var currentWindowTabs = await queryTabs({ currentWindow: true });
    for (var i = 0; i < currentWindowTabs.length; i += 1) {
      if (currentWindowTabs[i].active && shared.isNotebookUrl(currentWindowTabs[i].url || '')) {
        return currentWindowTabs[i];
      }
    }

    for (var j = 0; j < currentWindowTabs.length; j += 1) {
      if (shared.isNotebookUrl(currentWindowTabs[j].url || '')) {
        return currentWindowTabs[j];
      }
    }

    var allTabs = await queryTabs({});
    for (var k = 0; k < allTabs.length; k += 1) {
      if (shared.isNotebookUrl(allTabs[k].url || '')) {
        return allTabs[k];
      }
    }

    return null;
  }

  function setInlineStatus(message, tone) {
    statusText.textContent = message;
    statusText.className = 'status';
    if (tone === 'error') {
      statusText.classList.add('error');
    } else if (tone === 'success') {
      statusText.classList.add('success');
    }
  }

  function formatColorDistribution(summary) {
    if (!summary || !summary.colorDistribution) {
      return '';
    }

    return Object.keys(summary.colorDistribution)
      .map(function (key) {
        return key + '=' + summary.colorDistribution[key];
      })
      .join(', ');
  }

  function renderSummary(summary, completedAt) {
    if (!summary) {
      summaryCard.classList.add('hidden');
      summaryList.innerHTML = '';
      summaryMeta.textContent = '';
      return;
    }

    summaryCard.classList.remove('hidden');
    summaryList.innerHTML = '';

    var items = [
      { label: '書籍', value: summary.totalBooks || 0 },
      { label: 'ハイライト', value: summary.totalHighlights || 0 },
      { label: 'ノート付き', value: summary.highlightsWithNotes || 0 },
      { label: '色数', value: Object.keys(summary.colorDistribution || {}).length }
    ];

    for (var i = 0; i < items.length; i += 1) {
      var item = items[i];
      var wrapper = document.createElement('div');
      wrapper.className = 'summary-item';

      var strong = document.createElement('strong');
      strong.textContent = String(item.value);
      wrapper.appendChild(strong);

      var span = document.createElement('span');
      span.textContent = item.label;
      wrapper.appendChild(span);

      summaryList.appendChild(wrapper);
    }

    var metaParts = [];
    if (completedAt) {
      metaParts.push(new Date(completedAt).toLocaleString('ja-JP'));
    }

    var colorDistribution = formatColorDistribution(summary);
    if (colorDistribution) {
      metaParts.push('Colors: ' + colorDistribution);
    }

    summaryMeta.textContent = metaParts.join(' / ');
  }

  function renderState(state) {
    var currentState = state || shared.buildInitialState();
    var tone = currentState.status === 'error'
      ? 'error'
      : currentState.status === 'done'
        ? 'success'
        : '';

    setInlineStatus(currentState.message || 'Kindle Notebook タブを開いて「同期開始」を押してください', tone);

    if (currentState.status === 'running' && currentState.totalBooks > 0) {
      progressText.textContent = currentState.currentBookIndex + '/' + currentState.totalBooks + '冊目を処理中... ' + (currentState.currentBookTitle || '');
    } else if (currentState.startedAt) {
      progressText.textContent = '開始: ' + new Date(currentState.startedAt).toLocaleString('ja-JP');
    } else {
      progressText.textContent = '';
    }

    startButton.disabled = currentState.status === 'running';
    downloadButton.disabled = !currentState.downloadReady;
    renderSummary(currentState.summary, currentState.completedAt);
  }

  function loadState() {
    chrome.storage.local.get(stateKey, function (result) {
      renderState(result[stateKey]);
    });
  }

  async function handleStartClick() {
    setInlineStatus('同期対象の Kindle Notebook タブを探しています...');
    progressText.textContent = '';

    var tab = await findNotebookTab();
    if (!tab || typeof tab.id !== 'number') {
      setInlineStatus('Kindle Notebook タブが見つかりません。read.amazon のノートページを開いてください。', 'error');
      return;
    }

    try {
      var response = await sendMessageToTab(tab.id, {
        namespace: shared.NAMESPACE,
        type: messageTypes.START_SYNC
      });

      if (response && response.ok === false) {
        setInlineStatus(response.error || '同期を開始できませんでした', 'error');
        return;
      }

      setInlineStatus('同期を開始しました...');
      loadState();
    } catch (error) {
      setInlineStatus('このタブでは同期を開始できません。Kindle Notebook を開いた状態で再試行してください。', 'error');
    }
  }

  async function handleDownloadClick() {
    var tab = await findNotebookTab();
    if (!tab || typeof tab.id !== 'number') {
      setInlineStatus('JSON を取得するには、同期した Kindle Notebook タブを開いたままにしてください。', 'error');
      return;
    }

    try {
      var response = await sendMessageToTab(tab.id, {
        namespace: shared.NAMESPACE,
        type: messageTypes.GET_RESULT
      });

      if (!response || !response.ok || !response.result) {
        setInlineStatus('ダウンロード用データが見つかりませんでした。同期完了後に同じ Kindle タブで再試行してください。', 'error');
        return;
      }

      var filename = utils.safeFilenamePart(
        'kindle-highlights-' + (response.result.syncedAt || new Date().toISOString())
      ) + '.json';
      await utils.downloadJsonFile(filename, response.result);
      setInlineStatus('JSON ダウンロードを開始しました。', 'success');
    } catch (error) {
      setInlineStatus('JSON ダウンロードに失敗しました: ' + (error && error.message ? error.message : 'unknown error'), 'error');
    }
  }

  chrome.storage.onChanged.addListener(function (changes, areaName) {
    if (areaName !== 'local' || !changes[stateKey]) {
      return;
    }
    renderState(changes[stateKey].newValue);
  });

  startButton.addEventListener('click', function () {
    void handleStartClick();
  });

  downloadButton.addEventListener('click', function () {
    void handleDownloadClick();
  });

  loadState();
})(globalThis);
