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
    click('History');
    await wait(() => document.querySelector('.attempt'));
    document.querySelector('.attempt').click();
    await wait(() => document.querySelector('.detail video')?.readyState >= 2);
    const player = document.querySelector('.detail video');
    player.muted = true;
    await player.play();
    await wait(() => player.currentTime > .1);
    player.pause();
    const speed = document.querySelector('.controls select');
    speed.value = '2'; speed.dispatchEvent(new Event('change', { bubbles: true }));
    assert(player.playbackRate === 2, 'Playback rate not applied');
    const seek = document.querySelector('.seek input');
    seek.value = '.5'; seek.dispatchEvent(new Event('input', { bubbles: true }));
    await wait(() => Math.abs(player.currentTime - .5) < .2);
    const thumbnail = document.querySelector('.attempt img');
    await wait(() => thumbnail.complete && thumbnail.naturalWidth > 0);
    const thumbnailURL = thumbnail.src;
    const ratings = document.querySelectorAll('.ratings select');
    for (const [index, value] of [[0,1],[1,5],[2,4]]) {
      ratings[index].value = value; ratings[index].dispatchEvent(new Event('change', { bubbles: true }));
    }
    const note = document.querySelector('textarea'); note.value = 'Local history check'; note.dispatchEvent(new Event('input', { bubbles: true }));
    document.querySelector('.detail form').requestSubmit();
    await wait(() => document.querySelector('[role=status]')?.textContent.includes('Changes saved'));
    click('Settings');
    await wait(() => document.querySelector('#language'));
    click('History');
    await wait(() => document.querySelector('.attempt'));
    document.querySelector('.attempt').click();
    await wait(() => document.querySelector('textarea')?.value === 'Local history check');
    assert(document.querySelectorAll('.ratings select')[1].value === '5', 'Rating not persisted');
    assert(document.querySelector('.attempt img').src === thumbnailURL, 'Thumbnail not cached');
    result.checks.push('local video playback, seek, speed, cached thumbnail, persisted ratings and note');
    click('Training');
    await wait(() => [...document.querySelectorAll('button')].some(b => b.textContent === 'Start training'));
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
    click('History');
    await wait(() => document.querySelector('.attempt'));
    document.querySelector('.attempt').click();
    await wait(() => document.querySelector('.detail'));
    click('Delete recording');
    await wait(() => document.querySelector('.confirm'));
    document.querySelector('.confirm button').click();
    await wait(() => !document.querySelector('.attempt') && document.querySelector('[role=status]'));
    result.checks.push('confirmed deletion removes attempt from history');
    result.expectedVideos = 0;
    result.passed = true;
  } catch (error) { result.error = String(error.stack || error); }
  result.renderedText = document.body.innerText;
  window.chrome.webview.postMessage('pitch-stage-one:' + JSON.stringify(result));
})();
