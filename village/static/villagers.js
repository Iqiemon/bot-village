(function (g) {
  var W = 480;
  var SPEED = 48;
  var BODY = ["roof", "wall", "stone1", "wood1", "sand", "grass1", "stone2", "dirt2"];
  var HAT = ["stone0", "wood0", "grass0", "roof", "stone2", "dirt0"];

  var list = [];
  var byId = {};
  var last = 0;

  var FONT = "bold 7px monospace";
  var measure = document.createElement("canvas").getContext("2d");
  measure.font = FONT;
  function pillW(name) {
    return Math.round(measure.measureText(name).width) + 6;
  }

  function spawn(r) {
    // Seed is a uint64 sent as a string; reduce once to its low 32 bits, which
    // are all the masks below read, so no low bit is lost to float64 rounding.
    var s = Number(BigInt(r.seed || 0) & 0xffffffffn);
    var band = s % 3;
    var homeX = 40 + ((s >>> 4) % 400);
    var homeY = g.town.bandY(band) + 10 + ((s >>> 13) % 24);
    return {
      id: r.sessionId,
      name: r.name,
      seed: s,
      band: band,
      x: homeX < W / 2 ? -10 : W + 10,
      y: homeY,
      state: "walking",
      targetX: homeX,
      targetY: homeY,
      work: false,
      workX: Math.max(24, Math.min(456, homeX + (s & 1 ? 72 : -72))),
      workY: homeY
    };
  }

  function applyRoster(roster) {
    var seen = {};
    (roster || []).forEach(function (r) {
      seen[r.sessionId] = true;
      var v = byId[r.sessionId];
      if (v) {
        v.name = r.name;
        return;
      }
      v = spawn(r);
      byId[r.sessionId] = v;
      list.push(v);
    });
    for (var i = list.length - 1; i >= 0; i--) {
      if (!seen[list[i].id]) {
        delete byId[list[i].id];
        list.splice(i, 1);
      }
    }
  }

  function onActivity(ev) {
    var v = byId[ev.sessionId];
    if (!v) return;
    v.targetX = v.workX;
    v.targetY = v.workY;
    v.work = true;
    v.state = "walking";
  }

  function step(given) {
    var now = Date.now();
    var dt;
    if (given == null) {
      dt = last ? Math.min(0.1, (now - last) / 1000) : 0;
      last = now;
    } else {
      dt = given;
    }
    for (var i = 0; i < list.length; i++) {
      var v = list[i];
      if (v.state !== "walking") continue;
      var dx = v.targetX - v.x;
      var dy = v.targetY - v.y;
      var d = Math.sqrt(dx * dx + dy * dy);
      var move = SPEED * dt;
      if (d <= move) {
        v.x = v.targetX;
        v.y = v.targetY;
        v.state = v.work ? "working" : "idle";
      } else {
        v.x += (dx / d) * move;
        v.y += (dy / d) * move;
      }
    }
  }

  function tag(ctx, P, v, cx, cy) {
    var w = pillW(v.name);
    ctx.fillStyle = P.paper;
    ctx.beginPath();
    ctx.roundRect(cx - w / 2, cy - 5, w, 10, 5);
    ctx.fill();
    ctx.fillStyle = P.ink;
    ctx.fillText(v.name, cx, cy + 1);
  }

  function billboard(ctx, P, v) {
    var x = Math.round(v.x);
    var y = Math.round(v.y);
    var s = v.seed;
    ctx.fillStyle = P.shadow;
    ctx.fillRect(x - 5, y - 2, 11, 3);
    ctx.fillStyle = P[BODY[s % BODY.length]];
    ctx.fillRect(x - 4, y - 11, 8, 9);
    ctx.fillStyle = P[HAT[(s >>> 3) % HAT.length]];
    ctx.fillRect(x - 5, y - 19, 10, 3);
    ctx.fillStyle = P.wall;
    ctx.fillRect(x - 5, y - 16, 10, 5);
    ctx.fillStyle = P.ink;
    ctx.fillRect(x - 3, y - 15, 1, 2);
    ctx.fillRect(x + 2, y - 15, 1, 2);
    tag(ctx, P, v, x, y - 26);
  }

  function draw(ctx, hour, dt) {
    step(dt);
    var P = g.town.palette(hour);
    ctx.font = FONT;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    var order = list.slice().sort(function (a, b) { return a.y - b.y; });
    for (var i = 0; i < order.length; i++) billboard(ctx, P, order[i]);
  }

  // Front-most villager wins when boxes overlap.
  function pick(px, py) {
    var order = list.slice().sort(function (a, b) { return b.y - a.y; });
    for (var i = 0; i < order.length; i++) {
      var v = order[i];
      var x = Math.round(v.x), y = Math.round(v.y);
      var half = Math.max(5, Math.ceil(pillW(v.name) / 2));
      if (Math.abs(px - x) <= half && py >= y - 32 && py <= y + 2) return v.id;
    }
    return null;
  }

  g.villagers = {
    applyRoster: applyRoster,
    onActivity: onActivity,
    pick: pick,
    draw: draw,
    get: function (id) { return byId[id]; }
  };
})(globalThis);
