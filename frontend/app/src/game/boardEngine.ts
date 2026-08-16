/**
 * BoardEngine — the client-side canvas op log and its bitmap.
 *
 * Responsibilities:
 *   - Owns the offscreen logical board bitmap (1024x768) and the ordered op log.
 *   - LOCAL ECHO: the drawer's in-progress stroke renders immediately on a
 *     separate preview layer (zero perceived latency); the committed op is
 *     confirmed by the server broadcast (which we also receive) and reconciled
 *     by op id, so the drawer never "double paints".
 *   - Applies remote ops in server order; undo/redo/clear rebuild from the log
 *     exactly like the server does (same semantics as room.go applyUndo/applyRedo).
 *   - Bandwidth: outgoing strokes are point-decimated (min-distance filter),
 *     rounded to 0.1 px, and chunked to ≤256 points per op; long strokes
 *     continue seamlessly in follow-up ops sharing the same visual result.
 */

import {
  BOARD_HEIGHT,
  BOARD_WIDTH,
  MAX_STROKE_POINTS,
  clampBoardPoint,
  compactPoint,
  type DrawOp,
  type Point,
} from './drawTypes';
import { applyOp, clearBoard, createBoardCanvas, pickColor, replay } from './renderer';

export type EngineListener = () => void;

export class BoardEngine {
  readonly canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private log: DrawOp[] = [];
  /** Ops sent locally but not yet confirmed by a server broadcast. */
  private pendingIDs = new Set<string>();
  private listeners = new Set<EngineListener>();
  private localCounter = 0;
  private clientTag: string;

  constructor(clientTag?: string) {
    this.canvas = createBoardCanvas();
    const ctx = this.canvas.getContext('2d', { willReadFrequently: true });
    if (!ctx) throw new Error('canvas 2d context unavailable');
    this.ctx = ctx;
    this.clientTag = clientTag ?? Math.random().toString(36).slice(2, 8);
    clearBoard(this.ctx);
  }

  onChange(fn: EngineListener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(): void {
    this.listeners.forEach((fn) => fn());
  }

  /** Generates a client-unique op id the server will keep on broadcast. */
  nextOpID(): string {
    this.localCounter += 1;
    return `c-${this.clientTag}-${Date.now().toString(36)}-${this.localCounter}`;
  }

  getLog(): readonly DrawOp[] {
    return this.log;
  }

  colorAt(x: number, y: number): string {
    return pickColor(this.ctx, x, y);
  }

  /** Full re-sync from the server (canvas_sync on join/reconnect). */
  sync(ops: DrawOp[]): void {
    this.log = ops.filter(isVisualOp);
    this.pendingIDs.clear();
    replay(this.ctx, this.log);
    this.emit();
  }

  /**
   * Applies an op that the local user created, before server confirmation.
   * Visual ops paint immediately; the op id is remembered so the server's
   * echo broadcast is not applied twice.
   */
  applyLocal(op: DrawOp): void {
    if (op.op === 'undo' || op.op === 'redo') {
      // Undo/redo are server-authoritative (the server picks the target op).
      // Don't predict them — round-trip is imperceptible for a button press.
      return;
    }
    if (op.op === 'clear') {
      this.log = [];
      clearBoard(this.ctx);
      this.emit();
      return;
    }
    if (op.id) this.pendingIDs.add(op.id);
    this.log.push(op);
    applyOp(this.ctx, op);
    this.emit();
  }

  /** Applies a server-broadcast op (including our own echoes). */
  applyRemote(op: DrawOp): void {
    switch (op.op) {
      case 'clear': {
        this.log = [];
        this.pendingIDs.clear();
        clearBoard(this.ctx);
        break;
      }
      case 'undo': {
        if (op.target_id) {
          this.log = this.log.filter((o) => o.id !== op.target_id);
          replay(this.ctx, this.log);
        }
        break;
      }
      case 'redo': {
        if (op.target) {
          this.log.push(op.target);
          applyOp(this.ctx, op.target);
        }
        break;
      }
      default: {
        if (op.id && this.pendingIDs.has(op.id)) {
          // Our own echo — already painted via local echo. Adopt the
          // server-authoritative metadata (server_seq/user_id) in the log.
          this.pendingIDs.delete(op.id);
          const idx = this.log.findIndex((o) => o.id === op.id);
          if (idx >= 0) this.log[idx] = op;
          break;
        }
        this.log.push(op);
        applyOp(this.ctx, op);
        break;
      }
    }
    this.emit();
  }

  destroy(): void {
    this.listeners.clear();
  }
}

function isVisualOp(op: DrawOp): boolean {
  return op.op === 'stroke' || op.op === 'erase' || op.op === 'shape' || op.op === 'fill' || op.op === 'text';
}

// ---------------------------------------------------------------------------
// Stroke building (decimation + pressure banding + chunking)
// ---------------------------------------------------------------------------

/**
 * Minimum distance (board px) between kept samples. Pointer events can fire
 * >120 Hz; this typically cuts point count ~4-6x with no visible difference
 * after midpoint smoothing.
 */
const MIN_SAMPLE_DIST = 2.2;

export type StrokeChunk = {
  points: Point[];
  /** Pressure-scaled brush size for this chunk (server caps at 64). */
  size: number;
};

/**
 * StrokeBuilder collects pointer samples for one continuous gesture,
 * decimates them, tracks average pressure, and cuts the gesture into wire
 * chunks of ≤ MAX_STROKE_POINTS. Chunks overlap by one point so the curve
 * has no seams.
 *
 * Pressure: the wire has one size per op, so pressure modulates size at
 * chunk granularity — a chunk is also cut when average pressure drifts
 * >18% from the chunk's start, giving visible thick/thin response on pens
 * while staying protocol-compatible.
 */
export class StrokeBuilder {
  private samples: Point[] = [];
  private pressures: number[] = [];
  private baseSize: number;
  private pressureEnabled: boolean;
  private chunkStartPressure = 0.5;

