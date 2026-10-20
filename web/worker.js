// A Cloudflare Worker that serves the install script and counts how often it
// is fetched. That count is the number of installs: the script itself sends
// nothing, so the fetch is the one moment
// that can be counted without touching the user's machine.
//
//   /install.sh   the script, from the repository's main branch, counted
//   /             the same for curl and wget; a browser is sent to GitHub
//   /badge.json   a shields.io endpoint: https://img.shields.io/endpoint?url=…/badge.json
//   /count        {"installs": n, "today": n}
//
// Deploy with `wrangler deploy` from this directory, after creating the KV
// namespace named in wrangler.toml.

const REPO = "vitruves/ghostty-config";
const SCRIPT = `https://raw.githubusercontent.com/${REPO}/main/install.sh`;

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    switch (url.pathname) {
      case "/":
      case "/install.sh":
        return serveScript(request, env, ctx, url.pathname === "/");
      case "/badge.json":
        return badge(env);
      case "/count":
        return json({ installs: await total(env), today: await today(env) });
      default:
        return new Response("not found\n", { status: 404 });
    }
  },
};

async function serveScript(request, env, ctx, bare) {
  const agent = request.headers.get("user-agent") || "";
  if (bare && !/curl|wget|fetch|httpie/i.test(agent)) {
    return Response.redirect(`https://github.com/${REPO}`, 302);
  }
  const upstream = await fetch(SCRIPT, { cf: { cacheTtl: 300, cacheEverything: true } });
  if (!upstream.ok) {
    return new Response("the install script could not be fetched from GitHub\n", { status: 502 });
  }
  ctx.waitUntil(bump(env));
  return new Response(await upstream.text(), {
    headers: { "content-type": "text/x-shellscript; charset=utf-8", "cache-control": "no-store" },
  });
}

function dayKey() {
  return "day:" + new Date().toISOString().slice(0, 10);
}

async function bump(env) {
  // KV is eventually consistent: two fetches in the same second may count
  // as one. For a tally of installs that is fine.
  const n = await total(env);
  const d = await today(env);
  await env.COUNTS.put("installs", String(n + 1));
  await env.COUNTS.put(dayKey(), String(d + 1), { expirationTtl: 60 * 60 * 24 * 400 });
}

async function total(env) {
  return parseInt((await env.COUNTS.get("installs")) || "0", 10) || 0;
}

async function today(env) {
  return parseInt((await env.COUNTS.get(dayKey())) || "0", 10) || 0;
}

async function badge(env) {
  const n = await total(env);
  return json({ schemaVersion: 1, label: "installs", message: n.toLocaleString("en-US"), color: "2f7a45", cacheSeconds: 300 });
}

function json(value) {
  return new Response(JSON.stringify(value), {
    headers: { "content-type": "application/json; charset=utf-8", "cache-control": "public, max-age=300" },
  });
}
