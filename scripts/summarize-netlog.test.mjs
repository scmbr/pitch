import { test } from 'node:test';
import assert from 'node:assert/strict';
import { summarize } from './summarize-netlog.mjs';
const constants={logEventTypes:{TCP_CONNECT_ATTEMPT:1,UDP_BYTES_SENT:2,UDP_CONNECT:3}};
test('counts external attempts and transmissions without exposing endpoints',()=>{
  const report=summarize({constants,events:[
    {type:1,params:{address:'127.0.0.1:80'}},
    {type:1,params:{address:'[::1]:80'}},
    {type:1,params:{address:'192.0.2.1:443'}},
    {type:2}, {type:3},
  ]});
  assert.equal(report.externalTCPConnectAttempts,1);
  assert.equal(report.tcpConnectAttempts,3);
  assert.equal(report.udpBytesSentEvents,1);
  assert.equal(report.udpRouteProbes,1);
  assert.ok(!JSON.stringify(report).includes('192.0.2.1'));
});
test('a route probe alone is not reported as transmitted data',()=>{
  const report=summarize({constants,events:[{type:3}]});
  assert.equal(report.externalTCPConnectAttempts,0);
  assert.equal(report.udpBytesSentEvents,0);
  assert.throws(()=>summarize({events:[]}));
  assert.throws(()=>summarize({constants:{logEventTypes:{}},events:[]}));
});
test('begin/end records are not double counted and repeated attempts remain counted',()=>{
  const report=summarize({constants,events:[
    {type:1,params:{address:'192.0.2.1:443'}}, {type:1,params:{net_error:0}},
    {type:1,params:{address:'192.0.2.1:443'}}, {type:1,params:{net_error:0}},
  ]});
  assert.equal(report.tcpConnectAttempts,2);
  assert.equal(report.externalTCPConnectAttempts,2);
});
