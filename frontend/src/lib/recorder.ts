import { RecordingService } from '../../bindings/github.com/scmbr/pitch/internal/app';

// Uploads are ordered and bounded. A slow/full disk ends capture instead of
// keeping an ever-growing backlog of unencoded recordings in memory.
export class Recorder {
  stream: MediaStream | null = null;
  private media: MediaRecorder | null = null;
  private token = '';
  private uploads = Promise.resolve();
  private pending = 0;
  private failure: unknown = null;
  private cancelled = false;
  onFailure: () => void = () => {};

  async open() {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { width: { ideal: 1280 }, height: { ideal: 720 }, frameRate: { ideal: 15, max: 30 } },
      audio: true,
    });
    if (this.cancelled) { stream.getTracks().forEach(t => t.stop()); throw new Error('cancelled'); }
    this.stream = stream;
    return stream;
  }

  async start() {
    if (!this.stream || this.cancelled) throw new Error('cancelled');
    const mimeType = ['video/webm;codecs=vp8,opus', 'video/webm'].find(t => MediaRecorder.isTypeSupported(t));
    if (!mimeType) throw new Error('unsupportedRecording');
    this.token = await RecordingService.Begin();
    if (this.cancelled) { await RecordingService.Abort(this.token); throw new Error('cancelled'); }
    this.media = new MediaRecorder(this.stream, { mimeType, videoBitsPerSecond: 2_500_000 });
    this.media.ondataavailable = event => {
      if (!event.data.size || this.cancelled || this.failure) return;
      this.pending += event.data.size;
      if (this.pending > 32 * 1024 * 1024) { this.fail(new Error('recordingBacklog')); return; }
      this.uploads = this.uploads.then(async () => {
        if (this.failure || this.cancelled) return;
        for (let offset = 0; offset < event.data.size; offset += 256 * 1024) {
          const bytes = new Uint8Array(await event.data.slice(offset, offset + 256 * 1024).arrayBuffer());
          let binary = '';
          for (const byte of bytes) binary += String.fromCharCode(byte);
          await RecordingService.Append(this.token, btoa(binary));
        }
      }).catch(error => this.fail(error)).finally(() => { this.pending -= event.data.size; });
    };
    this.media.onerror = () => this.fail(new Error('recordingFailed'));
    this.media.start(1000);
  }

  private fail(error: unknown) {
    if (this.failure) return;
    this.failure = error;
    this.release();
    this.onFailure();
  }

  pause(paused: boolean) {
    if (paused && this.media?.state === 'recording') this.media.pause();
    if (!paused && this.media?.state === 'paused') this.media.resume();
  }

  private async stop() {
    if (this.media && this.media.state !== 'inactive') {
      const media = this.media;
      await new Promise<void>(resolve => {
        media.addEventListener('stop', () => resolve(), { once: true });
        media.stop();
      });
    }
    this.release();
    await this.uploads;
  }

  async finish() {
    await this.stop();
    if (this.failure) { await this.abort(); throw this.failure; }
    const id = await RecordingService.Finish(this.token);
    this.token = '';
    return id;
  }

  async abort() {
    this.cancelled = true;
    await this.stop();
    if (this.token) await RecordingService.Abort(this.token);
    this.token = '';
  }

  release() {
    this.stream?.getTracks().forEach(track => track.stop());
    this.stream = null;
  }
}
