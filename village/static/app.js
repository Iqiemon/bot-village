(function () {
  var canvas = document.getElementById("stage");
  if (!canvas) return;
  var town = globalThis.town.create(canvas);

  function fit() {
    var scale = Math.floor(Math.min(innerWidth / town.W, innerHeight / town.H));
    if (scale < 1) scale = 1;
    canvas.style.width = town.W * scale + "px";
    canvas.style.height = town.H * scale + "px";
  }

  addEventListener("resize", fit);
  fit();

  function frame(t) {
    town.draw(t);
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);
})();
