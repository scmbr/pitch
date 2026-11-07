// Summarizes a diagnostic Chromium NetLog without publishing local addresses.
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
export function summarize(log) {
  const types = log.constants?.logEventTypes;
  if (!types || !Array.isArray(log.events) ||
      !['TCP_CONNECT_ATTEMPT', 'UDP_BYTES_SENT', 'UDP_CONNECT'].every(key => Number.isInteger(types[key]))) {
    throw new Error('Invalid or unsupported NetLog');
  }
  const names = Object.fromEntries(Object.entries(types).map(([name, id]) => [id, name]));
  const counts = {};
  let attempts = 0;
  let external = 0;
  for (const event of log.events) {
    const name = names[event.type];
    counts[name] = (counts[name] ?? 0) + 1;
    // Count begin records with endpoints, not both begin/end records.
    if (name === 'TCP_CONNECT_ATTEMPT' && event.params?.address) {
      attempts++;
      const endpoint = event.params.address.toLowerCase();
      const local = endpoint.startsWith('127.') || endpoint.startsWith('[::1]:') ||
        endpoint.startsWith('[::ffff:127.') || endpoint === '::1';
      if (!local) external++;
    }
  }
  return {
    events: log.events.length,
    tcpConnectAttempts: attempts,
    externalTCPConnectAttempts: external,
    udpBytesSentEvents: counts.UDP_BYTES_SENT ?? 0,
    udpRouteProbes: counts.UDP_CONNECT ?? 0,
    note: 'NetLog covers Chromium only; pair with OS tracing. Route probes are not proof of transmitted packets.',
  };
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const path = process.argv[2];
  if (!path) throw new Error('Usage: node scripts/summarize-netlog.mjs <netlog.json>');
  const report = summarize(JSON.parse(readFileSync(path, 'utf8')));
  writeFileSync(path + '.summary.json', JSON.stringify(report, null, 2) + '\n');
  console.log(JSON.stringify(report, null, 2));
  if (report.externalTCPConnectAttempts || report.udpBytesSentEvents) process.exitCode = 1;
}
