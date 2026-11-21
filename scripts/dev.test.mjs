import assert from "node:assert/strict";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { createCommands } from "./dev.mjs";

const settings = {
  DATABASE_URL: "postgresql://development@localhost/postgres",
  TEST_DATABASE_URL: "postgresql://pair_test@localhost/postgres",
  SUPABASE_URL: "https://class.supabase.co",
  SUPABASE_ANON_KEY: "public-key",
};

async function project(t, values = settings) {
  const rootDir = await mkdtemp(join(tmpdir(), "carpool-env-"));
  t.after(() => rm(rootDir, { recursive: true, force: true }));
  await mkdir(join(rootDir, "frontend"));
  await mkdir(join(rootDir, "backend"));
  await writeFile(join(rootDir, ".env"), Object.entries(values).map(([key, value]) => `${key}="${value}"`).join("\n"));
  return rootDir;
}

function runtime() {
  const calls = [];
  const processes = [];
  return {
    calls, processes,
    async exec(command, args, options) {
      calls.push({ command, args, options });
      return { stdout: "" };
    },
    start(command, args, options) {
      calls.push({ command, args, options });
      let finish;
      const done = new Promise((resolve) => { finish = resolve; });
      const child = { done, finish, stopped: false, async stop() { child.stopped = true; finish(0); } };
      processes.push(child);
      return child;
    },
  };
}

test("dev passes supplied connections and shared Auth settings to local processes", async (t) => {
  const rootDir = await project(t);
  const fake = runtime();
  const controller = new AbortController();
  const running = createCommands({ rootDir, ...fake }).dev(controller.signal);
  while (fake.processes.length < 2) await new Promise((resolve) => setImmediate(resolve));
  controller.abort();
  assert.equal(await running, 0);
  assert.equal(fake.calls.length, 2);
  assert.equal(fake.calls[0].options.env.DATABASE_URL, settings.DATABASE_URL);
  assert.equal(fake.calls[0].options.env.TEST_DATABASE_URL, settings.TEST_DATABASE_URL);
  assert.equal(fake.calls[1].options.env.NEXT_PUBLIC_SUPABASE_URL, settings.SUPABASE_URL);
  assert.equal(fake.calls[1].options.env.NEXT_PUBLIC_SUPABASE_ANON_KEY, settings.SUPABASE_ANON_KEY);
  assert.ok(fake.processes.every((child) => child.stopped));
});

test("setup installs dependencies and checks hosted configuration", async (t) => {
  const rootDir = await project(t);
  const fake = runtime();
  assert.equal(await createCommands({ rootDir, ...fake }).setup(), 0);
  assert.deepEqual(fake.calls.map(({ command, args }) => [command, args]), [
    ["go", ["version"]], ["npm", ["ci"]], ["npm", ["ci"]], ["go", ["run", "./cmd/devdb", "check"]],
  ]);
});

test("missing configuration names the missing key without exposing credentials", async (t) => {
  const rootDir = await project(t, { DATABASE_URL: "postgresql://name:private-password@localhost/postgres" });
  await assert.rejects(createCommands({ rootDir, ...runtime() }).dev(), (error) => {
    assert.match(error.message, /TEST_DATABASE_URL/);
    assert.doesNotMatch(error.message, /private-password/);
    return true;
  });
});

test("dev stops the remaining process when an app exits", async (t) => {
  const rootDir = await project(t);
  const fake = runtime();
  const running = createCommands({ rootDir, ...fake }).dev();
  while (fake.processes.length < 2) await new Promise((resolve) => setImmediate(resolve));
  fake.processes[0].finish(7);
  assert.equal(await running, 7);
  assert.ok(fake.processes.every((child) => child.stopped));
});
