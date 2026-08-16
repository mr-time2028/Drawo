/**
 * BoardEngine + StrokeBuilder unit tests.
 *
 * jsdom has no real 2D raster, so rendering is smoke-tested via the mocked
 * canvas context (setup provides one); the op-log semantics (sync, local echo
 * reconciliation, undo/redo/clear) are what we assert precisely — they must
 * mirror the server's room.go behaviour.
 */
import { beforeEach, describe, expect, it } from 'vitest';

import { BoardEngine, StrokeBuilder } from './boardEngine';
import { MAX_STROKE_POINTS, type DrawOp } from './drawTypes';

const stroke = (id: string, extra?: Partial<DrawOp>): DrawOp => ({
  op: 'stroke',
  id,
  tool: 'pencil',
  color: '#112233',
  size: 4,
  points: [
    { x: 1, y: 2 },
    { x: 3, y: 4 },
  ],
  ...extra,
});

describe('BoardEngine', () => {
  let engine: BoardEngine;

  beforeEach(() => {
    engine = new BoardEngine('test');
  });

  it('sync replaces the log with visual ops only', () => {
    engine.sync([stroke('a'), { op: 'undo' }, stroke('b')]);
    expect(engine.getLog().map((o) => o.id)).toEqual(['a', 'b']);
  });

  it('applyLocal + server echo does not duplicate the op', () => {
    const op = stroke('c-1');
    engine.applyLocal(op);
    expect(engine.getLog()).toHaveLength(1);

    // Server echoes the same op back with authoritative metadata.
    engine.applyRemote({ ...op, server_seq: 7, user_id: 'u-1' });
    expect(engine.getLog()).toHaveLength(1);
    expect(engine.getLog()[0].server_seq).toBe(7);
  });

  it('applies remote ops from other users', () => {
    engine.applyRemote(stroke('r-1', { user_id: 'u-2' }));
    expect(engine.getLog()).toHaveLength(1);
  });

  it('remote undo removes the targeted op', () => {
    engine.applyRemote(stroke('r-1'));
    engine.applyRemote(stroke('r-2'));
    engine.applyRemote({ op: 'undo', target_id: 'r-1' });
    expect(engine.getLog().map((o) => o.id)).toEqual(['r-2']);
  });

  it('remote redo restores the embedded target op', () => {
    engine.applyRemote(stroke('r-1'));
    engine.applyRemote({ op: 'undo', target_id: 'r-1' });
    engine.applyRemote({ op: 'redo', target_id: 'r-1', target: stroke('r-1') });
    expect(engine.getLog().map((o) => o.id)).toEqual(['r-1']);
  });

  it('clear empties the log (local and remote)', () => {
    engine.applyLocal(stroke('c-1'));
    engine.applyRemote({ op: 'clear' });
    expect(engine.getLog()).toHaveLength(0);

    engine.applyLocal(stroke('c-2'));
    engine.applyLocal({ op: 'clear' });
    expect(engine.getLog()).toHaveLength(0);
  });

  it('generates unique op ids', () => {
    const a = engine.nextOpID();
    const b = engine.nextOpID();
    expect(a).not.toBe(b);
  });

  it('notifies listeners on change and supports unsubscribe', () => {
    let calls = 0;
    const off = engine.onChange(() => {
      calls += 1;
    });
    engine.applyLocal(stroke('x'));
    off();
    engine.applyLocal(stroke('y'));
    expect(calls).toBe(1);
  });
});

describe('StrokeBuilder', () => {
  it('decimates points closer than the minimum distance', () => {
    const b = new StrokeBuilder(6, false);
    expect(b.add({ x: 10, y: 10 }, 0.5)).toBe(true);
    expect(b.add({ x: 10.5, y: 10.5 }, 0.5)).toBe(false); // too close
    expect(b.add({ x: 20, y: 20 }, 0.5)).toBe(true);
    expect(b.pointCount).toBe(2);
  });

  it('clamps points to board bounds', () => {
    const b = new StrokeBuilder(6, false);
    b.add({ x: -50, y: 99999 }, 0.5);
    const [p] = b.currentSamples();
    expect(p.x).toBeGreaterThanOrEqual(0);
    expect(p.y).toBeLessThanOrEqual(768);
  });

  it('cuts a chunk when the point budget fills and reseeds continuity', () => {
    const b = new StrokeBuilder(6, false);
    for (let i = 0; i < MAX_STROKE_POINTS; i++) {
      b.add({ x: i * 3, y: 0 }, 0.5);
    }
    expect(b.shouldCut()).toBe(true);
    const chunk = b.cut();
    expect(chunk).not.toBeNull();
    expect(chunk!.points.length).toBeLessThanOrEqual(MAX_STROKE_POINTS);
    // Continuity: builder reseeded with the last point of the cut chunk.
    expect(b.pointCount).toBe(1);
    expect(b.currentSamples()[0]).toEqual(chunk!.points[chunk!.points.length - 1]);
  });

  it('finish duplicates single-point dots to satisfy the 2-point minimum', () => {
    const b = new StrokeBuilder(6, false);
    b.add({ x: 5, y: 5 }, 0.5);
    const chunk = b.finish();
    expect(chunk!.points).toHaveLength(2);
  });

  it('pressure scales chunk size within the server cap', () => {
    const soft = new StrokeBuilder(20, true);
    soft.add({ x: 0, y: 0 }, 0.1);
    soft.add({ x: 10, y: 10 }, 0.1);
    const hard = new StrokeBuilder(20, true);
    hard.add({ x: 0, y: 0 }, 1);
    hard.add({ x: 10, y: 10 }, 1);
    expect(soft.chunkSize()).toBeLessThan(hard.chunkSize());
    expect(hard.chunkSize()).toBeLessThanOrEqual(64);
  });

  it('no pressure support → constant size', () => {
    const b = new StrokeBuilder(12, false);
    b.add({ x: 0, y: 0 }, 0.9);
    b.add({ x: 10, y: 10 }, 0.1);
    expect(b.chunkSize()).toBe(12);
  });
});
