// Diagnostic WebView script, injected by a local Go overlay, never by the app.
// __pitchCheck contains expectedDir and nextDir for an isolated test profile.
(async () => {
  if (window.__pitchCheckRunning) return;
  window.__pitchCheckRunning = true;
  const { expectedDir, nextDir } = window.__pitchCheck;
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
  const waitFor = async predicate => {
    for (let attempt = 0; attempt < 150; attempt++) {
      if (predicate()) return;
      await delay(100);
    }
    throw new Error('Timed out: ' + predicate.toString());
  };
  const change = (selector, value, event) => {
    const field = document.querySelector(selector);
    field.value = value;
    field.dispatchEvent(new Event(event, { bubbles: true }));
  };
  const save = () => document.querySelector('form').requestSubmit();
  const result = { passed: false, checks: [] };
  try {
    await waitFor(() => document.querySelector('#dataDir')?.value === expectedDir);
    result.checks.push('settings loaded through real Wails IPC');
    if (nextDir) {
      change('#dataDir', 'relative-folder', 'input');
      save();
      await waitFor(() => !!document.querySelector('[role=alert]'));
      result.checks.push('relative data directory rejected');
      change('#dataDir', expectedDir, 'input');
      change('#language', 'ru', 'change');
      save();
      await waitFor(() => document.documentElement.lang === 'ru' && document.querySelector('[role=status]')?.textContent.includes('Настройки сохранены'));
      assert(document.querySelector('h1').textContent === 'Настройки', 'Russian heading missing');
      result.checks.push('Russian settings saved and rendered');
      change('#language', 'en', 'change');
      save();
      await waitFor(() => document.documentElement.lang === 'en' && document.querySelector('[role=status]')?.textContent.includes('Settings saved'));
      for (const selector of ['#dataDir', 'button[type=button]', '#language', 'button[type=submit]']) {
        const field = document.querySelector(selector);
        assert(field.tabIndex >= 0 && !field.disabled, 'Control not keyboard accessible: ' + selector);
        field.focus();
        assert(document.activeElement === field, 'Control not focusable: ' + selector);
      }
      result.checks.push('English settings saved; main controls focusable');
      const blocked = await new Promise(resolve => {
        const timer = setTimeout(() => resolve(false), 2000);
        document.addEventListener('securitypolicyviolation', event => {
          if (event.effectiveDirective === 'connect-src' && event.blockedURI.includes('offline-check.invalid')) {
            clearTimeout(timer); resolve(true);
          }
        }, { once: true });
        fetch('https://offline-check.invalid/').catch(() => {});
      });
      assert(blocked, 'CSP did not block an external connection');
      result.checks.push('external fetch blocked by CSP');
      change('#dataDir', nextDir, 'input');
      save();
      await waitFor(() => !!document.querySelector('.notice') && !!document.querySelector('[role=status]'));
      assert([...document.querySelectorAll('.hint')].some(item => item.textContent.includes(expectedDir)), 'Active data directory changed before restart');
      result.checks.push('new data directory saved; restart required');
    } else {
      await waitFor(() => document.documentElement.lang === 'en');
      assert(!document.querySelector('.notice'), 'Restart notice remained after restart');
      result.checks.push('saved language and new data directory restored after restart');
    }
    result.passed = true;
  } catch (error) {
    result.error = String(error.stack || error);
  }
  result.renderedText = document.body.innerText;
  window.chrome.webview.postMessage('pitch-stage-one:' + JSON.stringify(result));
})();
