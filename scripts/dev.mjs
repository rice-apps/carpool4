import { spawn, spawnSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parseEnv } from "node:util";

const scriptPath = fileURLToPath(import.meta.url);
const defaultRootDir = resolve(dirname(scriptPath), "..");

function resolveCommand(command, args) {
  if (command === "npm" && process.env.npm_execpath) {
    return [process.execPath, [process.env.npm_execpath, ...args]];
  }
  return [command, args];
}

function execute(command, args, options = {}) {
  return new Promise((resolvePromise, reject) => {
    const capture = options.capture ?? false;
    let stdout = "";
    const [program, resolvedArgs] = resolveCommand(command, args);
    const child = spawn(program, resolvedArgs, {
      cwd: options.cwd,
      env: options.env,
      stdio: capture
        ? ["ignore", "pipe", options.quiet ? "ignore" : "inherit"]
        : "inherit",
    });

    if (capture) {
      child.stdout.setEncoding("utf8");
      child.stdout.on("data", (chunk) => {
        stdout += chunk;
      });
    }

    child.once("error", (error) => {
      reject(new Error(`Could not run ${command}: ${error.message}`, { cause: error }));
    });
    child.once("close", (code) => {
      if (code === 0) {
        resolvePromise({ stdout });
        return;
      }
      reject(new Error(`${command} exited with code ${code ?? "unknown"}`));
    });
  });
}

function startProcess(command, args, options = {}) {
  const [program, resolvedArgs] = resolveCommand(command, args);
  const child = spawn(program, resolvedArgs, {
    cwd: options.cwd,
    env: options.env,
    stdio: "inherit",
    detached: process.platform !== "win32",
  });
  let exited = false;
  let resolveDone;
  const done = new Promise((resolvePromise) => {
    resolveDone = resolvePromise;
  });

  child.once("error", (error) => {
    console.error(`Could not run ${command}: ${error.message}`);
    exited = true;
    resolveDone(1);
  });
  child.once("close", (code) => {
    exited = true;
    resolveDone(code ?? 1);
  });

  return {
    done,
    async stop() {
      if (!exited && child.pid) {
        if (process.platform === "win32") {
          spawnSync("taskkill", ["/pid", String(child.pid), "/T", "/F"], {
            stdio: "ignore",
          });
        } else {
          try {
            process.kill(-child.pid, "SIGTERM");
          } catch (error) {
            if (error.code !== "ESRCH") {
              throw error;
            }
          }
        }
      }
      return done;
    },
  };
}

function interrupted(signal) {
  if (!signal) {
    return new Promise(() => {});
  }
  if (signal.aborted) {
    return Promise.resolve(0);
  }
  return new Promise((resolvePromise) => {
    signal.addEventListener("abort", () => resolvePromise(0), { once: true });
  });
}

async function environment(rootDir) {
  let contents = "";
  try {
    contents = await readFile(join(rootDir, ".env"), "utf8");
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
  const env = { ...parseEnv(contents), ...process.env };
  for (const key of ["DATABASE_URL", "TEST_DATABASE_URL", "SUPABASE_URL", "SUPABASE_ANON_KEY"]) {
    if (!env[key]) throw new Error(`${key} is missing. Copy your pair's supplied configuration to the root .env file.`);
  }
  return {
    ...env,
    DATABASE_MAX_CONNECTIONS: env.DATABASE_MAX_CONNECTIONS || "2",
    NEXT_PUBLIC_SUPABASE_URL: env.SUPABASE_URL,
    NEXT_PUBLIC_SUPABASE_ANON_KEY: env.SUPABASE_ANON_KEY,
  };
}

export function createCommands({
  rootDir = defaultRootDir,
  nodeVersion = process.versions.node,
  exec = execute,
  start = startProcess,
} = {}) {
  const backendDir = join(rootDir, "backend");
  const frontendDir = join(rootDir, "frontend");
  const startBackend = (env) => start("go", ["run", "./cmd/server"], { cwd: backendDir, env });
  const startFrontend = (env) => start("npm", ["run", "dev"], { cwd: frontendDir, env });

  async function runAttached(startApp, signal) {
    const app = startApp(await environment(rootDir));
    try {
      return await Promise.race([app.done, interrupted(signal)]);
    } finally {
      await app.stop();
    }
  }

  return {
    async setup() {
      const env = await environment(rootDir);
      const [major, minor] = nodeVersion.split(".").map(Number);
      if (major < 20 || (major === 20 && minor < 19)) {
        throw new Error(`Node 20.19 or newer is required; found ${nodeVersion}.`);
      }
      await exec("go", ["version"], { cwd: rootDir, capture: true });
      await exec("npm", ["ci"], { cwd: rootDir });
      await exec("npm", ["ci"], { cwd: frontendDir });
      await exec("go", ["run", "./cmd/devdb", "check"], { cwd: backendDir, env });
      return 0;
    },
    backend(signal) { return runAttached(startBackend, signal); },
    frontend(signal) { return runAttached(startFrontend, signal); },
    async testBackend() {
      await exec("go", ["test", "./..."], { cwd: backendDir, env: await environment(rootDir) });
      return 0;
    },
    async dev(signal) {
      const env = await environment(rootDir);
      const backend = startBackend(env);
      let frontend;
      try {
        frontend = startFrontend(env);
        return await Promise.race([backend.done, frontend.done, interrupted(signal)]);
      } finally {
        const results = await Promise.allSettled([backend.stop(), frontend?.stop()]);
        const failure = results.find((result) => result.status === "rejected");
        if (failure) throw failure.reason;
      }
    },
  };
}

async function main() {
  const commands = createCommands();
  const selected = {
    setup: commands.setup,
    backend: commands.backend,
    frontend: commands.frontend,
    dev: commands.dev,
    "test:backend": commands.testBackend,
  }[process.argv[2]];
  if (!selected) throw new Error("Usage: node scripts/dev.mjs <setup|backend|frontend|dev|test:backend>");
  const controller = new AbortController();
  const interrupt = () => controller.abort();
  process.once("SIGINT", interrupt);
  process.once("SIGTERM", interrupt);
  try {
    return await selected(controller.signal);
  } finally {
    process.removeListener("SIGINT", interrupt);
    process.removeListener("SIGTERM", interrupt);
  }
}

if (resolve(process.argv[1] ?? "") === scriptPath) {
  main().then((code) => { process.exitCode = code; }).catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}