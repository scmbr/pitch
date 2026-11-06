// Uses synthetic devices supplied only by the diagnostic WebView overlay.
(async () => {
  if (window.__pitchCheckRunning) return;
  window.__pitchCheckRunning = true;
  const result = { passed: false, checks: [] };
  const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (ok, message) => { if (!ok) throw Error(message); };
  const wait = async fn => {
    for (let i = 0; i < 200; i++) { if (fn()) return; await delay(100); }
    throw Error('Timed out: ' + fn.toString());
  };
  const click = text => {
    const button = [...document.querySelectorAll('button')].find(b => b.textContent.trim() === text);
    assert(button && !button.disabled, 'Missing enabled button ' + text); button.click();
  };
  const streams = [];
  const original = navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
  navigator.mediaDevices.getUserMedia = async constraints => {
    const stream = await original(constraints); streams.push(stream); return stream;
  };
  const released = () => streams.every(s => s.getTracks().every(t => t.readyState === 'ended'));
  try {
    await wait(() => document.querySelector('#language'));
    const language = document.querySelector('#language');
    language.value = 'en'; language.dispatchEvent(new Event('change', { bubbles: true }));
    document.querySelector('form').requestSubmit();
    await wait(() => document.documentElement.lang === 'en');
    click('Training');
    await wait(() => document.querySelector('input[type=number]'));
    const durations = document.querySelectorAll('input[type=number]');
    for (const [index, value] of [[0, 1], [1, 2]]) {
      durations[index].value = value; durations[index].dispatchEvent(new Event('input', { bubbles: true }));
    }
    click('Start training');
    await wait(() => document.querySelector('.recording')?.textContent.includes('Recording'));
    result.capture = streams[0].getVideoTracks()[0].getSettings();
    assert(result.capture.width === 1280 && result.capture.height === 720, 'Expected 720p capture');
    await delay(1200);
    click('Pause');
    await wait(() => document.querySelector('.recording')?.textContent.includes('Paused'));
    const time = document.querySelector('output').textContent;
    await delay(1200);
    assert(document.querySelector('output').textContent === time, 'Paused timer advanced');
    click('Resume');
    await wait(() => document.querySelector('output')?.classList.contains('overtime'));
    click('Finish and save');
    await wait(() => document.querySelector('.saved'));
    assert(released(), 'Tracks not released after save');
    result.checks.push('720p capture, pause, overtime, chunked save, tracks released');
    click('Start training');
    await wait(() => streams.length === 2 && streams[1].active);
    click('Settings');
    await wait(released);
    await delay(1000);
    result.checks.push('navigation discards capture and releases every track');
    click('Training');
    await wait(() => document.querySelector('input[type=number]'));
    click('Start training');
    await wait(() => streams.length === 3 && streams[2].active);
    await wait(() => [...document.querySelectorAll('button')].some(b => b.textContent === 'Discard recording' && !b.disabled));
    click('Discard recording');
    await wait(released);
    result.checks.push('explicit discard releases media');
    result.passed = true;
  } catch (error) { result.error = String(error.stack || error); }
  result.renderedText = document.body.innerText;
  window.chrome.webview.postMessage('pitch-stage-one:' + JSON.stringify(result));
})();
