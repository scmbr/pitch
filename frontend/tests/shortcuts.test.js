import {test} from 'node:test';
import assert from 'node:assert/strict';
import {shortcut} from '../src/lib/shortcuts.js';
test('shortcuts use physical keys for both keyboard layouts and canonical modifier order',()=>{
  assert.equal(shortcut({code:'KeyR',ctrlKey:false,altKey:false,shiftKey:false,metaKey:false}),'KeyR');
  assert.equal(shortcut({code:'KeyH',ctrlKey:true,altKey:false,shiftKey:true,metaKey:false}),'Ctrl+Shift+KeyH');
  assert.equal(shortcut({code:'ShiftLeft',ctrlKey:false,altKey:false,shiftKey:true,metaKey:false}),'');
  assert.equal(shortcut({code:'Tab',ctrlKey:false,altKey:false,shiftKey:false,metaKey:false}),'');
});
