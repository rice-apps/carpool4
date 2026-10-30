import { spawn, spawnSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

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

function parseStatus(stdout) {
  let status;
  try {
    status = JSON.parse(stdout);
  } catch (error) {
    throw new Error("Supabase returned invalid status output", { cause: error });
  }

  const apiURL = status.API_URL ?? status.api_url;
  const databaseURL = status.DB_URL ?? status.db_url;
  const anonKey =
    status.PUBLISHABLE_KEY ??
    status.ANON_KEY ??
    status.publishable_key ??
    status.anon_key;
  if (!apiURL || !databaseURL || !anonKey) {
    throw new Error("Supabase status is missing its API URL, database URL, or public key");
  }
  return { apiURL, databaseURL, anonKey };
}

function configuredOAuth(contents) {
  const values = new Map();
  for (const line of contents.split(/\r?\n/)) {
    const separator = line.indexOf("=");
    if (separator < 0) {
      continue;
    }
    values.set(line.slice(0, separator).trim(), line.slice(separator + 1).trim());
  }

  const clientID = values.get("SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_ID");
  const clientSecret = values.get("SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_SECRET");
  return Boolean(clientID && clientSecret);
}

function supportedNode(version) {
  const [major, minor] = version.split(".").map(Number);
  return major > 20 || (major === 20 && minor >= 19);
}

export function createInterruptHandler({
  commandName,
  controller,
  stopDatabase,
}) {
  let handled = false;
  return () => {
    if (handled) {
      return;
    }
    handled = true;
    controller.abort();
    if (commandName === "dev" || commandName === "setup") {
      stopDatabase();
    }
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
  const supabaseProgram = join(
    rootDir,
    "node_modules",
    "supabase",
    "dist",
    "supabase.js",
  );

  const supabase = (args, options = {}) =>
    exec(process.execPath, [supabaseProgram, ...args], {
      cwd: rootDir,
      ...options,
    });

  async function stopDatabaseAfter(primaryError) {
    try {
      await supabase(["stop"]);
    } catch (cleanupError) {
      if (!primaryError) {
        throw cleanupError;
      }
      console.error(`Database cleanup also failed: ${cleanupError.message}`);
    }
  }

  async function localEnvironment() {
    const { stdout } = await supabase(["status", "--output", "json"], {
      capture: true,
    });
    return parseStatus(stdout);
  }

  function startBackend(environment) {
    return start("go", ["run", "./cmd/server"], {
      cwd: backendDir,
      env: {
        ...process.env,
        SUPABASE_URL: environment.apiURL,
        DATABASE_URL: environment.databaseURL,
      },
    });
  }

  function startFrontend(environment) {
    return start("npm", ["run", "dev"], {
      cwd: frontendDir,
      env: {
        ...process.env,
        NEXT_PUBLIC_SUPABASE_URL: environment.apiURL,
        NEXT_PUBLIC_SUPABASE_ANON_KEY: environment.anonKey,
      },
    });
  }

  async function runAttached(startApp, signal) {
    const environment = await localEnvironment();
    const app = startApp(environment);
    const code = await Promise.race([app.done, interrupted(signal)]);
    if (signal?.aborted) {
      await app.stop();
    }
    return code;
  }

  return {
    async setup() {
      let envContents;
      try {
        envContents = await readFile(join(rootDir, ".env"), "utf8");
      } catch {
        throw new Error(
          "OAuth credentials are missing. Copy .env.example to .env and add the provided values.",
        );
      }
      if (!configuredOAuth(envContents)) {
        throw new Error(
          "OAuth credentials are missing. Add both Google OAuth values to .env.",
        );
      }
      if (!supportedNode(nodeVersion)) {
        throw new Error(`Node 20.19 or newer is required; found ${nodeVersion}.`);
      }

      await exec("go", ["version"], { cwd: rootDir, capture: true });
      try {
        await exec("docker", ["info"], {
          cwd: rootDir,
          capture: true,
          quiet: true,
        });
      } catch (error) {
        throw new Error(
          "Docker Desktop is not running. Start it, wait for it to finish loading, and run setup again.",
          { cause: error },
        );
      }
      await exec("npm", ["ci"], { cwd: rootDir });
      await exec("npm", ["ci"], { cwd: frontendDir });

      let setupError;
      try {
        await supabase(["start"]);
        await localEnvironment();
      } catch (error) {
        setupError = error;
        throw error;
      } finally {
        await stopDatabaseAfter(setupError);
      }
      return 0;
    },
    backend(signal) {
      return runAttached(startBackend, signal);
    },
    frontend(signal) {
      return runAttached(startFrontend, signal);
    },
    async dev(signal) {
      let backend;
      let frontend;
      let devError;
      try {
        await supabase(["start"]);
        const environment = await localEnvironment();
        backend = startBackend(environment);
        frontend = startFrontend(environment);
        return await Promise.race([
          backend.done,
          frontend.done,
          interrupted(signal),
        ]);
      } catch (error) {
        devError = error;
        throw error;
      } finally {
        const stoppedApps = await Promise.allSettled([
          backend?.stop() ?? Promise.resolve(),
          frontend?.stop() ?? Promise.resolve(),
        ]);
        const appCleanupError = stoppedApps.find(
          (result) => result.status === "rejected",
        )?.reason;
        if (appCleanupError && devError) {
          console.error(`App cleanup also failed: ${appCleanupError.message}`);
        }
        await stopDatabaseAfter(devError ?? appCleanupError);
        if (!devError && appCleanupError) {
          throw appCleanupError;
        }
      }
    },
  };
}

async function main() {
  const commandName = process.argv[2];
  const commands = createCommands();
  const selected = {
    setup: commands.setup,
    backend: commands.backend,
    frontend: commands.frontend,
    dev: commands.dev,
  }[commandName];

  if (!selected) {
    console.error(
      "Usage: node scripts/dev.mjs <setup|backend|frontend|dev>",
    );
    return 1;
  }

  const controller = new AbortController();
  const interrupt = createInterruptHandler({
    commandName,
    controller,
    stopDatabase() {
      const supabaseProgram = join(
        defaultRootDir,
        "node_modules",
        "supabase",
        "dist",
        "supabase.js",
      );
      // npm on Windows can terminate this coordinator before async cleanup runs.
      spawnSync(process.execPath, [supabaseProgram, "stop"], {
        cwd: defaultRootDir,
        stdio: "inherit",
        timeout: 15_000,
      });
    },
  });
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
  main()
    .then((code) => {
      process.exitCode = code;
    })
    .catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
}
