(function (global) {
  var namespace = global.KindleHighlightsMvp || (global.KindleHighlightsMvp = {});

  function sleep(ms) {
    return new Promise(function (resolve) {
      window.setTimeout(resolve, ms);
    });
  }

  function randomBetween(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  }

  function normalizeText(value) {
    return String(value || '').replace(/\s+/g, ' ').trim();
  }

  function normalizeComparisonText(value) {
    return normalizeText(value).toLowerCase();
  }

  function serializeAttributes(element) {
    if (!element || !element.attributes) {
      return {};
    }

    var attributes = {};
    for (var i = 0; i < element.attributes.length; i += 1) {
      var attr = element.attributes[i];
      attributes[attr.name] = attr.value;
    }
    return attributes;
  }

  function serializeDataset(element) {
    if (!element || !element.dataset) {
      return {};
    }

    var dataset = {};
    var keys = Object.keys(element.dataset);
    for (var i = 0; i < keys.length; i += 1) {
      dataset[keys[i]] = element.dataset[keys[i]];
    }
    return dataset;
  }

  function waitForCondition(predicate, timeoutMs, intervalMs) {
    return new Promise(function (resolve, reject) {
      var startedAt = Date.now();
      var timer = window.setInterval(function () {
        var matched = false;
        try {
          matched = Boolean(predicate());
        } catch (_) {
          matched = false;
        }

        if (matched) {
          window.clearInterval(timer);
          resolve(true);
          return;
        }

        if (Date.now() - startedAt >= timeoutMs) {
          window.clearInterval(timer);
          reject(new Error('Timed out waiting for condition'));
        }
      }, intervalMs);
    });
  }

  function hasMeaningfulValue(value) {
    if (value === null || value === undefined) {
      return false;
    }
    if (typeof value === 'string') {
      return normalizeText(value).length > 0;
    }
    if (Array.isArray(value)) {
      return value.length > 0;
    }
    if (typeof value === 'object') {
      return Object.keys(value).length > 0;
    }
    return true;
  }

  function buildColorDistribution(highlights) {
    var distribution = {};
    for (var i = 0; i < highlights.length; i += 1) {
      var color = normalizeText(highlights[i].color || '') || 'unknown';
      distribution[color] = (distribution[color] || 0) + 1;
    }
    return distribution;
  }

  function buildFieldStats(highlights) {
    var counts = {};
    for (var i = 0; i < highlights.length; i += 1) {
      var highlight = highlights[i];
      var keys = Object.keys(highlight);
      for (var j = 0; j < keys.length; j += 1) {
        var key = keys[j];
        if (!counts[key]) {
          counts[key] = 0;
        }
        if (hasMeaningfulValue(highlight[key])) {
          counts[key] += 1;
        }
      }
    }

    var total = highlights.length;
    var stats = {};
    Object.keys(counts).forEach(function (key) {
      stats[key] = {
        count: counts[key],
        total: total,
        ratio: total === 0 ? 0 : counts[key] / total
      };
    });
    return stats;
  }

  function buildSummary(result) {
    var books = Array.isArray(result && result.books) ? result.books : [];
    var highlights = [];
    var perBookCounts = [];

    for (var i = 0; i < books.length; i += 1) {
      var book = books[i];
      var bookHighlights = Array.isArray(book.highlights) ? book.highlights : [];
      highlights = highlights.concat(bookHighlights);
      perBookCounts.push({
        title: normalizeText(book.title || book.bookTitle || '') || 'タイトル不明',
        asin: normalizeText(book.asin || ''),
        count: bookHighlights.length
      });
    }

    var highlightsWithNotes = 0;
    for (var j = 0; j < highlights.length; j += 1) {
      if (hasMeaningfulValue(highlights[j].note)) {
        highlightsWithNotes += 1;
      }
    }

    return {
      totalBooks: books.length,
      totalHighlights: highlights.length,
      highlightsWithNotes: highlightsWithNotes,
      colorDistribution: buildColorDistribution(highlights),
      perBookCounts: perBookCounts,
      detectedFields: buildFieldStats(highlights)
    };
  }

  function safeFilenamePart(value) {
    return normalizeText(value)
      .replace(/[\\/:*?"<>|]+/g, '-')
      .replace(/\s+/g, '-')
      .slice(0, 80) || 'kindle-highlights';
  }

  function downloadJsonFile(filename, data) {
    return new Promise(function (resolve, reject) {
      try {
        var json = JSON.stringify(data, null, 2);
        var blob = new Blob([json], { type: 'application/json' });
        var url = URL.createObjectURL(blob);
        chrome.downloads.download(
          {
            url: url,
            filename: filename,
            saveAs: true
          },
          function (downloadId) {
            if (chrome.runtime.lastError) {
              URL.revokeObjectURL(url);
              reject(new Error(chrome.runtime.lastError.message));
              return;
            }

            window.setTimeout(function () {
              URL.revokeObjectURL(url);
            }, 30000);

            resolve(downloadId);
          }
        );
      } catch (error) {
        reject(error);
      }
    });
  }

  namespace.utils = {
    sleep: sleep,
    randomBetween: randomBetween,
    normalizeText: normalizeText,
    normalizeComparisonText: normalizeComparisonText,
    serializeAttributes: serializeAttributes,
    serializeDataset: serializeDataset,
    waitForCondition: waitForCondition,
    hasMeaningfulValue: hasMeaningfulValue,
    buildSummary: buildSummary,
    buildFieldStats: buildFieldStats,
    buildColorDistribution: buildColorDistribution,
    safeFilenamePart: safeFilenamePart,
    downloadJsonFile: downloadJsonFile
  };
})(globalThis);
