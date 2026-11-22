(function () {
  var canvas = document.getElementById("stage");
  if (!canvas) return;
  var town = globalThis.town.create(canvas);

  function fit() {
    var room = Math.min(innerWidth / town.W, innerHeight / town.H);
    // shortcut: below 1x the ground is resampled, not sharp; upgrade if sub-1x becomes a target.
    var scale = room >= 1 ? Math.floor(room) : room;
    canvas.style.width = town.W * scale + "px";
    canvas.style.height = town.H * scale + "px";
  }

  addEventListener("resize", fit);
  fit();

  // First gesture anywhere unlocks audio once; later clicks just blip.
  addEventListener("pointerdown", function () {
    globalThis.audio.unlock();
    globalThis.audio.blip("select");
  });

  addEventListener("keydown", function (e) {
    if (e.key === "m" || e.key === "M") {
      globalThis.audio.setMuted(!globalThis.audio.isMuted());
      document.querySelector(".hint").textContent = globalThis.audio.isMuted()
        ? "Muted, press M to unmute"
        : "Press M to mute";
    }
  });

  function frame(t) {
    town.draw(t);
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);
})();
