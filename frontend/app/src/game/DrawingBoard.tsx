/**
 * DrawingBoard — the in-game drawing surface + tool dock.
 *
 * Layers (bottom → top):
 *   1. board bitmap  — the engine's offscreen canvas, blitted on every change
 *   2. preview layer — local in-progress stroke/shape (drawer only)
 * Both live inside a zoom/pan viewport transform. Zoom/pan are LOCAL VIEW
 * state — they never touch the wire protocol.
 *
 * The component is fully controlled by props: `canDraw` gates every input
 * (guessers get a read-only board), `onSend` emits validated wire ops.
 */
import {
  ALargeSmall,
  ArrowUpRight,
  Brush,
  Circle as CircleIcon,
  Eraser,
  Hand,
  Highlighter,
  Minus,
  MousePointerClick,
  PaintBucket,
  Pen,
  PenTool,
  Pencil,
  Pipette,
  Redo2,
  RotateCcw,
  Slash,
  Square,
  Trash2,
  Triangle as TriangleIcon,
  Undo2,
  ZoomIn,
  ZoomOut,
} from 'lucide-react';
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from 'react';
import { useTranslation } from 'react-i18next';

import {
  BOARD_HEIGHT,
  BOARD_WIDTH,
  MAX_TEXT_LENGTH,
  isStrokeTool,
  type DrawOp,
  type Point,
  type ShapeKind,
  type StrokeTool,
  type ToolId,
} from './drawTypes';
import { type BoardEngine, StrokeBuilder } from './boardEngine';
import { applyOp } from './renderer';

const PALETTE = [
  '#000000',
  '#666666',
  '#ffffff',
  '#e02020',
  '#f7801e',
  '#f7c948',
  '#6dd400',
  '#0f9d58',
  '#12c2e9',
  '#4a98f7',
  '#3b5bdb',
  '#8e44ad',
  '#e84393',
  '#8d5524',
  '#f5cba7',
];

const MIN_ZOOM = 1;
const MAX_ZOOM = 4;

type ShapePreview = { shape: ShapeKind; start: Point; end: Point };

export type DrawingBoardProps = {
  engine: BoardEngine;
  canDraw: boolean;
  onSend: (op: DrawOp) => void;
};

