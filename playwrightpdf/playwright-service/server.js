const http = require('http');
const { chromium } = require('playwright');

const PORT = process.env.PORT || 3000;

let browserPromise = chromium.launch({
  headless: true,
});

function readTextBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.on('data', chunk => {
      body += chunk;
      if (body.length > 5 * 1024 * 1024) {
        reject(new Error('request body too large'));
        req.destroy();
      }
    });
    req.on('end', () => resolve(body));
    req.on('error', reject);
  });
}

async function generatePdfFromHtml(html) {
  const browser = await browserPromise;
  const context = await browser.newContext();
  const page = await context.newPage();

  await page.setContent(html, { waitUntil: 'networkidle' });
  const pdfBuffer = await page.pdf({
    format: 'A4',
    preferCSSPageSize: true,
    printBackground: true,
    margin: {
      top: '16mm',
      right: '12mm',
      bottom: '16mm',
      left: '12mm',
    },
  });

  await context.close();
  return pdfBuffer;
}

function validateHtmlBody(html) {
  if (typeof html !== 'string' || !html.trim()) {
    return { ok: false, status: 400, error: 'html body is required' };
  }

  return { ok: true };
}

const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok' }));
    return;
  }

  if (req.method === 'POST' && req.url === '/pdf') {
    try {
      // Expected request body: raw HTML string.
      const html = await readTextBody(req);
      const validation = validateHtmlBody(html);
      if (!validation.ok) {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: validation.error }));
        return;
      }

      const pdfBuffer = await generatePdfFromHtml(html);
      res.writeHead(200, {
        'Content-Type': 'application/pdf',
        'Content-Disposition': 'inline; filename="generated.pdf"',
      });
      res.end(pdfBuffer);
      return;
    } catch (err) {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: err.message }));
      return;
    }
  }

  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'not found' }));
});

server.listen(PORT, () => {
  console.log(`playwright-pdf service listening on port ${PORT}`);
});

process.on('SIGINT', async () => {
  const browser = await browserPromise;
  await browser.close();
  process.exit(0);
});

process.on('SIGTERM', async () => {
  const browser = await browserPromise;
  await browser.close();
  process.exit(0);
});
