import {
  chmodSync,
  existsSync,
  readFileSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";

const signalExitCode = {
  SIGINT: 130,
  SIGTERM: 143,
};

const validPid = (pid) => Number.isSafeInteger(pid) && pid > 0;

export const recordOwnedChild = (pidPath, pid) => {
  if (!validPid(pid)) {
    throw new Error("owned child did not provide a valid process ID");
  }
  writeFileSync(pidPath, `${pid}\n`, { mode: 0o600, flag: "wx" });
  chmodSync(pidPath, 0o600);
};

const removeExactPidRecord = (pidPath, pid) => {
  if (!existsSync(pidPath)) {
    return;
  }
  if (readFileSync(pidPath, "utf8").trim() !== String(pid)) {
    throw new Error("owned child PID record changed during cleanup");
  }
  unlinkSync(pidPath);
};

const waitForChildExit = async (child, timeoutMilliseconds) => {
  if (child.exitCode !== null || child.signalCode !== null) {
    return true;
  }
  return await new Promise((resolve) => {
    const onExit = () => {
      clearTimeout(timer);
      resolve(true);
    };
    const timer = setTimeout(() => {
      child.off("exit", onExit);
      resolve(false);
    }, timeoutMilliseconds);
    child.once("exit", onExit);
  });
};

export const terminateOwnedChild = async (child, pidPath) => {
  if (!validPid(child?.pid)) {
    return;
  }
  if (child.exitCode === null && child.signalCode === null) {
    try {
      process.kill(child.pid, "SIGTERM");
    } catch (error) {
      if (error?.code !== "ESRCH") {
        throw error;
      }
    }
  }
  if (!(await waitForChildExit(child, 1_000))) {
    try {
      process.kill(child.pid, "SIGKILL");
    } catch (error) {
      if (error?.code !== "ESRCH") {
        throw error;
      }
    }
    await waitForChildExit(child, 1_000);
  }
  removeExactPidRecord(pidPath, child.pid);
};

export const installOwnedChildSignalCleanup = (child, pidPath) => {
  let handlingSignal = false;
  const handlers = new Map();
  for (const signal of Object.keys(signalExitCode)) {
    const handler = async () => {
      if (handlingSignal) {
        return;
      }
      handlingSignal = true;
      try {
        await terminateOwnedChild(child, pidPath);
      } catch (error) {
        process.stderr.write(`owned child signal cleanup failed: ${error.message}\n`);
      } finally {
        process.exit(signalExitCode[signal]);
      }
    };
    handlers.set(signal, handler);
    process.once(signal, handler);
  }
  return () => {
    for (const [signal, handler] of handlers) {
      process.off(signal, handler);
    }
  };
};
