(function (g) {
  var NOTES = { talk: 440, work: 330, select: 550 };
  var ctx = null, muted = false, scheduled = 0;

  // Only called from a user gesture, so the browser never auto-plays.
  function unlock() {
    if (!ctx) {
      var AC = g.AudioContext || g.webkitAudioContext;
      if (AC) ctx = new AC();
    }
    if (ctx && ctx.state === "suspended") ctx.resume();
  }

  function setMuted(v) {
    muted = !!v;
  }

  function isMuted() {
    return muted;
  }

  function blip(kind) {
    if (muted || !ctx) return;
    var t = ctx.currentTime;
    var osc = ctx.createOscillator();
    var amp = ctx.createGain();
    osc.type = "square";
    osc.frequency.value = NOTES[kind] || NOTES.talk;
    amp.gain.setValueAtTime(0.0001, t);
    amp.gain.exponentialRampToValueAtTime(0.08, t + 0.005);
    amp.gain.exponentialRampToValueAtTime(0.0001, t + 0.09);
    osc.connect(amp).connect(ctx.destination);
    osc.start(t);
    osc.stop(t + 0.1);
    scheduled++;
  }

  g.audio = {
    unlock: unlock,
    setMuted: setMuted,
    isMuted: isMuted,
    blip: blip,
    unlocked: function () { return !!ctx; },
    scheduled: function () { return scheduled; }
  };
})(globalThis);
