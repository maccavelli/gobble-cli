#!/usr/bin/env python3
"""Write gobble's icon, a cartoon turkey in profile, as a monochrome 24x24 SVG.

docs/decisions/0044-MADR-publish-gobbles-icon-and-its-generator.md, carried
out by 0044-PLAN. The icon is generated, never edited by hand: change the
constants below, look at the previews, and commit this script and the SVG
together. Consumers (magic-cli-remote) fetch the SVG at a `v*` tag.

The art is one colour, and its detail is negative space, so a consumer's
single-colour tint keeps it. A cut-out survives only if no other shape covers
it, so the geometry keeps its gaps:
  * the tail is one scalloped fan, an annular sector around a point behind
    the body, whose inner edge is the body's outline grown by a gap and whose
    outer edge stops short of the head; a striped band is cut out of it;
  * the body has a wing cut out, the head an eye (fill-rule evenodd);
  * the beak, the wattle, the neck and the legs are plain shapes that union
    with what they touch.

Writes:
  assets/gobble.svg   one line of ASCII, no newline at the end (0044-MADR D1)

Usage:
  scripts/gen_icon.py                write assets/gobble.svg
  scripts/gen_icon.py --check        exit 1 if assets/gobble.svg differs
  scripts/gen_icon.py --preview DIR  also write PNG previews into DIR

Exit status is 1 when --check finds a difference, and 2 when --preview lacks
rsvg-convert or ImageMagick's magick.
"""

from __future__ import annotations

import argparse
import math
import shutil
import subprocess
import sys
from pathlib import Path

SVG = Path(__file__).resolve().parent.parent / "assets" / "gobble.svg"

# --- the parts: these constants are the turkey (0044-MADR D2) ---------------------------------------------------
BODY = (12.4, 15.3, 5.5, 4.9, -14)
HEAD = (17.9, 7.1, 2.6)
EYE = (18.8, 6.55, 0.68)                    # cut out of the head
F = (11.2, 14.6)                            # the fan's centre, behind the body
SPAN = (170, 287)                           # degrees, y down: 180 is left, 270 is up
LOBES = 5
R_OUT, SCALLOP = 10.8, 1.45                 # outer radius, and the depth of the notches between feathers
GAP = 1.0                                   # fan to body, and fan to head
BAND = (2.75, 1.7)                          # the cut-out stripe: from R_OUT-SCALLOP-2.75 to ...-1.7
N = 480                                     # steps along the fan
TIE = 1e-9                                  # distances this close are a tie in rdp


def R(v: float) -> str:
    """v to two decimals, with no trailing zeros and no negative zero."""
    s = f"{v:.2f}".rstrip("0").rstrip(".")
    return "0" if s == "-0" else s


def poly(pts: list) -> str:
    """A closed path through pts."""
    return "M" + "L".join(f"{R(x)} {R(y)}" for x, y in pts) + "Z"


def ell_path(cx: float, cy: float, rx: float, ry: float, rot: float) -> str:
    """A closed path round the ellipse, as two arcs."""
    a = math.radians(rot)
    x1, y1 = cx - rx * math.cos(a), cy - rx * math.sin(a)
    x2, y2 = cx + rx * math.cos(a), cy + rx * math.sin(a)
    rr = f"{R(rx)} {R(ry)} {R(rot)}"
    return f"M{R(x1)} {R(y1)}A{rr} 1 0 {R(x2)} {R(y2)}A{rr} 1 0 {R(x1)} {R(y1)}Z"


def circle_path(cx: float, cy: float, r: float) -> str:
    """A closed path round the circle, as two arcs."""
    return f"M{R(cx - r)} {R(cy)}a{R(r)} {R(r)} 0 1 0 {R(2 * r)} 0a{R(r)} {R(r)} 0 1 0 {R(-2 * r)} 0Z"


def ray_ellipse(f, th, cx, cy, rx, ry, rot):
    """Distance along the ray from f at angle th to where it leaves the ellipse (f inside)."""
    a = math.radians(-rot)
    ox, oy = f[0] - cx, f[1] - cy
    ox, oy = ox * math.cos(a) - oy * math.sin(a), ox * math.sin(a) + oy * math.cos(a)
    dx, dy = math.cos(th), math.sin(th)
    dx, dy = dx * math.cos(a) - dy * math.sin(a), dx * math.sin(a) + dy * math.cos(a)
    A = (dx / rx) ** 2 + (dy / ry) ** 2
    B = 2 * (ox * dx / rx ** 2 + oy * dy / ry ** 2)
    C = (ox / rx) ** 2 + (oy / ry) ** 2 - 1
    return (-B + math.sqrt(B * B - 4 * A * C)) / (2 * A)


