(function (g) {
  var W = 480, H = 270;
  var BANDS = [128, 178, 228];
  var P = g.art.PALETTE;

  function bandY(i) {
    return BANDS[i];
  }

  function paint(ctx) {
    var sky = [
      [44, "sky0"],
      [52, "sky0", "sky1"],
      [74, "sky1"],
      [82, "sky1", "sky2"],
      [98, "sky2"],
      [106, "sky2", "sky3"],
      [118, "sky3"],
      [BANDS[0], "sky3", "haze"]
    ];
    var y = 0;
    sky.forEach(function (s) {
      var end = s[0];
      if (s[2]) {
        g.art.dither(ctx, 0, y, W, end - y, P[s[1]], P[s[2]], 0.5);
      } else {
        ctx.fillStyle = P[s[1]];
        ctx.fillRect(0, y, W, end - y);
      }
      y = end;
    });
    var ground = ["grass3", "grass2", "grass1"];
    var under = ["haze", "grass3", "grass2"];
    BANDS.forEach(function (top, i) {
      ctx.fillStyle = P[ground[i]];
      ctx.fillRect(0, top, W, H - top);
      g.art.dither(ctx, 0, top + 1, W, 4, P[ground[i]], P[under[i]], 0.35);
    });
  }

  function create(canvas) {
    canvas.width = W;
    canvas.height = H;
    var scene = document.createElement("canvas");
    scene.width = W;
    scene.height = H;
    paint(scene.getContext("2d"));
    var ctx = canvas.getContext("2d");
    return {
      draw: function (t) {
        ctx.drawImage(scene, 0, 0);
      },
      W: W,
      H: H
    };
  }

  g.town = { W: W, H: H, bandY: bandY, create: create };
})(globalThis);
