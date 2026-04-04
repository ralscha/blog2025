import { spawn } from 'node:child_process';
import { extname, dirname, join, normalize } from 'node:path';
import { fileURLToPath } from 'node:url';

type BunFileLike = Blob & {
  readonly type: string;
  exists(): Promise<boolean>;
};

declare const Bun: {
  serve(options: {
    hostname?: string;
    port?: number;
    routes: Record<string, (request: Request) => Response | Promise<Response>>;
  }): {
    readonly hostname: string;
    readonly port: number;
  };
  file(path: string): BunFileLike;
};

const scriptDir = dirname(fileURLToPath(import.meta.url));
const distDir = join(scriptDir, '..', 'dist', 'sw-update', 'browser');
const defaultDocument = 'index.html';
const hostname = process.env.HOST ?? '127.0.0.1';
const port = Number.parseInt(process.env.PORT ?? '4200', 10);
const shouldOpenBrowser = process.env.OPEN_BROWSER !== 'false';

const noCacheHeaders = {
  'Cache-Control': 'no-store, no-cache, must-revalidate',
  Pragma: 'no-cache',
  Expires: '0',
} satisfies Record<string, string>;

const server = Bun.serve({
  hostname,
  port,
  routes: {
    '/*': handleRequest,
  },
});

console.log(`Serving ${distDir} at http://${server.hostname}:${server.port}`);

if (shouldOpenBrowser) {
  openBrowser(`http://${server.hostname}:${server.port}`);
}

async function handleRequest(request: Request): Promise<Response> {
  const { pathname } = new URL(request.url);
  const relativePath = pathname === '/' ? defaultDocument : safePathFromUrl(pathname);
  const filePath = join(distDir, relativePath);
  const file = Bun.file(filePath);

  if (await file.exists()) {
    return createFileResponse(file, request.method);
  }

  if (!hasFileExtension(pathname)) {
    const indexFile = Bun.file(join(distDir, defaultDocument));

    if (await indexFile.exists()) {
      return createFileResponse(indexFile, request.method);
    }
  }

  return new Response('Not Found', {
    status: 404,
    headers: noCacheHeaders,
  });
}

function safePathFromUrl(pathname: string): string {
  const normalizedPath = normalize(decodeURIComponent(pathname)).replace(/^[\\/]+/, '');

  if (normalizedPath.startsWith('..')) {
    return defaultDocument;
  }

  return normalizedPath;
}

function hasFileExtension(pathname: string): boolean {
  return extname(pathname) !== '';
}

function createFileResponse(file: BunFileLike, method: string): Response {
  const headers = new Headers(noCacheHeaders);

  if (file.type) {
    headers.set('Content-Type', file.type);
  }

  return new Response(method === 'HEAD' ? null : file, {
    headers,
  });
}

function openBrowser(url: string): void {
  const command = getBrowserCommand(url);

  if (!command) {
    return;
  }

  try {
    const child = spawn(command.file, command.args, {
      detached: true,
      stdio: 'ignore',
    });

    child.unref();
  } catch (error) {
    console.warn(`Failed to open browser automatically: ${formatError(error)}`);
  }
}

function getBrowserCommand(url: string): { file: string; args: string[] } | null {
  switch (process.platform) {
    case 'win32':
      return { file: 'cmd', args: ['/c', 'start', '', url] };
    case 'darwin':
      return { file: 'open', args: [url] };
    case 'linux':
      return { file: 'xdg-open', args: [url] };
    default:
      return null;
  }
}

function formatError(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