def ray_circle(f, th, cx, cy, r):
    """Distance along the ray to the first crossing of the circle, or None."""
    dx, dy = math.cos(th), math.sin(th)
    ox, oy = f[0] - cx, f[1] - cy
    b = ox * dx + oy * dy
    c = ox * ox + oy * oy - r * r
    disc = b * b - c
    if disc < 0:
        return None
    t = -b - math.sqrt(disc)
    return t if t > 0 else None


def rdp(pts, eps=0.03):
    """Ramer-Douglas-Peucker: drop points within eps of the line through their neighbours.

    The fan's edge is symmetric, so the farthest point is often a tie. Of the
    distances within TIE of the largest, the first is kept, so the platform's
    maths library does not choose (0044-MADR D2, amendment of 2026-11-07)."""
    if len(pts) < 3:
        return pts
    (x1, y1), (x2, y2) = pts[0], pts[-1]
    den = math.hypot(x2 - x1, y2 - y1) or 1e-9
    dists = [abs((y2 - y1) * x - (x2 - x1) * y + x2 * y1 - y2 * x1) / den for x, y in pts[1:-1]]
    far = max(dists)
    i = next(k for k, d in enumerate(dists) if d >= far - TIE)
    if dists[i] <= eps:
        return [pts[0], pts[-1]]
    return rdp(pts[:i + 2], eps)[:-1] + rdp(pts[i + 1:], eps)


def bez(p0, p1, p2, p3, n=24):
    """n+1 points along the cubic Bezier p0..p3."""
    return [tuple((1 - t) ** 3 * a + 3 * (1 - t) ** 2 * t * b + 3 * (1 - t) * t * t * c + t ** 3 * d
                  for a, b, c, d in zip(p0, p1, p2, p3)) for t in (k / n for k in range(n + 1))]


def fan_paths() -> tuple[str, str]:
    """The tail fan, and the stripe cut out of it ("" when there is no room for one)."""
    body_gap = (BODY[0], BODY[1], BODY[2] + GAP, BODY[3] + GAP, BODY[4])
    head_gap = (HEAD[0], HEAD[1], HEAD[2] + GAP)
    outer, inner, band_out, band_in = [], [], [], []
    for i in range(N + 1):
        u = i / N
        th = math.radians(SPAN[0] + (SPAN[1] - SPAN[0]) * u)
        lobe = abs(math.sin(math.pi * u * LOBES))
        r = R_OUT - SCALLOP + SCALLOP * lobe ** 0.45          # round feather tips, sharp notches between them
        hit = ray_circle(F, th, *head_gap)
        if hit is not None:
            r = min(r, hit)
        r_in = ray_ellipse(F, th, *body_gap)
        assert r > r_in + 0.6, f"the fan is too thin at {math.degrees(th):.0f} degrees"
        outer.append((F[0] + r * math.cos(th), F[1] + r * math.sin(th)))
        inner.append((F[0] + r_in * math.cos(th), F[1] + r_in * math.sin(th)))
        if 0.04 < u < 0.96:
            b_hi, b_lo = R_OUT - SCALLOP - BAND[1], R_OUT - SCALLOP - BAND[0]
            if b_lo > r_in + 0.55 and (hit is None or b_hi < hit - 0.55):
                band_out.append((F[0] + b_hi * math.cos(th), F[1] + b_hi * math.sin(th)))
                band_in.append((F[0] + b_lo * math.cos(th), F[1] + b_lo * math.sin(th)))
    fan = poly(rdp(outer) + rdp(inner[::-1]))
    band = poly(rdp(band_out) + rdp(band_in[::-1])) if len(band_out) > 2 else ""
    return fan, band