  constructor(baseSize: number, pressureEnabled: boolean) {
    this.baseSize = baseSize;
    this.pressureEnabled = pressureEnabled;
  }

  get pointCount(): number {
    return this.samples.length;
  }

  /** Read-only view of the in-progress samples (for local preview). */
  currentSamples(): readonly Point[] {
    return this.samples;
  }

  /** Returns true when the new sample was kept (passed decimation). */
  add(p: Point, pressure: number): boolean {
    const clamped = clampBoardPoint(p);
    const prev = this.samples[this.samples.length - 1];
    if (prev) {
      const dx = clamped.x - prev.x;
      const dy = clamped.y - prev.y;
      if (dx * dx + dy * dy < MIN_SAMPLE_DIST * MIN_SAMPLE_DIST) return false;
    }
    this.samples.push(compactPoint(clamped));
    this.pressures.push(this.pressureEnabled && pressure > 0 ? pressure : 0.5);
    if (this.samples.length === 1) this.chunkStartPressure = this.pressures[0];
    return true;
  }

  private avgPressure(): number {
    if (this.pressures.length === 0) return 0.5;
    let sum = 0;
    for (const p of this.pressures) sum += p;
    return sum / this.pressures.length;
  }

  chunkSize(): number {
    if (!this.pressureEnabled) return this.baseSize;
    // Map pressure 0..1 → 0.4x..1.6x of base size.
    const scale = 0.4 + this.avgPressure() * 1.2;
    return Math.min(64, Math.max(1, Math.round(this.baseSize * scale * 10) / 10));
  }

  /**
   * Whether the gesture should be flushed into a wire chunk now:
   * either the point budget is nearly full, or pressure drifted enough
   * that the stroke width should visibly change.
   */
  shouldCut(): boolean {
    if (this.samples.length >= MAX_STROKE_POINTS) return true;
    if (!this.pressureEnabled || this.samples.length < 12) return false;
    const current = this.avgPressure();
    return Math.abs(current - this.chunkStartPressure) > 0.18;
  }

  /**
   * Cuts the current samples into a chunk and re-seeds the builder with the
   * last point so the next chunk continues seamlessly.
   */
  cut(): StrokeChunk | null {
    if (this.samples.length < 2) return null;
    const chunk: StrokeChunk = { points: this.samples, size: this.chunkSize() };
    const seed = this.samples[this.samples.length - 1];
    const seedPressure = this.pressures[this.pressures.length - 1];
    this.samples = [seed];
    this.pressures = [seedPressure];
    this.chunkStartPressure = seedPressure;
    return chunk;
  }

  /** Flushes whatever remains at gesture end (pointer up). */
  finish(): StrokeChunk | null {
    if (this.samples.length === 0) return null;
    if (this.samples.length === 1) {
      // A dot: duplicate the point so the server's 2-point minimum passes.
      this.samples.push({ ...this.samples[0] });
    }
    const chunk: StrokeChunk = { points: this.samples, size: this.chunkSize() };
    this.samples = [];
    this.pressures = [];
    return chunk;
  }
}

export { BOARD_WIDTH, BOARD_HEIGHT };
