import './style.css';
import libheif from 'libheif-js/wasm-bundle';
import JSZip from 'jszip';

// --- DOM Elements ---
const dropZone = document.getElementById('drop-zone');
const fileInput = document.getElementById('file-input');
const fileBtn = document.getElementById('file-btn');
const qualitySlider = document.getElementById('quality-slider');
const qualityValue = document.getElementById('quality-value');
const downloadAllBtn = document.getElementById('download-all-btn');
const resultsContainer = document.getElementById('results');

// --- State ---
const convertedFiles = []; // { name, blob, url }

// --- Quality Slider ---
qualitySlider.addEventListener('input', () => {
  qualityValue.textContent = qualitySlider.value;
});

// --- File Selection ---
fileBtn.addEventListener('click', (e) => {
  e.stopPropagation();
  fileInput.click();
});

dropZone.addEventListener('click', () => {
  fileInput.click();
});

fileInput.addEventListener('change', () => {
  if (fileInput.files.length > 0) {
    handleFiles(Array.from(fileInput.files));
    fileInput.value = '';
  }
});

// --- Drag & Drop ---
dropZone.addEventListener('dragover', (e) => {
  e.preventDefault();
  dropZone.classList.add('drag-over');
});

dropZone.addEventListener('dragleave', () => {
  dropZone.classList.remove('drag-over');
});

dropZone.addEventListener('drop', (e) => {
  e.preventDefault();
  dropZone.classList.remove('drag-over');
  const files = Array.from(e.dataTransfer.files).filter((f) =>
    /\.heic$/i.test(f.name) || /\.heif$/i.test(f.name)
  );
  if (files.length > 0) {
    handleFiles(files);
  }
});

// --- Prevent default drag on window ---
window.addEventListener('dragover', (e) => e.preventDefault());
window.addEventListener('drop', (e) => e.preventDefault());

// --- Conversion Pipeline ---
async function handleFiles(files) {
  for (const file of files) {
    const card = createCard(file.name);
    resultsContainer.appendChild(card);
    // Process sequentially to avoid memory pressure
    try {
      await convertFile(file, card);
    } catch (err) {
      showError(card, file.name, err.message);
    }
  }
}

function createCard(filename) {
  const card = document.createElement('div');
  card.className = 'result-card converting';
  card.innerHTML = `
    <div class="preview-placeholder" style="width:100%;aspect-ratio:4/3;background:#f1f5f9;"></div>
    <div class="info">
      <div class="spinner"></div>
      <div>
        <div class="filename">${escapeHtml(filename)}</div>
        <div class="meta">Converting...</div>
      </div>
    </div>
  `;
  return card;
}

async function convertFile(file, card) {
  const arrayBuffer = await file.arrayBuffer();
  const buffer = new Uint8Array(arrayBuffer);

  const decoder = new libheif.HeifDecoder();
  const images = decoder.decode(buffer);

  if (!images || images.length === 0) {
    throw new Error('No images found in file');
  }

  const image = images[0];
  const width = image.get_width();
  const height = image.get_height();

  // Decode to RGBA pixels via callback
  const imageData = await new Promise((resolve, reject) => {
    const output = {
      data: new Uint8ClampedArray(width * height * 4),
      width,
      height,
    };
    image.display(output, (result) => {
      if (!result) {
        return reject(new Error('HEIF decoding failed'));
      }
      resolve(result);
    });
  });

  // Render to canvas and export as JPEG
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext('2d');
  const imgData = new ImageData(
    new Uint8ClampedArray(imageData.data),
    width,
    height
  );
  ctx.putImageData(imgData, 0, 0);

  const quality = parseInt(qualitySlider.value, 10) / 100;
  const blob = await new Promise((resolve) => {
    canvas.toBlob(resolve, 'image/jpeg', quality);
  });

  const jpegName = file.name.replace(/\.heic$/i, '.jpg').replace(/\.heif$/i, '.jpg');
  const url = URL.createObjectURL(blob);

  // Store for ZIP download
  convertedFiles.push({ name: jpegName, blob, url });
  downloadAllBtn.disabled = false;

  // Update card with preview and download button
  showResult(card, url, jpegName, width, height, blob.size);

  // Clean up libheif handles
  try { image.free(); } catch (_) { /* ignore */ }
}

function showResult(card, url, filename, width, height, size) {
  card.className = 'result-card';
  const sizeStr = size > 1024 * 1024
    ? (size / (1024 * 1024)).toFixed(1) + ' MB'
    : (size / 1024).toFixed(0) + ' KB';

  card.innerHTML = `
    <img class="preview" src="${url}" alt="${escapeHtml(filename)}" />
    <div class="info">
      <div class="filename" title="${escapeHtml(filename)}">${escapeHtml(filename)}</div>
      <div class="meta">${width} x ${height} &middot; ${sizeStr}</div>
      <a class="download-btn" href="${url}" download="${escapeHtml(filename)}">Download</a>
    </div>
  `;
}

function showError(card, filename, message) {
  card.className = 'result-card error';
  card.innerHTML = `
    <div style="width:100%;aspect-ratio:4/3;background:#fef2f2;display:flex;align-items:center;justify-content:center;color:#ef4444;">
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
      </svg>
    </div>
    <div class="info">
      <div class="filename">${escapeHtml(filename)}</div>
      <div class="meta">Error: ${escapeHtml(message)}</div>
    </div>
  `;
}

// --- Download All as ZIP ---
downloadAllBtn.addEventListener('click', async () => {
  if (convertedFiles.length === 0) return;

  downloadAllBtn.disabled = true;
  downloadAllBtn.textContent = 'Creating ZIP...';

  try {
    const zip = new JSZip();
    for (const f of convertedFiles) {
      zip.file(f.name, f.blob);
    }
    const zipBlob = await zip.generateAsync({ type: 'blob' });
    const url = URL.createObjectURL(zipBlob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'converted-photos.zip';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  } catch (err) {
    alert('Failed to create ZIP: ' + err.message);
  } finally {
    downloadAllBtn.disabled = false;
    downloadAllBtn.textContent = 'Download All as ZIP';
  }
});

// --- Utility ---
function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}
