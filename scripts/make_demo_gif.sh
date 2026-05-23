#!/usr/bin/env bash
# Generates assets/demo.gif by recording a pitlist demo session inside mmterm.
# Shows CLI commands first, then the TUI.
# Requirements: mmterm (release build), xdotool, ffmpeg, gifsicle, DISPLAY set.

set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
PITLIST="$REPO/bin/pitlist"
OUT_DIR="$REPO/assets"
TMP_MP4="/tmp/pitlist_demo_$$.mp4"
OUT_GIF="$OUT_DIR/demo.gif"
CFG_DIR="/tmp/pitlist_demo_cfg_$$"
DEMO_DATA="/tmp/pitlist_demo_data_$$"

die() { echo "ERROR: $*" >&2; exit 1; }

[[ -x "$PITLIST" ]] || die "pitlist not built — run: go build -o bin/pitlist ./cmd/pitlist"

MMTERM="mmterm"
command -v mmterm   >/dev/null || die "mmterm not found on PATH"
command -v xdotool  >/dev/null || die "xdotool not found"
command -v ffmpeg   >/dev/null || die "ffmpeg not found"
command -v gifsicle >/dev/null || die "gifsicle not found"
[[ -n "${DISPLAY:-}" ]]        || die "DISPLAY not set"

# Seed demo data before launching the terminal so the shell can use it
mkdir -p "$DEMO_DATA"
"$PITLIST" demo-seed "$DEMO_DATA"

mkdir -p "$OUT_DIR" "$CFG_DIR/mmterm"

# ── Shell wrapper: interactive shell with PITLIST_DATA_DIR pre-set ─────────────
cat > /tmp/pitlist_demo_shell.sh << SHEOF
#!/usr/bin/env bash
export TERM=xterm-256color
export PITLIST_DATA_DIR="$DEMO_DATA"
export PATH="$REPO/bin:$PATH"
export PS1='\$ '
cd "$REPO"
clear
exec bash --norc --noprofile
SHEOF
chmod +x /tmp/pitlist_demo_shell.sh

# ── mmterm config: 1280×720, clean font, our shell wrapper ───────────────────
cat > "$CFG_DIR/mmterm/config.toml" << TOML
[font]
family = "monospace"
size   = 14.0

[window]
width           = 1280
height          = 720
title           = "pitlist demo"
cursor_blink_ms = 500

[shell]
program = "/tmp/pitlist_demo_shell.sh"
TOML

# ── Launch mmterm ─────────────────────────────────────────────────────────────
echo "→ Launching mmterm…"
FFMPEG_PID=0
XDG_CONFIG_HOME="$CFG_DIR" "$MMTERM" &
MMTERM_PID=$!
trap 'kill $MMTERM_PID 2>/dev/null; [[ $FFMPEG_PID -ne 0 ]] && kill $FFMPEG_PID 2>/dev/null; rm -rf "$CFG_DIR" "$DEMO_DATA" /tmp/pitlist_demo_shell.sh; wait 2>/dev/null' EXIT

# Wait for window (up to 8 s)
WID=""
for i in $(seq 1 40); do
    WID=$(xdotool search --pid "$MMTERM_PID" --onlyvisible 2>/dev/null | head -1 || true)
    [[ -n "$WID" ]] && break
    sleep 0.2
done
[[ -n "$WID" ]] || die "mmterm window never appeared"

xdotool windowraise "$WID"
xdotool windowfocus --sync "$WID"
sleep 1.5   # wait for shell to be ready

# Window geometry
WIN_INFO=$(xwininfo -id "$WID" 2>/dev/null)
PX=$(echo "$WIN_INFO" | awk '/Absolute upper-left X:/ {print $NF}')
PY=$(echo "$WIN_INFO" | awk '/Absolute upper-left Y:/ {print $NF}')
W=$(echo  "$WIN_INFO" | awk '/Width:/  {print $NF}')
H=$(echo  "$WIN_INFO" | awk '/Height:/ {print $NF}')
echo "→ Window $WID  ${W}x${H} at ${PX},${PY}"

# ── Start recording ───────────────────────────────────────────────────────────
echo "→ Recording…"
ffmpeg -y \
    -f x11grab -r 20 -s "${W}x${H}" -i ":0.0+${PX},${PY}" \
    -c:v libx264 -preset ultrafast -crf 18 \
    "$TMP_MP4" 2>/dev/null &
FFMPEG_PID=$!
sleep 0.5

# ── Helpers ───────────────────────────────────────────────────────────────────
T()     { xdotool type --window "$WID" --delay 55 "$@"; }
K()     { xdotool key  --window "$WID" --clearmodifiers "$@"; sleep 0.08; }
ENTER() { K Return; }
PAUSE() { sleep "${1:-1}"; }

# ── Demo script ───────────────────────────────────────────────────────────────
PAUSE 1   # let recording settle

# ── Scene 1: list today's tasks ───────────────────────────────────────────────
T "pitlist list"; PAUSE 0.4; ENTER; PAUSE 1.8