def build() -> str:
    """The icon's SVG text."""
    fan, band = fan_paths()
    # The wing: a teardrop swept back along the body, cut out of it.
    wing_top = bez((13.9, 14.6), (13.6, 13.0), (10.6, 13.6), (9.0, 17.2))
    wing_bottom = bez((9.0, 17.2), (11.2, 16.9), (14.2, 16.6), (13.9, 14.6))
    wing = wing_top + wing_bottom[1:]
    # The neck: a curved band from the body's shoulder up into the head.
    neck = bez((14.6, 11.0), (15.9, 10.2), (15.9, 8.6), (16.3, 7.9)) + bez((19.2, 8.6), (18.7, 10.2), (18.4, 11.6), (16.9, 12.6))
    beak = [(20.3, 6.25), (23.1, 7.2), (20.3, 8.2)]
    # The wattle hangs from the chin.
    wattle = bez((19.4, 8.6), (20.8, 8.9), (21.3, 10.7), (20.6, 11.6)) + bez((20.6, 11.6), (19.85, 12.2), (19.25, 10.8), (19.05, 9.3))
    legs = [
        [(10.75, 19.6), (11.75, 19.6), (11.75, 21.7), (13.15, 21.7), (13.15, 22.75), (9.6, 22.75), (9.6, 21.7), (10.75, 21.7)],
        [(14.1, 19.3), (15.1, 19.3), (15.1, 21.7), (16.5, 21.7), (16.5, 22.75), (13.0, 22.75), (13.0, 21.7), (14.1, 21.7)],
    ]

    # Nothing may cover the eye or the wing.
    for p in [*neck, *beak, *wattle]:
        assert math.dist(p, EYE[:2]) > EYE[2] + 0.35, "a part covers the eye"
    wx = [p[0] for p in wing]
    wy = [p[1] for p in wing]
    for p in neck:
        assert not (min(wx) - 0.5 < p[0] < max(wx) + 0.5 and min(wy) - 0.5 < p[1] < max(wy) + 0.5), "the neck covers the wing"

    paths = [
        fan + band,                                                   # the tail, with its stripe cut out
        ell_path(*BODY) + poly(rdp(wing_top) + rdp(wing_bottom)[1:-1]),  # the body, with the wing cut out
        circle_path(*HEAD) + circle_path(*EYE),                       # the head, with the eye cut out
        poly(rdp(neck)), poly(beak), poly(rdp(wattle)), *(poly(l) for l in legs),
    ]
    return ('<svg fill="currentColor" fill-rule="evenodd" height="1em" style="flex:none;line-height:1" '
            'viewBox="0 0 24 24" width="1em" xmlns="http://www.w3.org/2000/svg"><title>gobble</title>'
            + "".join(f'<path d="{d}"/>' for d in paths) + "</svg>")


def preview(out: Path) -> int:
    """PNGs of SVG into out at 512, 72, 48 and 24 px, and the 48 and 24 px pixel-enlarged; 2 if a tool is missing."""
    for tool in ("rsvg-convert", "magick"):
        if shutil.which(tool) is None:
            print(f"gen_icon: --preview needs {tool}, which is not on PATH", file=sys.stderr)
            return 2
    out.mkdir(parents=True, exist_ok=True)
    for size in (512, 72, 48, 24):
        subprocess.run(["rsvg-convert", "-w", str(size), "-h", str(size), "-b", "white", "-o",
                        str(out / f"gobble-{size}.png"), str(SVG)], check=True)
    for size in (48, 24):
        subprocess.run(["magick", str(out / f"gobble-{size}.png"), "-filter", "point", "-resize", "384x384",
                        str(out / f"gobble-{size}-x.png")], check=True)
    print(f"gen_icon: previews written to {out}")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="Write gobble's icon, assets/gobble.svg.")
    mode = ap.add_mutually_exclusive_group()
    mode.add_argument("--check", action="store_true", help="write nothing; exit 1 if assets/gobble.svg differs")
    mode.add_argument("--preview", type=Path, metavar="DIR", help="also write PNG previews into DIR")
    args = ap.parse_args()
    data = build().encode("utf-8")
    if args.check:
        if not SVG.is_file() or SVG.read_bytes() != data:
            print("gen_icon: assets/gobble.svg differs from the generator; run scripts/gen_icon.py", file=sys.stderr)
            return 1
        return 0
    SVG.parent.mkdir(parents=True, exist_ok=True)
    SVG.write_bytes(data)
    print(f"gen_icon: {SVG.name}: {len(data)} bytes")
    if args.preview is not None:
        return preview(args.preview)
    return 0


if __name__ == "__main__":
    sys.exit(main())
