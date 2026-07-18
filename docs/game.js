/* Suburble — daily guess-the-Cape-Town-suburb. Vanilla JS, no build step. */
(function () {
  "use strict";

  var MAX_GUESSES = 6;
  var MAX_KM = 80; // metro span used for proximity scaling
  var EPOCH = "2026-07-17"; // launch day = puzzle #1
  var TZ = "Africa/Johannesburg";

  var data = null;    // suburbs.json
  var mode = "normal"; // "normal" (curated answers) | "hard" (all suburbs)
  var state = { day: 0, guesses: [], done: false, won: false };
  var answer = null;

  // --- daily selection ---------------------------------------------------

  function sastToday() {
    return new Intl.DateTimeFormat("en-CA", { timeZone: TZ }).format(new Date());
  }

  function dayNumber() {
    var ms = new Date(sastToday() + "T00:00:00Z") - new Date(EPOCH + "T00:00:00Z");
    return Math.max(0, Math.round(ms / 86400000));
  }

  // mulberry32 — deterministic shuffle shared by every visitor
  function mulberry32(a) {
    return function () {
      a |= 0; a = (a + 0x6D2B79F5) | 0;
      var t = Math.imul(a ^ (a >>> 15), 1 | a);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  function shuffled(n, seed) {
    var rnd = mulberry32(seed);
    var idx = Array.from({ length: n }, function (_, i) { return i; });
    for (var i = n - 1; i > 0; i--) {
      var j = Math.floor(rnd() * (i + 1));
      var t = idx[i]; idx[i] = idx[j]; idx[j] = t;
    }
    return idx;
  }

  // answerPool returns the suburb indices eligible as answers for a mode.
  function answerPool(m) {
    var pool = [];
    data.suburbs.forEach(function (s, i) {
      if (s.name.indexOf("CAPE FARMS") === 0) return; // guessable, never the answer
      if (m === "normal" && !s.known) return;
      pool.push(i);
    });
    return pool;
  }

  function todaysAnswer(m, day) {
    var pool = answerPool(m);
    var order = shuffled(pool.length, m === "hard" ? 0x5b3b1e : 0x50f7);
    return pool[order[day % pool.length]];
  }

  // --- geometry ----------------------------------------------------------

  function haversineKm(a, b) {
    var R = 6371, rad = Math.PI / 180;
    var la1 = a[1] * rad, la2 = b[1] * rad;
    var dla = la2 - la1, dlo = (b[0] - a[0]) * rad;
    var h = Math.sin(dla / 2) ** 2 + Math.cos(la1) * Math.cos(la2) * Math.sin(dlo / 2) ** 2;
    return 2 * R * Math.asin(Math.sqrt(h));
  }

  function bearingDeg(a, b) {
    var rad = Math.PI / 180;
    var la1 = a[1] * rad, la2 = b[1] * rad, dlo = (b[0] - a[0]) * rad;
    var y = Math.sin(dlo) * Math.cos(la2);
    var x = Math.cos(la1) * Math.sin(la2) - Math.sin(la1) * Math.cos(la2) * Math.cos(dlo);
    return ((Math.atan2(y, x) * 180 / Math.PI) + 360) % 360;
  }

  var ARROWS = ["⬆️", "↗️", "➡️", "↘️", "⬇️", "↙️", "⬅️", "↖️"];
  function arrow(deg) { return ARROWS[Math.round(deg / 45) % 8]; }

  // --- display helpers ---------------------------------------------------

  var SMALL_WORDS = { DE: "de", DER: "der", VAN: "van", DIE: "die" };
  function displayName(raw) {
    return raw.split(" ").map(function (w, i) {
      if (i > 0 && SMALL_WORDS[w]) return SMALL_WORDS[w];
      return w.charAt(0).toUpperCase() + w.slice(1).toLowerCase();
    }).join(" ");
  }

  function proximitySquares(km) {
    var prox = Math.max(0, 1 - km / MAX_KM);
    var fifths = prox * 5;
    var s = "";
    for (var i = 0; i < 5; i++) {
      if (fifths >= i + 1) s += "🟩";
      else if (fifths >= i + 0.5) s += "🟨";
      else s += "⬛";
    }
    return s;
  }

  // --- storage (all keys are per-mode) ------------------------------------

  function stateKey() { return "suburble-state-" + mode; }
  function statsKey() { return "suburble-stats-" + mode; }

  function loadState(day) {
    try {
      var s = JSON.parse(localStorage.getItem(stateKey()));
      if (s && s.day === day) return s;
    } catch (e) { /* fresh */ }
    return { day: day, guesses: [], done: false, won: false };
  }

  function saveState() {
    try { localStorage.setItem(stateKey(), JSON.stringify(state)); } catch (e) { /* private mode */ }
  }

  function loadStats() {
    try { return JSON.parse(localStorage.getItem(statsKey())) || {}; } catch (e) { return {}; }
  }

  function recordResult(won, guesses) {
    var st = loadStats();
    st.played = (st.played || 0) + 1;
    st.won = (st.won || 0) + (won ? 1 : 0);
    st.dist = st.dist || {};
    if (won) st.dist[guesses] = (st.dist[guesses] || 0) + 1;
    if (won && st.lastWinDay === state.day - 1) st.streak = (st.streak || 0) + 1;
    else st.streak = won ? 1 : 0;
    if (won) st.lastWinDay = state.day;
    st.maxStreak = Math.max(st.maxStreak || 0, st.streak || 0);
    try { localStorage.setItem(statsKey(), JSON.stringify(st)); } catch (e) { /* ok */ }
  }

  // --- dom ---------------------------------------------------------------

  function $(id) { return document.getElementById(id); }

  function renderSilhouette(sub, solved) {
    var svg = $("shape");
    svg.innerHTML = "";
    var path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    var d = sub.rings.map(function (ring) {
      return "M" + ring.map(function (p) { return p[0] + " " + p[1]; }).join("L") + "Z";
    }).join("");
    path.setAttribute("d", d);
    path.setAttribute("class", solved ? "shape solved" : "shape");
    svg.appendChild(path);
  }

  function renderGuesses() {
    var box = $("guesses");
    box.innerHTML = "";
    for (var i = 0; i < MAX_GUESSES; i++) {
      var row = document.createElement("div");
      row.className = "guess-row";
      var g = state.guesses[i];
      if (g) {
        var sub = data.suburbs[g.idx];
        var correct = g.idx === state.answerIdx;
        row.innerHTML =
          '<span class="g-name' + (correct ? " g-win" : "") + '">' + displayName(sub.name) + "</span>" +
          '<span class="g-dist">' + (correct ? "🎉" : g.km.toFixed(1) + " km") + "</span>" +
          '<span class="g-dir">' + (correct ? "" : g.arrow) + "</span>" +
          '<span class="g-prox">' + proximitySquares(g.km) + "</span>";
      } else {
        row.innerHTML = '<span class="g-empty">·</span>';
      }
      box.appendChild(row);
    }
    renderHint();
  }

  // renderHint reveals a little more after the 3rd and 4th misses.
  function renderHint() {
    var el = $("hint");
    var misses = state.guesses.filter(function (g) { return g.idx !== state.answerIdx; }).length;
    if (state.done || misses < 3) { el.hidden = true; return; }
    var hints = ["area: " + answer.km2 + " km²"];
    if (misses >= 4) hints.push("starts with “" + displayName(answer.name).charAt(0) + "”");
    el.hidden = false;
    el.textContent = "hint — " + hints.join(" · ");
  }

  function showResult(verdict) {
    $("guess-form").hidden = true;
    renderSilhouette(answer, true);
    $("result").hidden = false;
    $("result-name").textContent = displayName(answer.name);
    $("result-meta").textContent = answer.km2 + " km² · puzzle #" + (state.day + 1) + (mode === "hard" ? " · hard" : "");
    $("result-verdict").textContent = verdict;
    renderStats();
  }

  function finish(won) {
    state.done = true;
    state.won = won;
    saveState();
    recordResult(won, state.guesses.length);
    renderHint();
    showResult(won
      ? ["Legend!", "Lekker!", "Sharp sharp!", "Nice one!", "Got there!", "Phew!"][state.guesses.length - 1]
      : "Next time!");
  }

  function renderStats() {
    var st = loadStats();
    $("stats").hidden = false;
    $("stat-played").textContent = st.played || 0;
    $("stat-winpct").textContent = st.played ? Math.round(100 * (st.won || 0) / st.played) + "%" : "–";
    $("stat-streak").textContent = st.streak || 0;
    $("stat-max").textContent = st.maxStreak || 0;
  }

  function shareText() {
    var tag = mode === "hard" ? " (hard)" : "";
    var lines = ["Cape Town Suburble #" + (state.day + 1) + tag + " " + (state.won ? state.guesses.length : "X") + "/" + MAX_GUESSES];
    state.guesses.forEach(function (g) {
      lines.push(proximitySquares(g.km) + (g.idx === state.answerIdx ? "🎯" : g.arrow));
    });
    lines.push("https://richardwooding.github.io/suburble/");
    return lines.join("\n");
  }

  // --- autocomplete ------------------------------------------------------

  // ALIASES maps official dataset names to the spellings people actually
  // use, so typing "Bloubergstrand" finds BLAAUWBERGSTRAND. Matching only;
  // the official name is always what's displayed.
  var ALIASES = {
    "BLAAUWBERGSTRAND": ["BLOUBERGSTRAND", "BLOUBERG STRAND"],
    "CAMPS BAY / BAKOVEN": ["CAMPS BAY", "BAKOVEN"],
    "GUGULETU": ["GUGULETHU"],
    "EERSTERIVIER": ["EERSTE RIVER"],
    "MELKBOSCH STRAND": ["MELKBOSSTRAND"],
    "GORDONS BAY": ["GORDON'S BAY"],
    "SIMON'S TOWN": ["SIMONS TOWN", "SIMONSTOWN"],
    "BO-KAAP": ["SCHOTSCHE KLOOF", "BOKAAP"],
    "MITCHELLS PLAIN CBD": ["MITCHELLS PLAIN"],
    "KUILSRIVIER SOUTH SMALLHOLDINGS": ["KUILS RIVER"],
    "ST JAMES": ["SAINT JAMES"]
  };

  function wireInput() {
    var input = $("guess-input");
    var list = $("suggestions");
    var names = data.suburbs.map(function (s, i) {
      return { i: i, d: displayName(s.name), u: s.name, a: ALIASES[s.name] || [] };
    });
    var active = -1, shown = [];

    function hide() { list.hidden = true; active = -1; }

    function show(q) {
      var needle = q.trim().toUpperCase();
      if (!needle) { hide(); return; }
      var guessed = {};
      state.guesses.forEach(function (g) { guessed[g.idx] = true; });
      var matches = function (n) {
        if (n.u.indexOf(needle) >= 0) return true;
        return n.a.some(function (al) { return al.indexOf(needle) >= 0; });
      };
      var starts = function (n) {
        if (n.u.indexOf(needle) === 0) return true;
        return n.a.some(function (al) { return al.indexOf(needle) === 0; });
      };
      shown = names.filter(function (n) { return !guessed[n.i] && matches(n); }).slice(0, 8);
      shown.sort(function (a, b) {
        var as = starts(a) ? 0 : 1, bs = starts(b) ? 0 : 1;
        return as - bs || a.u.localeCompare(b.u);
      });
      list.innerHTML = "";
      shown.forEach(function (n) {
        var li = document.createElement("li");
        li.textContent = n.d;
        li.setAttribute("role", "option");
        li.addEventListener("mousedown", function (e) { e.preventDefault(); pick(n.i); });
        list.appendChild(li);
      });
      list.hidden = shown.length === 0;
      active = -1;
    }

    function highlight() {
      Array.prototype.forEach.call(list.children, function (li, i) {
        li.className = i === active ? "active" : "";
      });
    }

    function pick(idx) {
      input.value = "";
      hide();
      guess(idx);
    }

    input.addEventListener("input", function () { show(input.value); });
    input.addEventListener("keydown", function (e) {
      if (list.hidden) return;
      if (e.key === "ArrowDown") { active = Math.min(active + 1, shown.length - 1); highlight(); e.preventDefault(); }
      else if (e.key === "ArrowUp") { active = Math.max(active - 1, 0); highlight(); e.preventDefault(); }
      else if (e.key === "Enter") {
        e.preventDefault();
        if (active >= 0) pick(shown[active].i);
        else if (shown.length > 0) pick(shown[0].i);
      } else if (e.key === "Escape") hide();
    });
    input.addEventListener("blur", function () { setTimeout(hide, 150); });
  }

  function guess(idx) {
    if (state.done || state.guesses.length >= MAX_GUESSES) return;
    var sub = data.suburbs[idx];
    var km = haversineKm(sub.c, answer.c);
    var g = { idx: idx, km: km, arrow: arrow(bearingDeg(sub.c, answer.c)) };
    state.guesses.push(g);
    saveState();
    renderGuesses();
    if (idx === state.answerIdx) finish(true);
    else if (state.guesses.length >= MAX_GUESSES) finish(false);
  }

  // --- mode + boot ---------------------------------------------------------

  function setMode(m) {
    mode = m;
    try { localStorage.setItem("suburble-mode", m); } catch (e) { /* ok */ }
    $("mode-normal").setAttribute("aria-selected", String(m === "normal"));
    $("mode-hard").setAttribute("aria-selected", String(m === "hard"));
    startDay();
  }

  function startDay() {
    var day = dayNumber();
    state = loadState(day);
    state.answerIdx = todaysAnswer(mode, day);
    answer = data.suburbs[state.answerIdx];

    $("day-no").textContent = "#" + (day + 1);
    $("result").hidden = true;
    $("stats").hidden = true;
    $("guess-form").hidden = false;
    renderSilhouette(answer, state.done);
    renderGuesses();
    if (state.done) showResult(state.won ? "Solved!" : "Next time!");
  }

  fetch("data/suburbs.json").then(function (r) { return r.json(); }).then(function (d) {
    data = d;
    try { mode = localStorage.getItem("suburble-mode") === "hard" ? "hard" : "normal"; } catch (e) { /* default */ }
    wireInput();
    $("mode-normal").addEventListener("click", function () { setMode("normal"); });
    $("mode-hard").addEventListener("click", function () { setMode("hard"); });
    $("share").addEventListener("click", function () {
      navigator.clipboard.writeText(shareText()).then(function () {
        $("share").textContent = "copied!";
        setTimeout(function () { $("share").textContent = "share"; }, 1500);
      });
    });
    setMode(mode);
  }).catch(function (err) {
    $("guesses").textContent = "could not load suburb data — " + err;
  });
})();
