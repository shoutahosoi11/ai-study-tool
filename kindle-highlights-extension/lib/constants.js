(function (global) {
  var namespace = global.KindleHighlightsMvp || (global.KindleHighlightsMvp = {});

  namespace.NAMESPACE = 'kindle-highlights-mvp';
  namespace.STATE_KEY = 'kindle-highlights-mvp:state';
  namespace.MESSAGE_TYPES = {
    START_SYNC: 'KHM_START_SYNC',
    GET_STATE: 'KHM_GET_STATE',
    GET_RESULT: 'KHM_GET_RESULT'
  };
  namespace.AMAZON_NOTEBOOK_HOSTS = [
    'read.amazon.co.jp',
    'read.amazon.com',
    'read.amazon.co.uk',
    'read.amazon.de',
    'read.amazon.fr',
    'read.amazon.es',
    'read.amazon.it',
    'read.amazon.in',
    'read.amazon.ca',
    'read.amazon.com.au',
    'read.amazon.com.mx',
    'read.amazon.com.br',
    'read.amazon.nl',
    'read.amazon.sg'
  ];

  namespace.isNotebookUrl = function (rawUrl) {
    try {
      var url = new URL(rawUrl);
      var hostname = url.hostname.toLowerCase();
      if (namespace.AMAZON_NOTEBOOK_HOSTS.indexOf(hostname) === -1) {
        return false;
      }

      var pathname = url.pathname.toLowerCase();
      return pathname.indexOf('/kp/notebook') !== -1 || pathname.indexOf('/notebook') !== -1;
    } catch (_) {
      return false;
    }
  };

  namespace.buildInitialState = function () {
    return {
      status: 'idle',
      message: 'Kindle Notebook タブを開いて「同期開始」を押してください',
      currentBookIndex: 0,
      totalBooks: 0,
      currentBookTitle: '',
      startedAt: '',
      completedAt: '',
      lastUpdatedAt: new Date().toISOString(),
      amazonDomain: '',
      summary: null,
      errors: [],
      downloadReady: false
    };
  };
})(globalThis);
