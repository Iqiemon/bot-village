(function (g) {
  var BODY = ["#c25b4a", "#4f8f4a", "#5c6470", "#966f46", "#5a6fa8", "#a85a8a"];

  function village() {
    var list = [];
    var byId = {};

    function colour(seed) {
      return BODY[seed % BODY.length];
    }

    function spawn(v, i) {
      var col = i % 4, row = (i / 4) | 0;
      var vv = {
        id: v.sessionId,
        name: v.name,
        seed: v.seed,
        x: 60 + col * 112 + (seedHash(v.sessionId) % 9),
        y: 150 + row * 40 + (seedHash(v.sessionId) >>> 4) % 5,
        state: "idle"
      };
      list.push(vv);
      byId[vv.id] = vv;
    }

    function seedHash(s) {
      var h = 0;
      for (var i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
      return h;
    }

    function applyRoster(vs) {
      var seen = {};
      (vs || []).forEach(function (v, i) {
        seen[v.sessionId] = true;
        if (!byId[v.sessionId]) spawn(v, i);
      });
      list = list.filter(function (v) {
        if (seen[v.id]) return true;
        delete byId[v.id];
        return false;
      });
    }

    function onActivity(ev) {
      var v = byId[ev.sessionId];
      if (!v) return;
      v.state = "working";
      v.until = performance.now() + 1500;
    }

    function update(now) {
      list.forEach(function (v) {
        if (v.state === "working" && now > v.until) v.state = "idle";
      });
    }

    function draw(ctx, now) {
      update(now);
      list.slice().sort(function (a, b) { return a.y - b.y; }).forEach(function (v) {
        var bob = v.state === "working" ? ((now / 120 | 0) % 2) : ((now / 400 | 0) % 2);
        ctx.fillStyle = "rgba(0,0,0,0.25)";
        ctx.fillRect(v.x - 5, v.y + 2, 12, 3);
        ctx.fillStyle = colour(v.seed);
        ctx.fillRect(v.x - 4, v.y - 10 - bob, 9, 11);
        ctx.fillStyle = "#f6f0dd";
        ctx.fillRect(v.x - 2, v.y - 8 - bob, 2, 2);
        ctx.fillRect(v.x + 2, v.y - 8 - bob, 2, 2);
        ctx.fillStyle = "#2b2430";
        ctx.font = "7px monospace";
        ctx.textAlign = "center";
        ctx.fillText(v.name, v.x, v.y - 14 - bob);
        if (v.state === "working") {
          ctx.fillStyle = "#c25b4a";
          ctx.fillRect(v.x + 6, v.y - 12 - bob, 2, 3);
        }
      });
    }

    return {
      applyRoster: applyRoster,
      onActivity: onActivity,
      draw: draw
    };
  }

  g.villagers = { create: village };
})(globalThis);