export function DrawingBoard({ engine, canDraw, onSend }: DrawingBoardProps) {
  const { t } = useTranslation();
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const boardCanvasRef = useRef<HTMLCanvasElement | null>(null);
  const previewCanvasRef = useRef<HTMLCanvasElement | null>(null);

  const [tool, setTool] = useState<ToolId>('pencil');
  const [color, setColor] = useState('#000000');
  const [size, setSize] = useState(6);
  const [opacity, setOpacity] = useState(1);
  const [fillShape, setFillShape] = useState(false);
  const [fontSize, setFontSize] = useState(28);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState<Point>({ x: 0, y: 0 });
  const [spaceHeld, setSpaceHeld] = useState(false);
  const [textDraft, setTextDraft] = useState<{ at: Point; value: string } | null>(null);

  // Gesture state lives in refs — pointer events must not re-render per move.
  const strokeRef = useRef<StrokeBuilder | null>(null);
  const strokeToolRef = useRef<StrokeTool | 'eraser'>('pencil');
  const shapeRef = useRef<ShapePreview | null>(null);
  const panningRef = useRef<{ startClient: Point; startPan: Point } | null>(null);
  const activePointerRef = useRef<number | null>(null);

  // ------------------------------------------------------------------
  // Blit engine bitmap → visible canvas whenever the log changes.
  // ------------------------------------------------------------------
  useEffect(() => {
    const blit = () => {
      const visible = boardCanvasRef.current;
      if (!visible) return;
      const ctx = visible.getContext('2d');
      if (!ctx) return;
      ctx.clearRect(0, 0, BOARD_WIDTH, BOARD_HEIGHT);
      ctx.drawImage(engine.canvas, 0, 0);
    };
    blit();
    return engine.onChange(blit);
  }, [engine]);

  // Pan with the keyboard spacebar (desktop convenience).
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.code === 'Space' && !textDraft) setSpaceHeld(true);
    };
    const up = (e: KeyboardEvent) => {
      if (e.code === 'Space') setSpaceHeld(false);
    };
    window.addEventListener('keydown', down);
    window.addEventListener('keyup', up);
    return () => {
      window.removeEventListener('keydown', down);
      window.removeEventListener('keyup', up);
    };
  }, [textDraft]);

  const clampPan = useCallback((p: Point, z: number): Point => {
    const vp = viewportRef.current;
    if (!vp || z <= 1) return { x: 0, y: 0 };
    const rect = vp.getBoundingClientRect();
    const maxX = (rect.width * (z - 1)) / 2;
    const maxY = (rect.height * (z - 1)) / 2;
    return {
      x: Math.min(maxX, Math.max(-maxX, p.x)),
      y: Math.min(maxY, Math.max(-maxY, p.y)),
    };
  }, []);

  const applyZoom = useCallback(
    (next: number) => {
      const z = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Math.round(next * 100) / 100));
      setZoom(z);
      setPan((p) => clampPan(p, z));
    },
    [clampPan],
  );

  /** Converts a client (screen) point into logical board coordinates. */
  const toBoard = useCallback(
    (clientX: number, clientY: number): Point => {
      const vp = viewportRef.current;
      if (!vp) return { x: 0, y: 0 };
      const rect = vp.getBoundingClientRect();
      // Undo the CSS transform: translate(pan) scale(zoom) around center.
      const cx = rect.left + rect.width / 2;
      const cy = rect.top + rect.height / 2;
      const ux = (clientX - cx - pan.x) / zoom + rect.width / 2;
      const uy = (clientY - cy - pan.y) / zoom + rect.height / 2;
      return {
        x: (ux / rect.width) * BOARD_WIDTH,
        y: (uy / rect.height) * BOARD_HEIGHT,
      };
    },
    [pan, zoom],
  );

  const clearPreview = useCallback(() => {
    const canvas = previewCanvasRef.current;
    const ctx = canvas?.getContext('2d');
    if (canvas && ctx) ctx.clearRect(0, 0, canvas.width, canvas.height);
  }, []);

  const drawPreviewOp = useCallback((op: DrawOp) => {
    const canvas = previewCanvasRef.current;
    const ctx = canvas?.getContext('2d');
    if (!canvas || !ctx) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    applyOp(ctx, op);
  }, []);

  /** Paints only the incremental tail of the active stroke (fast path). */
  const previewStrokeTail = useCallback(
    (points: Point[], strokeSize: number) => {
      const canvas = previewCanvasRef.current;
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || points.length < 2) return;
      const wireTool = strokeToolRef.current;
      const op: DrawOp =
        wireTool === 'eraser'
          ? { op: 'erase', size: strokeSize, points }
          : { op: 'stroke', tool: wireTool, color, size: strokeSize, opacity, points };
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      applyOp(ctx, op);
    },
    [color, opacity],
  );

  // ------------------------------------------------------------------
  // Wire senders
  // ------------------------------------------------------------------
  const sendStrokeChunk = useCallback(
    (points: Point[], chunkSize: number) => {
      const wireTool = strokeToolRef.current;
      const op: DrawOp =
        wireTool === 'eraser'
          ? { op: 'erase', id: engine.nextOpID(), size: chunkSize, points }
          : {
              op: 'stroke',
              id: engine.nextOpID(),
              tool: wireTool,
              color,
              size: chunkSize,
              points,
              ...(opacity < 1 ? { opacity } : {}),
            };
      engine.applyLocal(op);
      onSend(op);
    },
    [engine, onSend, color, opacity],
  );

  const commitShape = useCallback(
    (preview: ShapePreview) => {
      const { shape, start, end } = preview;
      let op: DrawOp;
      if (shape === 'line' || shape === 'arrow') {
        op = {
          op: 'shape',
          id: engine.nextOpID(),
          shape,
          color,
          size,
          x: start.x,
          y: start.y,
          width: end.x,
          height: end.y,
          ...(opacity < 1 ? { opacity } : {}),
        };
      } else {
        const x = Math.min(start.x, end.x);
        const y = Math.min(start.y, end.y);
        const w = Math.abs(end.x - start.x);
        const h = Math.abs(end.y - start.y);
        if (w < 2 || h < 2) return;
        op = {
          op: 'shape',
          id: engine.nextOpID(),
          shape,
          color,
          size,
          x,
          y,
          width: w,
          height: h,
          filled: fillShape,
          ...(opacity < 1 ? { opacity } : {}),
        };
      }
      engine.applyLocal(op);
      onSend(op);
    },
    [engine, onSend, color, size, opacity, fillShape],
  );

  const commitText = useCallback(() => {
    if (!textDraft) return;
    const value = textDraft.value.trim().slice(0, MAX_TEXT_LENGTH);
    setTextDraft(null);
    if (!value) return;
    const op: DrawOp = {
      op: 'text',
      id: engine.nextOpID(),
      text: value,
      color,
      x: textDraft.at.x,
      y: textDraft.at.y,
      font_size: fontSize,
      ...(opacity < 1 ? { opacity } : {}),
    };
    engine.applyLocal(op);
    onSend(op);
  }, [textDraft, engine, onSend, color, fontSize, opacity]);

  // ------------------------------------------------------------------
  // Pointer handlers
  // ------------------------------------------------------------------
  const isPanMode = tool === 'pan' || spaceHeld;

  const handlePointerDown = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      if (activePointerRef.current !== null) return; // single-pointer gestures
      if (e.button !== 0 && e.pointerType === 'mouse') return;
      const vp = viewportRef.current;
      if (!vp) return;

      // Clicks on overlay controls (zoom bar, text input) must reach those
      // elements — capturing the pointer here would swallow their click
      // events entirely (this was why zoom buttons appeared dead).
      const target = e.target as HTMLElement;
      if (target.closest('.board-zoom-controls') || target.closest('.board-text-input')) {
        return;
      }

      // Commit any open text draft when clicking elsewhere.
      if (textDraft) {
        commitText();
        return;
      }

      activePointerRef.current = e.pointerId;
      vp.setPointerCapture(e.pointerId);

      if (isPanMode) {
        panningRef.current = { startClient: { x: e.clientX, y: e.clientY }, startPan: pan };
        return;
      }
      if (!canDraw) return;

      const p = toBoard(e.clientX, e.clientY);

      if (tool === 'eyedropper') {
        setColor(engine.colorAt(p.x, p.y));
        setTool('pencil');
        return;
      }
      if (tool === 'fill') {
        const op: DrawOp = {
          op: 'fill',
          id: engine.nextOpID(),
          color,
          x: Math.round(p.x),
          y: Math.round(p.y),
        };
        engine.applyLocal(op);
        onSend(op);
        return;
      }
      if (tool === 'text') {
        setTextDraft({ at: p, value: '' });
        return;
      }
      if (tool.startsWith('shape:')) {
        shapeRef.current = { shape: tool.slice(6) as ShapeKind, start: p, end: p };
        return;
      }
      if (isStrokeTool(tool) || tool === 'eraser') {
        strokeToolRef.current = tool === 'eraser' ? 'eraser' : tool;
        const pressureCapable = e.pointerType === 'pen';
        const builder = new StrokeBuilder(tool === 'eraser' ? size * 2.5 : size, pressureCapable);
        builder.add(p, e.pressure);
        strokeRef.current = builder;
      }
    },
    [textDraft, commitText, isPanMode, canDraw, toBoard, tool, engine, color, size, pan, onSend],
  );

  const handlePointerMove = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      if (activePointerRef.current !== e.pointerId) return;

      if (panningRef.current) {
        const { startClient, startPan } = panningRef.current;
        setPan(
          clampPan(
            { x: startPan.x + (e.clientX - startClient.x), y: startPan.y + (e.clientY - startClient.y) },
            zoom,
          ),
        );
        return;
      }
      if (!canDraw) return;

      const builder = strokeRef.current;
      if (builder) {
        // Use coalesced events when available: full input fidelity at
        // high polling rates without extra renders.
        const native = e.nativeEvent as globalThis.PointerEvent;
        const events =
          typeof native.getCoalescedEvents === 'function' ? native.getCoalescedEvents() : [native];
        let added = false;
        for (const ev of events) {
          if (builder.add(toBoard(ev.clientX, ev.clientY), ev.pressure)) added = true;
        }
        if (!added) return;
        if (builder.shouldCut()) {
          const chunk = builder.cut();
          if (chunk) {
            clearPreview();
            sendStrokeChunk(chunk.points, chunk.size);
          }
        } else {
          previewStrokeTail([...builder.currentSamples()], builder.chunkSize());
        }
        return;
      }

      const shape = shapeRef.current;
      if (shape) {
        shape.end = toBoard(e.clientX, e.clientY);
        const { start, end } = shape;
        const previewOp: DrawOp =
          shape.shape === 'line' || shape.shape === 'arrow'
            ? {
                op: 'shape',
                shape: shape.shape,
                color,
                size,
                opacity,
                x: start.x,
                y: start.y,
                width: end.x,
                height: end.y,
              }
            : {
                op: 'shape',
                shape: shape.shape,
                color,
                size,
                opacity,
                x: Math.min(start.x, end.x),
                y: Math.min(start.y, end.y),
                width: Math.abs(end.x - start.x),
                height: Math.abs(end.y - start.y),
                filled: fillShape,
              };
        drawPreviewOp(previewOp);
      }
    },
    [
      canDraw,
      zoom,
      clampPan,
      toBoard,
      clearPreview,
      sendStrokeChunk,
      previewStrokeTail,
      drawPreviewOp,
      color,
      size,
      opacity,
      fillShape,
    ],
  );

  const endGesture = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      if (activePointerRef.current !== e.pointerId) return;
      activePointerRef.current = null;
      panningRef.current = null;

      const builder = strokeRef.current;
      if (builder) {
        strokeRef.current = null;
        clearPreview();
        const chunk = builder.finish();
        if (chunk) sendStrokeChunk(chunk.points, chunk.size);
      }
      const shape = shapeRef.current;
      if (shape) {
        shapeRef.current = null;
        clearPreview();
        commitShape(shape);
      }
    },
    [clearPreview, sendStrokeChunk, commitShape],
  );

  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      if (!e.ctrlKey && !e.metaKey) return;
      e.preventDefault();
      applyZoom(zoom * (e.deltaY < 0 ? 1.15 : 1 / 1.15));
    },
    [zoom, applyZoom],
  );

  // ------------------------------------------------------------------
  // Toolbar model
  // ------------------------------------------------------------------
  const strokeTools = useMemo(
    () =>
      [
        { id: 'pencil' as ToolId, icon: Pencil, label: t('game.tools.pencil', 'Pencil') },
        { id: 'brush' as ToolId, icon: Brush, label: t('game.tools.brush', 'Brush') },
        { id: 'marker' as ToolId, icon: Highlighter, label: t('game.tools.marker', 'Marker') },
        { id: 'calligraphy' as ToolId, icon: PenTool, label: t('game.tools.calligraphy', 'Calligraphy') },
        { id: 'eraser' as ToolId, icon: Eraser, label: t('game.tools.eraser', 'Eraser') },
        { id: 'fill' as ToolId, icon: PaintBucket, label: t('game.tools.fill', 'Bucket fill') },
        { id: 'eyedropper' as ToolId, icon: Pipette, label: t('game.tools.eyedropper', 'Eyedropper') },
        { id: 'text' as ToolId, icon: ALargeSmall, label: t('game.tools.text', 'Text') },
      ] as const,
    [t],
  );

  const shapeTools = useMemo(
    () =>
      [
        { id: 'shape:line' as ToolId, icon: Slash, label: t('game.tools.line', 'Line') },
        { id: 'shape:arrow' as ToolId, icon: ArrowUpRight, label: t('game.tools.arrow', 'Arrow') },
        { id: 'shape:rectangle' as ToolId, icon: Square, label: t('game.tools.rectangle', 'Rectangle') },
        { id: 'shape:ellipse' as ToolId, icon: CircleIcon, label: t('game.tools.circle', 'Circle') },
        { id: 'shape:triangle' as ToolId, icon: TriangleIcon, label: t('game.tools.triangle', 'Triangle') },
      ] as const,
    [t],
  );

  const sendSimple = useCallback(
    (op: DrawOp['op']) => {
      const wireOp: DrawOp = { op };
      if (op === 'clear') engine.applyLocal(wireOp);
      onSend(wireOp);
    },
    [engine, onSend],
  );

  const cursorClass = isPanMode
    ? 'is-panning'
    : tool === 'fill' || tool === 'eyedropper' || tool === 'text'
      ? 'is-precise'
      : 'is-drawing';

  return (
    <div className="drawing-board">
      {canDraw && (
        <div className="board-tools" role="toolbar" aria-label={t('game.tools.title', 'Drawing tools')}>
          <div className="board-tool-group">
            {strokeTools.map(({ id, icon: Icon, label }) => (
              <button
                key={id}
                type="button"
                title={label}
                aria-label={label}
                className={`board-tool ${tool === id ? 'is-active' : ''}`}
                onClick={() => setTool(id)}
              >
                <Icon size={17} />
              </button>
            ))}
          </div>

          <div className="board-tool-group">
            {shapeTools.map(({ id, icon: Icon, label }) => (
              <button
                key={id}
                type="button"
                title={label}
                aria-label={label}
                className={`board-tool ${tool === id ? 'is-active' : ''}`}
                onClick={() => setTool(id)}
              >
                <Icon size={17} />
              </button>
            ))}
            <button
              type="button"
              title={t('game.tools.fillShape', 'Filled shapes')}
              aria-label={t('game.tools.fillShape', 'Filled shapes')}
              aria-pressed={fillShape}
              className={`board-tool ${fillShape ? 'is-active' : ''}`}
              onClick={() => setFillShape((v) => !v)}
            >
              <MousePointerClick size={17} />
            </button>
          </div>

          <div className="board-tool-group board-sliders">
            <label className="board-slider" title={t('game.tools.size', 'Brush size')}>
              <Pen size={13} aria-hidden />
              <input
                type="range"
                min={1}
                max={40}
                value={size}
                onChange={(e) => setSize(Number(e.target.value))}
                aria-label={t('game.tools.size', 'Brush size')}
              />
              <span className="board-slider-value">{size}</span>
            </label>
            <label className="board-slider" title={t('game.tools.opacity', 'Opacity')}>
              <Minus size={13} aria-hidden />
              <input
                type="range"
                min={10}
                max={100}
                value={Math.round(opacity * 100)}
                onChange={(e) => setOpacity(Number(e.target.value) / 100)}
                aria-label={t('game.tools.opacity', 'Opacity')}
              />
              <span className="board-slider-value">{Math.round(opacity * 100)}%</span>
            </label>
            {tool === 'text' && (
              <label className="board-slider" title={t('game.tools.fontSize', 'Font size')}>
                <ALargeSmall size={13} aria-hidden />
                <input
                  type="range"
                  min={12}
                  max={72}
                  value={fontSize}
                  onChange={(e) => setFontSize(Number(e.target.value))}
                  aria-label={t('game.tools.fontSize', 'Font size')}
                />
                <span className="board-slider-value">{fontSize}</span>
              </label>
            )}
          </div>

          <div className="board-tool-group board-colors">
            {PALETTE.map((c) => (
              <button
                key={c}
                type="button"
                className={`board-color ${color === c ? 'is-active' : ''}`}
                style={{ background: c }}
                aria-label={c}
                onClick={() => setColor(c)}
              />
            ))}
            <label
              className="board-color board-color-custom"
              title={t('game.tools.colorPicker', 'Custom color')}
            >
              <input
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                aria-label={t('game.tools.colorPicker', 'Custom color')}
              />
              <span style={{ background: color }} />
            </label>
          </div>

          <div className="board-tool-group">
            <button
              type="button"
              className="board-tool"
              title={t('game.tools.undo', 'Undo')}
              aria-label={t('game.tools.undo', 'Undo')}
              onClick={() => sendSimple('undo')}
            >
              <Undo2 size={17} />
            </button>
            <button
              type="button"
              className="board-tool"
              title={t('game.tools.redo', 'Redo')}
              aria-label={t('game.tools.redo', 'Redo')}
              onClick={() => sendSimple('redo')}
            >
              <Redo2 size={17} />
            </button>
            <button
              type="button"
              className="board-tool board-tool-danger"
              title={t('game.tools.clear', 'Clear canvas')}
              aria-label={t('game.tools.clear', 'Clear canvas')}
              onClick={() => sendSimple('clear')}
            >
              <Trash2 size={17} />
            </button>
          </div>
        </div>
      )}

      <div className="board-stage">
        <div
          ref={viewportRef}
          className={`board-viewport ${cursorClass} ${canDraw ? '' : 'is-readonly'}`}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={endGesture}
          onPointerCancel={endGesture}
          onWheel={handleWheel}
        >
          <div
            className="board-transform"
            style={{ transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})` }}
          >
            <canvas ref={boardCanvasRef} className="board-layer" width={BOARD_WIDTH} height={BOARD_HEIGHT} />
            <canvas
              ref={previewCanvasRef}
              className="board-layer board-layer-preview"
              width={BOARD_WIDTH}
              height={BOARD_HEIGHT}
            />
            {textDraft && (
              <input
                className="board-text-input"
                style={{
                  left: `${(textDraft.at.x / BOARD_WIDTH) * 100}%`,
                  top: `${(textDraft.at.y / BOARD_HEIGHT) * 100}%`,
                  fontSize: `${fontSize}px`,
                  color,
                }}
                // Focus on mount: the input only exists after an intentional
                // canvas click, so stealing focus is the expected behaviour.
                ref={(el) => el?.focus()}
                maxLength={MAX_TEXT_LENGTH}
                value={textDraft.value}
                placeholder={t('game.tools.textPlaceholder', 'Type…')}
                onChange={(e) => setTextDraft((d) => (d ? { ...d, value: e.target.value } : d))}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') commitText();
                  if (e.key === 'Escape') setTextDraft(null);
                }}
                onBlur={commitText}
              />
            )}
          </div>

          <div className="board-zoom-controls">
            <button
              type="button"
              className="board-tool"
              title={t('game.tools.zoomIn', 'Zoom in')}
              aria-label={t('game.tools.zoomIn', 'Zoom in')}
              onClick={() => applyZoom(zoom * 1.25)}
            >
              <ZoomIn size={15} />
            </button>
            <span className="board-zoom-value">{Math.round(zoom * 100)}%</span>
            <button
              type="button"
              className="board-tool"
              title={t('game.tools.zoomOut', 'Zoom out')}
              aria-label={t('game.tools.zoomOut', 'Zoom out')}
              onClick={() => applyZoom(zoom / 1.25)}
            >
              <ZoomOut size={15} />
            </button>
            {canDraw && (
              <button
                type="button"
                className={`board-tool ${tool === 'pan' ? 'is-active' : ''}`}
                title={t('game.tools.pan', 'Pan')}
                aria-label={t('game.tools.pan', 'Pan')}
                onClick={() => setTool(tool === 'pan' ? 'pencil' : 'pan')}
              >
                <Hand size={15} />
              </button>
            )}
            {(zoom !== 1 || pan.x !== 0 || pan.y !== 0) && (
              <button
                type="button"
                className="board-tool"
                title={t('game.tools.resetView', 'Reset view')}
                aria-label={t('game.tools.resetView', 'Reset view')}
                onClick={() => {
                  setZoom(1);
                  setPan({ x: 0, y: 0 });
                }}
              >
                <RotateCcw size={15} />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
