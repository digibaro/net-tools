export const STREAMS = 3
export const DOWNLOAD_BYTES = 50 * 1024 * 1024
export const UPLOAD_MIN_BYTES = 8 * 1024 * 1024
export const UPLOAD_MAX_BYTES = 32 * 1024 * 1024 // server limit per request
export const PHASE_MS = 10_000
/** Header carrying the speed-test session token (one rate-limited session per test). */
export const TOKEN_HEADER = 'X-Speedtest-Token'

/** Doubles the upload size while requests finish in under 250 ms, so fast links need few requests. */
export function nextUploadSize(size: number, ms: number): number {
  return ms < 250 ? Math.min(size * 2, UPLOAD_MAX_BYTES) : size
}

/** Megabits per second from bytes over milliseconds. */
export const mbps = (bytes: number, ms: number) => (ms > 0 ? Math.round(bytes * 8 / ms / 1000 * 10) / 10 : 0)

/** Random bytes (incompressible), filled in 64 KiB slices (getRandomValues' limit). */
export function randomPayload(size: number): Uint8Array<ArrayBuffer> {
  const out = new Uint8Array(size)
  for (let i = 0; i < size; i += 65536)
    crypto.getRandomValues(out.subarray(i, Math.min(size, i + 65536)))
  return out
}

/** Reads a response body, counting bytes, until done or the signal aborts. */
export async function drain(res: Response, onBytes: (n: number) => void): Promise<void> {
  const reader = res.body?.getReader()
  if (!reader)
    return
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done)
        return
      onBytes(value.byteLength)
    }
  }
  catch {
    // aborted at the end of the phase
  }
}
