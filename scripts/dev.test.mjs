import assert from "node:assert/strict";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import { createCommands, createInterruptHandler } from "./dev.mjs";

const localStatus = {
  API_URL: "http://127.0.0.1:54321",
  DB_URL: "postgresql://postgres:postgres@127.0.0.1:54322/postgres",
  ANON_KEY: "local-anon-key",
};

function fakeRuntime({
  status = localStatus,
  failingCommand,
  failingSupabaseAction,
} = {}) {
  const calls = [];
  const processes = [];

  return {
    calls,
    processes,
    async exec(command, args, options = {}) {
      calls.push({ type: "exec", command, args, options });
      if (command === failingCommand) {
        throw new Error(`${command} failed`);
      }
      if (args.at(-1) === failingSupabaseAction) {
        throw new Error(`Supabase ${failingSupabaseAction} failed`);
      }
      if (command === "go" && args[0] === "version") {
        return { stdout: "go version go1.27.1 test/amd64" };
      }
      if (args.includes("status")) {
        return { stdout: JSON.stringify(status) };
      }
      return { stdout: "" };
    },
    start(command, args, options = {}) {
      calls.push({ type: "start", command, args, options });
      let finish;
      const done = new Promise((resolve) => {
        finish = resolve;
      });
      const process = {
        done,
        finish,
        stopped: false,
        async stop() {
          process.stopped = true;
          finish(0);
        },
      };
      processes.push(process);
      return process;
    },
  };
}

async function temporaryProject() {
  const rootDir = await mkdtemp(join(tmpdir(), "carpool-dev-test-"));
  await mkdir(join(rootDir, "backend"));
  await mkdir(join(rootDir, "frontend"));
  await writeFile(
    join(rootDir, "frontend", "package.json"),
    JSON.stringify({ scripts: { dev: "node -e \"process.exit(0)\"" } }),
  );
  await writeFile(
    join(rootDir, ".env"),
    [
      "SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_ID=test-id",
      "SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_SECRET=test-secret",
      "",
    ].join("\n"),
  );
  return rootDir;
}

test("backend and frontend receive the running local Supabase settings", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime();
  const commands = createCommands({ rootDir, ...runtime });

  try {
    const backend = commands.backend();
    await new Promise((resolve) => setImmediate(resolve));
    runtime.processes[0].finish(0);
    await backend;

    const frontend = commands.frontend();
    await new Promise((resolve) => setImmediate(resolve));
    runtime.processes[1].finish(0);
    await frontend;
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  const starts = runtime.calls.filter(({ type }) => type === "start");
  assert.equal(starts[0].command, "go");
  assert.equal(starts[0].options.cwd, join(rootDir, "backend"));
  assert.equal(starts[0].options.env.SUPABASE_URL, localStatus.API_URL);
  assert.equal(starts[0].options.env.DATABASE_URL, localStatus.DB_URL);

  assert.equal(starts[1].command, "npm");
  assert.equal(starts[1].options.cwd, join(rootDir, "frontend"));
  assert.equal(starts[1].options.env.NEXT_PUBLIC_SUPABASE_URL, localStatus.API_URL);
  assert.equal(starts[1].options.env.NEXT_PUBLIC_SUPABASE_ANON_KEY, localStatus.ANON_KEY);
});

test("frontend can launch npm on the current operating system", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime();
  const commands = createCommands({ rootDir, exec: runtime.exec });

  try {
    assert.equal(await commands.frontend(), 0);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }
});

test("dev stops both app processes and Supabase when interrupted", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime();
  const commands = createCommands({ rootDir, ...runtime });
  const controller = new AbortController();

  try {
    const dev = commands.dev(controller.signal);
    await new Promise((resolve) => setImmediate(resolve));
    controller.abort();
    assert.equal(await dev, 0);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  assert.equal(runtime.processes.length, 2);
  assert.ok(runtime.processes.every(({ stopped }) => stopped));
  assert.deepEqual(
    runtime.calls
      .filter(({ type }) => type === "exec")
      .map(({ args }) => args.slice(1)),
    [["start"], ["status", "--output", "json"], ["stop"]],
  );
});

test("dev stops the other process and Supabase when an app exits", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime();
  const commands = createCommands({ rootDir, ...runtime });

  try {
    const dev = commands.dev();
    await new Promise((resolve) => setImmediate(resolve));
    runtime.processes[0].finish(1);
    assert.equal(await dev, 1);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  assert.ok(runtime.processes.every(({ stopped }) => stopped));
  assert.deepEqual(
    runtime.calls
      .filter(({ type }) => type === "exec")
      .map(({ args }) => args.slice(1)),
    [["start"], ["status", "--output", "json"], ["stop"]],
  );
});

test("dev attempts to stop Supabase when startup fails partway", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime({ failingSupabaseAction: "start" });
  const commands = createCommands({ rootDir, ...runtime });

  try {
    await assert.rejects(commands.dev(), /Supabase start failed/);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  assert.deepEqual(
    runtime.calls.map(({ args }) => args.slice(1)),
    [["start"], ["stop"]],
  );
  assert.equal(runtime.processes.length, 0);
});

test("dev interruption aborts apps before stopping Supabase synchronously", () => {
  const calls = [];
  const controller = new AbortController();
  controller.signal.addEventListener("abort", () => calls.push("abort"));
  const interrupt = createInterruptHandler({
    commandName: "dev",
    controller,
    stopDatabase: () => calls.push("stop database"),
  });

  interrupt();

  assert.deepEqual(calls, ["abort", "stop database"]);
});

test("setup installs dependencies, verifies Supabase, and leaves it stopped", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime();
  const commands = createCommands({
    rootDir,
    nodeVersion: "20.19.0",
    ...runtime,
  });

  try {
    assert.equal(await commands.setup(), 0);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  assert.deepEqual(
    runtime.calls.map(({ command, args, options }) => [command, args, options.cwd]),
    [
      ["go", ["version"], rootDir],
      ["docker", ["info"], rootDir],
      ["npm", ["ci"], rootDir],
      ["npm", ["ci"], join(rootDir, "frontend")],
      [process.execPath, [join(rootDir, "node_modules", "supabase", "dist", "supabase.js"), "start"], rootDir],
      [process.execPath, [join(rootDir, "node_modules", "supabase", "dist", "supabase.js"), "status", "--output", "json"], rootDir],
      [process.execPath, [join(rootDir, "node_modules", "supabase", "dist", "supabase.js"), "stop"], rootDir],
    ],
  );
});

test("setup rejects a missing OAuth configuration before installing anything", async () => {
  const rootDir = await mkdtemp(join(tmpdir(), "carpool-dev-test-"));
  const runtime = fakeRuntime();
  const commands = createCommands({ rootDir, ...runtime });

  try {
    await assert.rejects(commands.setup(), /OAuth credentials are missing/);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  assert.deepEqual(runtime.calls, []);
});

test("setup explains that Docker Desktop must be running", async () => {
  const rootDir = await temporaryProject();
  const runtime = fakeRuntime({ failingCommand: "docker" });
  const commands = createCommands({ rootDir, ...runtime });

  try {
    await assert.rejects(commands.setup(), /Docker Desktop is not running/);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }

  assert.deepEqual(
    runtime.calls.map(({ command, args }) => [command, args]),
    [
      ["go", ["version"]],
      ["docker", ["info"]],
    ],
  );
});