# ── Scene 2: list only personal tasks ────────────────────────────────────────
T "pitlist list --label fitness --label health"; PAUSE 0.4; ENTER; PAUSE 1.8

# ── Scene 3: add a new personal task ─────────────────────────────────────────
T 'pitlist add "Plan weekend trip to the coast" -c personal -l travel -p low'; PAUSE 0.4; ENTER; PAUSE 1.2

# ── Scene 4: add a work task ──────────────────────────────────────────────────
T 'pitlist add "Set up staging DB for new schema" -c work -l devops -p high'; PAUSE 0.4; ENTER; PAUSE 1.2

# ── Scene 5: list again to see new tasks ─────────────────────────────────────
T "pitlist list"; PAUSE 0.4; ENTER; PAUSE 2.0

# ── Scene 6: mark a task done ────────────────────────────────────────────────
T "pitlist done t-demo-010"; PAUSE 0.4; ENTER; PAUSE 1.2

# ── Scene 7: log a personal activity ─────────────────────────────────────────
T 'pitlist log add "Evening walk — cleared my head" --tag personal --tag fitness'; PAUSE 0.4; ENTER; PAUSE 1.2

# ── Scene 8: log a work activity ─────────────────────────────────────────────
T 'pitlist log add "PR review for auth middleware" --tag work --tag review'; PAUSE 0.4; ENTER; PAUSE 1.2

# ── Scene 9: show activity log ───────────────────────────────────────────────
T "pitlist log list"; PAUSE 0.4; ENTER; PAUSE 2.0

# ── Scene 10: show agenda ────────────────────────────────────────────────────
T "pitlist agenda"; PAUSE 0.4; ENTER; PAUSE 2.0

# ── Scene 11: open the TUI ───────────────────────────────────────────────────
T "pitlist"; PAUSE 0.4; ENTER; PAUSE 2.5

# ── Scene 12: browse Tasks tab ───────────────────────────────────────────────
PAUSE 1.0
K Down; PAUSE 0.4
K Down; PAUSE 0.5
K Down; PAUSE 0.4

# ── Scene 13: open detail pane ───────────────────────────────────────────────
K Tab;  PAUSE 1.2

# ── Scene 14: switch to Activity tab ─────────────────────────────────────────
K 2;    PAUSE 1.2
K Down; PAUSE 0.4
K Down; PAUSE 0.6

# ── Scene 15: switch to Agenda tab ───────────────────────────────────────────
K 3;    PAUSE 1.5
K Down; PAUSE 0.4

# ── Scene 16: switch to Search tab, search personal ──────────────────────────
K 4;    PAUSE 0.6
xdotool type --window "$WID" --delay 60 "trip"
PAUSE 1.5
K Down; PAUSE 0.4

# ── Scene 16b: esc exits search input, switch tab ────────────────────────────
K 4;    PAUSE 0.6
xdotool type --window "$WID" --delay 60 "zzz"
PAUSE 1.0
K Escape; PAUSE 0.6
K 1;    PAUSE 0.8

# ── Scene 17: back to Tasks, filter ──────────────────────────────────────────
K 1;    PAUSE 0.6
K slash; PAUSE 0.6
xdotool type --window "$WID" --delay 60 "auth"
PAUSE 1.2
K Escape; PAUSE 0.8

# ── Scene 18: quit ───────────────────────────────────────────────────────────
K q;    PAUSE 0.5

# ── Stop recording ────────────────────────────────────────────────────────────
kill $FFMPEG_PID
wait $FFMPEG_PID 2>/dev/null || true
echo "→ Recording stopped."
kill $MMTERM_PID 2>/dev/null || true
trap - EXIT
rm -rf "$CFG_DIR" "$DEMO_DATA" /tmp/pitlist_demo_shell.sh
wait 2>/dev/null || true

# ── MP4 → GIF ────────────────────────────────────────────────────────────────
echo "→ Converting to GIF…"
PALETTE="/tmp/pitlist_pal_$$.png"

# Skip first 0.5 s (startup flicker), 18 fps, scale to 1280 wide
VFBASE="trim=start=0.5,setpts=PTS-STARTPTS,fps=18,scale=1280:-1:flags=lanczos"

ffmpeg -y -i "$TMP_MP4" \
    -vf "${VFBASE},palettegen=stats_mode=diff" \
    -update 1 "$PALETTE" 2>/dev/null

ffmpeg -y -i "$TMP_MP4" -i "$PALETTE" \
    -filter_complex "${VFBASE}[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=5" \
    "$OUT_GIF" 2>/dev/null

rm -f "$PALETTE" "$TMP_MP4"

echo "→ Optimising…"
gifsicle -O3 --lossy=60 --colors 256 "$OUT_GIF" -o "$OUT_GIF"

SIZE=$(du -sh "$OUT_GIF" | cut -f1)
echo "✓  $OUT_GIF  ($SIZE)"
