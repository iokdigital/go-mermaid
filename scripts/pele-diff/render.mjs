// render.mjs — render .mmd files with Pele, emit one JSON result per line.
//
// Usage: node render.mjs <file.mmd> [<file.mmd> ...]
//
// Each output line: {file, ok, type, mermaidType, width, height, svgLength}
// or on failure:    {file, ok:false, code, message, line, column}
import { readFileSync } from 'node:fs';
import { render } from 'pele';

for (const file of process.argv.slice(2)) {
  const source = readFileSync(file, 'utf8');
  try {
    const { svg, width, height, type, mermaidType } = render(source);
    console.log(JSON.stringify({
      file, ok: true, type, mermaidType, width, height, svgLength: svg.length,
    }));
  } catch (e) {
    console.log(JSON.stringify({
      file,
      ok: false,
      code: e?.code ?? null,
      message: String(e?.message ?? e).slice(0, 500),
      line: e?.line ?? 0,
      column: e?.column ?? 0,
    }));
  }
}
