(async () => {
  if (window.__pitchCheckRunning) return;
  window.__pitchCheckRunning=true;
  const result={passed:false,checks:[],expectedVideos:1};
  const delay=ms=>new Promise(r=>setTimeout(r,ms));
  const assert=(ok,message)=>{if(!ok)throw Error(message)};
  const wait=async fn=>{for(let i=0;i<200;i++){if(fn())return;await delay(100)}throw Error('Timed out: '+fn.toString())};
  const click=text=>{const b=[...document.querySelectorAll('button')].find(b=>b.textContent.trim()===text);assert(b&&!b.disabled,'Missing enabled '+text);b.click()};
  const input=(field,value,event='input')=>{field.value=value;field.dispatchEvent(new Event(event,{bubbles:true}))};
  const ready=()=>[...document.querySelectorAll('button')].some(b=>b.textContent==='New topic'&&!b.disabled);
  const prompt=()=>document.querySelector('.prompt')?.textContent;
  const streams=[];const original=navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
  navigator.mediaDevices.getUserMedia=async c=>{const s=await original(c);streams.push(s);return s};
  const released=()=>streams.every(s=>s.getTracks().every(t=>t.readyState==='ended'));
  try {
    await wait(()=>document.querySelector('#language'));
    input(document.querySelector('#language'),'en','change');document.querySelector('form').requestSubmit();await wait(()=>document.documentElement.lang==='en');
    click('Impromptu');await wait(ready);
    const seen=[prompt()];for(let i=0;i<3;i++){click('New topic');await wait(()=>ready()&&!seen.includes(prompt()));seen.push(prompt())}
    result.checks.push('English builtin topics reroll without recent repeats');
    const boxes=document.querySelectorAll('.topics input[type=checkbox]');
    boxes[1].checked=false;boxes[1].dispatchEvent(new Event('change',{bubbles:true}));
    await wait(()=>document.querySelector('.topics [role=alert]')&&!boxes[0].disabled);
    boxes[0].checked=true;boxes[0].dispatchEvent(new Event('change',{bubbles:true}));
    await wait(()=>ready()&&/[А-Яа-яЁё]/.test(prompt()));
    result.checks.push('Russian builtin set and empty-selection feedback');
    click('Import TXT/CSV');await wait(()=>ready()&&document.querySelectorAll('.topics input[type=checkbox]').length===3);
    assert(['Build a repair cafe, together','Explain a useful mistake','Pitch a quiet workspace'].includes(prompt()),'Imported CSV topic missing');
    const preset=document.querySelector('.durations select');input(preset,'elevator','change');await wait(()=>document.querySelectorAll('.durations input')[1].value==='60');
    input(preset,'demo','change');await wait(()=>document.querySelectorAll('.durations input')[1].value==='180');
    input(preset,'standard','change');await wait(()=>document.querySelectorAll('.durations input')[1].value==='120');
    result.checks.push('quoted CSV import and duration presets');
    const topic=prompt();const fields=document.querySelectorAll('.durations input');input(fields[0],0);input(fields[1],1);
    assert(streams.length===0,'Camera opened before start');click('Start training');await wait(()=>document.querySelector('.recording')?.textContent.includes('Recording'));
    await delay(1300);click('Finish and save');await wait(()=>document.querySelector('.saved'));assert(released(),'Media retained after save');
    click('History');await wait(()=>document.querySelector('.attempt')?.textContent.includes(topic));
    result.checks.push('impromptu topic saved with recording in free-training history');
    click('Impromptu');await wait(ready);input(document.querySelectorAll('.durations input')[0],0);
    click('Start training');await wait(()=>streams.length===2&&streams[1].active);
    click('Training');await wait(()=>document.querySelector('h1')?.textContent==='Training'&&released());
    result.checks.push('switching practice mode releases active capture');
    result.passed=true;
  }catch(error){result.error=String(error.stack||error)}
  result.renderedText=document.body.innerText;
  window.chrome.webview.postMessage('pitch-stage-one:'+JSON.stringify(result));
})();
