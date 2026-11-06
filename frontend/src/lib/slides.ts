import { getDocument, GlobalWorkerOptions } from 'pdfjs-dist';
import workerURL from 'pdfjs-dist/build/pdf.worker.min.mjs?url';
import { ProjectService } from '../../bindings/github.com/scmbr/pitch/internal/app';
import type { Version } from '../../bindings/github.com/scmbr/pitch/internal/projects/models';

GlobalWorkerOptions.workerSrc = workerURL;

// PDF actions and annotations are never executed. Fonts, CMaps and image
// decoders come from the embedded app assets, and all URLs are local.
export async function prepareSlides(version: Version, progress: (page: number, total: number) => void, signal: AbortSignal) {
  if (version.slideCount > 0) return version;
  const task = getDocument({
    url: `/slides/${version.id}/document.pdf`,
    cMapUrl: '/pdfjs/cmaps/', cMapPacked: true,
    standardFontDataUrl: '/pdfjs/standard_fonts/', wasmUrl: '/pdfjs/wasm/',
    useWasm: false, disableWorkerRendering: true,
    maxImageSize: 16_000_000,
  });
  const abort = () => { void task.destroy(); };
  signal.addEventListener('abort', abort, { once: true });
  try {
    const pdf = await task.promise;
    if (pdf.numPages > 500) throw new Error('presentationTooLarge');
    for (let page = 1; page <= pdf.numPages; page++) {
      signal.throwIfAborted();
      progress(page, pdf.numPages);
      const slide = await pdf.getPage(page);
      const original = slide.getViewport({ scale: 1 });
      const viewport = slide.getViewport({ scale: Math.min(1280 / original.width, 1280 / original.height) });
      const canvas = document.createElement('canvas');
      canvas.width = Math.ceil(viewport.width); canvas.height = Math.ceil(viewport.height);
      await slide.render({ canvas, viewport, annotationMode: 0 }).promise;
      await ProjectService.CacheSlide(version.id, page, canvas.toDataURL('image/png').split(',')[1]);
      slide.cleanup(); canvas.width = 0; canvas.height = 0;
    }
    await ProjectService.CompleteSlides(version.id, pdf.numPages);
    return { ...version, slideCount: pdf.numPages };
  } finally { signal.removeEventListener('abort', abort); await task.destroy(); }
}
