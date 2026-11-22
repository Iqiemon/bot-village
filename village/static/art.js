(function (g) {
  var PALETTE = {
    sky0: "#8ecfe8", sky1: "#a7dcef", sky2: "#c2e9f5", sky3: "#e6f6f2", haze: "#f3ecd2",
    grass0: "#4f8f4a", grass1: "#67a94f", grass2: "#86be5a", grass3: "#a9d270", grass4: "#c8e08c",
    dirt0: "#7a5a3a", dirt1: "#966f46", dirt2: "#b98a55", sand: "#d9b878",
    stone0: "#5c6470", stone1: "#7d8794", stone2: "#a3adb8",
    roof: "#c25b4a", wall: "#ead9b8", wood0: "#6b4a2f", wood1: "#8f6439",
    ink: "#2b2430", paper: "#f6f0dd", shadow: "#3b6b3d"
  };
  var BAYER = [
    0, 8, 2, 10,
    12, 4, 14, 6,
    3, 11, 1, 9,
    15, 7, 13, 5
  ];

  function dither(ctx, x, y, w, h, c1, c2, ratio) {
    ctx.fillStyle = c1;
    ctx.fillRect(x, y, w, h);
    ctx.fillStyle = c2;
    for (var j = 0; j < h; j++) {
      for (var i = 0; i < w; i++) {
        if ((BAYER[(j & 3) * 4 + (i & 3)] + 0.5) / 16 < ratio) {
          ctx.fillRect(x + i, y + j, 1, 1);
        }
      }
    }
  }

  g.art = { PALETTE: PALETTE, dither: dither };
})(globalThis);
