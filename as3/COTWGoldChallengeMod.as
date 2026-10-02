package {
	import flash.display.DisplayObject;
	import flash.display.DisplayObjectContainer;
	import flash.display.Graphics;
	import flash.display.MovieClip;
	import flash.display.Shape;
	import flash.filters.DropShadowFilter;
	import flash.filters.GlowFilter;
	import flash.geom.ColorTransform;
	import flash.geom.Point;
	import flash.geom.Rectangle;
	import flash.display.Sprite;
	import flash.net.URLLoader;
	import flash.net.URLRequest;
	import flash.text.TextField;
	import flash.text.TextFormat;
	import flash.utils.ByteArray;
	import flash.utils.Endian;
	import flash.utils.getTimer;

	// In-game challenge: species wall in the HUD (hud.gfx) and best medal already harvested for the
	// species aimed at with the binoculars (clue_hud.gfx). Both movies run their own copy of this class.
	public class COTWGoldChallengeMod {
		public static const LOG_NAME:String = "COTWGoldChallengeLog";
		public static const GAUGE_NAME:String = "COTWGoldChallenge";
		private static const SAVE_URL:String = "/cotwgc_saves/hunting_log_adf";
		private static const SPECIES_URL:String = "cotwgc_species.txt";
		private static const SCORING_URL:String = "cotwgc_scoring.txt";
		private static const SETTINGS_URL:String = "cotwgc_settings.txt";
		private static const RESERVES_URL:String = "cotwgc_reserves.txt";
		private static const REGIONS_URL:String = "cotwgc_regions.txt";
		private static const HISTORY_URL:String = "cotwgc_history.txt";
		private static const SAME_TRIP_S:int = 4 * 3600; // harvests this close in time are in the same reserve
		private static const WORLD_URL:String = "/cotwgc_saves/reserveworlddata_adf";
		public static const HUD_NAME:String = "COTWGoldChallengeHud";
		private static const RELOAD_MS:int = 2000;
		// cotwgc_toggle.txt, written by COTWGoldChallenge.exe while its shortcut runs:
		// "<beat> <shown 0|1> <names held 0|1> <opaque held 0|1>". The beat changes every second; when it stops, the
		// program is closed and the overlay shows again.
		private static const TOGGLE_URL:String = "cotwgc_toggle.txt";
		private static const TOGGLE_MS:int = 100;
		private static const TOGGLE_STALE_MS:int = 3000;
		private static const COLORS:Array = [0x8FE3FF, 0xFFC42E, 0xE2E2E2, 0xC87A35]; // rank 0 diamond .. 3 bronze

		private static var settingsDone:Boolean;
		private static var settingsText:String;
		private static var settingsVersion:int; // changes with each settings read, for the redraw keys
		private static var speciesDone:Boolean;
		private static var scoringDone:Boolean;
		private static var scoring:Object = {}; // icon -> {w, S, G, D, d: [dist]}
		private static var estKey:String;
		private static var estValue:Object;
		private static var estInfo:String = ""; // debug: inputs and result of the last estimate
		private static var pending:Object = {};
		private static var ticking:Boolean;
		private static var reservesDone:Boolean;
		private static var regionsDone:Boolean;
		private static var regionReserve:Object = {}; // region hash -> reserve number
		private static var worldLocked:Boolean; // the HUD told the reserve: stop reading it from the save
		private static var reserves:Object = {}; // number -> {name, hashes}
		private static var reserve:int = -1;
		private static var hud:*;
		private static var hudKey:String;
		private static var hashByIcon:Object = {};
		private static var aliasOf:Object = {}; // other hash used by the hunting log -> species hash
		private static var iconByHash:Object = {};
		private static var keyByHash:Object = {};
		private static var clsByHash:Object = {};
		private static var nameByHash:Object = {};
		private static const TINTS:Array = [0x9EE9FF, 0xFFC42E, 0xF2F2F2, 0xE69240]; // icon colours, rank 0 .. 3
		private static var settings:Object = {};
		private static var since:Number = 0;
		private static var nextLoad:int = -RELOAD_MS;
		private static var saveLength:int = -1;
		private static var saveTail:Number = -1;
		private static var version:int;
		private static var entries:Array = [];
		private static var bySpecies:Object = {}; // species hash -> its entries
		private static var historyText:String;
		private static var logRaw:ByteArray;
		private static var placeAgain:Boolean; // entries to read and place again
		private static var status:String = "";
		private static var shownKey:String;
		private static var placed:Object = {}; // hash -> icon of the HUD wall, see record()
		private static var recording:Boolean;
		private static var doneBefore:Object; // species validated in the previous HUD wall
		private static var doneRules:String;
		private static var doneVersion:int = -1;
		private static var fests:Array = []; // celebrations: {hash, m, t0, look, fx}
		CONFIG::preview {
			private static var testFest:uint;
			private static var testNext:int = 5000;
			private static var testM:int = 1;
		}
		public static const MENU_NAME:String = "COTWGoldChallengeMenu";
		private static var menuPanel:*;
		private static var menuKey:String;
		private static var menuIndex:int;
		private static var wallOff:Boolean; // hidden with the shortcut
		private static var namesOn:Boolean; // names key held: missing species listed with their names
		private static var opaqueOn:Boolean; // opacity key held: species outside the reserve fully opaque
		private static var nextToggle:int;
		private static var toggleBeat:String;
		private static var toggleSeen:int;
		private static var toggleLoader:URLLoader;
		private static var toggleStarted:int;
		private static var wallChanged:Boolean;
		private static var shownPanel:*; // last binoculars panel and species, redrawn when the wall toggles
		private static var shownIcon:int;
		// hud.gfx and clue_hud.gfx are separate movies, each with its own copy of this class and no way
		// to talk. The clue copy builds the same wall and keeps only the aimed species, drawn over the HUD.
		private static const HUD_WIDTH:Number = 1280; // stage of hud.gfx, stretched to the screen as ours
		private static const HUD_HEIGHT:Number = 720;
		private static var overlay:Sprite;
		private static var overlayKey:String;
		private static var cluePanel:*;
		private static var heardClip:*; // audio clue shown, called once: retried until the data is loaded
		private static var heardIcon:int;
		private static var grow:Object; // aimed icon growing: {ic, cx, cy, from, to, f, at, out}

		public static function update(panel:*, data:*):void {
			CONFIG::preview {
				// Outside the game SetData runs once: repeat it.
				if (!ticking) {
					ticking = true;
					hudCreated(panel.root);
					panel.addEventListener("enterFrame", function(e:*):void {
						update(panel, data);
					});
				}
			}
			try {
				tick();
				CONFIG::preview {
					selfTest();
				}
				show(panel, int(data.animal_id));
			} catch (e:Error) {
				CONFIG::preview {
					trace("COTWGC " + e + " " + status);
				}
				status = "error: " + e.message;
				shownKey = null;
				try {
					show(panel, int(data.animal_id));
				} catch (e2:Error) {
					CONFIG::preview {
						trace("COTWGC2 " + e2);
					}
				}
			}
		}

		// clue_hud.gfx, audio clue panel: an animal is heard, its species is shown in the wall too.
		public static function heard(panel:*, data:*):void {
			try {
				tick();
				heardClip = panel.MCI_audioClue || panel;
				heardIcon = int(data.animal_id);
				aimOverlay(heardClip, uint(hashByIcon[heardIcon]));
			} catch (e:Error) {
			}
		}

		// cotwgc_scoring.txt: icon|weight based|silver|gold|diamond|gender,great one,wmin,wmax,smin,smax,dev;...
		private static function readScoring(text:String):void {
			for each (var line:String in text.split("\n")) {
				var f:Array = trim(line).split("|");
				if (f.length < 6) {
					continue;
				}
				var ds:Array = [];
				for each (var t:String in String(f[5]).split(";")) {
					var v:Array = t.split(",");
					if (v.length >= 7) {
						ds.push({g: int(v[0]), go: int(v[1]), wmin: Number(v[2]), wmax: Number(v[3]), smin: Number(v[4]), smax: Number(v[5]), dev: Number(v[6])});
					}
				}
				scoring[int(f[0])] = {w: f[1] == "1", S: Number(f[2]), G: Number(f[3]), D: Number(f[4]), d: ds};
			}
		}

		// Trophy estimate of the binoculars panel, from the rules of the game (research notes):
		// - the score range is a window of width 2 * 0.1 * R around the true score (R: score range of the
		//   gender, Great Ones included), at a random position; its bounds are truncated to integers;
		// - the weight range is the quarter of the species weight range holding the animal, truncated too;
		// - for weight trophies, score = smin + (weight fraction +- dev) * R.
		// Returns {pS, pG, pD, lo, hi} or null (then the panel keeps the simple estimate).
		// Name of the skill the gauge needs (Spotting Knowledge, level 3), in the game language.
		public static function skillName(panel:*):String {
			try {
				var s:* = panel.root.GetLocalizedStringFromString("skill_02_name_02");
				if (s && String(s).length > 0 && String(s).indexOf("LOC:") != 0 && String(s) != "skill_02_name_02") {
					return String(s) + " 3";
				}
			} catch (e:Error) {
			}
			return "Spotting Knowledge 3";
		}

		public static function estimate(data:*):Object {
			try {
				if (!scoringDone || !data.animal_score_visible) {
					return null;
				}
				var sp:Object = scoring[int(data.animal_id)];
				if (!sp) {
					return null;
				}
				var m:int = int(data.animal_score_min);
				var M:int = int(data.animal_score_max);
				var wv:Boolean = Boolean(data.animal_weight_visible);
				var wm:int = int(data.animal_weight_min);
				var wM:int = int(data.animal_weight_max);
				var gs:String = String(data.animal_gender);
				var key:String = data.animal_id + "/" + gs + "/" + m + "/" + M + "/" + wv + "/" + wm + "/" + wM;
				if (key == estKey) {
					return estValue;
				}
				estKey = key;
				var genders:Array = gs == "Male" ? [0] : gs == "Female" ? [1] : [0, 1];
				var r:Object = null;
				// with everything, then without the weight, then without the window width
				for (var level:int = 0; level < 3 && !r; level++) {
					r = posterior(sp, genders, m, M, wv && level == 0, wm, wM, level < 2);
				}
				estValue = r;
				estInfo = gs + " " + m + "-" + M + (wv ? " " + wm + "-" + wM : "") + " > " + (r ? Math.round(r.pS * 100) + "/" + Math.round(r.pG * 100) + "/" + Math.round(r.pD * 100) + "% " + int(r.lo * 10) / 10 + "-" + int(r.hi * 10) / 10 + " (" + (level - 1) + ")" : "none");
				return r;
			} catch (e:Error) {
			}
			return null;
		}

		CONFIG::preview {
			// Outside the game: compares the estimator with the reference model (cotwgc_esttest.txt).
			private static var selfTested:Boolean;
			private static function selfTest():void {
				if (selfTested || !scoringDone) {
					return;
				}
				var text:* = fetch("cotwgc_esttest.txt", false, !pending["cotwgc_esttest.txt"]);
				if (text == null) {
					return;
				}
				selfTested = true;
				var n:int = 0, bad:int = 0, worst:Number = 0;
				for each (var line:String in String(text).split("\n")) {
					var f:Array = line.split("|");
					if (f.length < 7) {
						continue;
					}
					estKey = null;
					var r:Object = estimate({animal_id: f[0], animal_gender: f[1], animal_score_min: f[2], animal_score_max: f[3],
						animal_weight_min: f[4], animal_weight_max: f[5], animal_score_visible: true, animal_weight_visible: true});
					var ex:Array = String(f[6]).split(",");
					n++;
					var err:Number = r ? Math.max(Math.abs(r.pS - ex[0]), Math.abs(r.pG - ex[1]), Math.abs(r.pD - ex[2])) : 1;
					var flip:Boolean = !r || (r.pG == 0) != (Number(ex[1]) == 0) || (r.pG == 1) != (Number(ex[1]) == 1) || (r.pS == 0) != (Number(ex[0]) == 0) || (r.pD == 0) != (Number(ex[2]) == 0);
					if (err > 0.03 || flip) {
						bad++;
						if (bad < 8) {
							trace("COTWGC EST mismatch " + line + " -> " + (r ? r.pS + "," + r.pG + "," + r.pD : "null"));
						}
					}
					worst = Math.max(worst, err);
				}
				trace("COTWGC EST selftest " + n + " cases, " + bad + " mismatches, worst " + worst);
			}
		}

		private static function dist(sp:Object, g:int, go:int):Object {
			for each (var d:Object in sp.d) {
				if (d.g == g && d.go == go) {
					return d;
				}
			}
			return null;
		}

		// Score intervals (weighted uniform pieces) of the prior, for one gender.
		private static function prior(sp:Object, g:int, useWeight:Boolean, wm:int, wM:int):Array {
			var n:Object = dist(sp, g, 0);
			if (!n || n.smax <= 0) {
				return [];
			}
			var R:Number = n.smax - n.smin;
			var dv:Number = n.dev * R;
			var out:Array = [];
			if (sp.w && useWeight && n.wmax > n.wmin) {
				var o:Object = dist(sp, g, 1);
				// species weight range, without then with the Great Ones; gender ranges likewise
				var u:Array = unions(sp);
				var cw:Array = [[n.wmin, n.wmax]];
				if (o && o.wmax > 0) {
					cw.push([Math.min(n.wmin, o.wmin), Math.max(n.wmax, o.wmax)]);
				}
				for (var conv:int = 0; conv < 2 && out.length == 0; conv++) {
					var k2:Number = conv == 0 ? 1 : 2.20462; // kg, then pounds
					for each (var U:Array in u) {
						var ur:Number = U[1] - U[0];
						for (var k:int = 0; k < 4; k++) {
							var a:Number = U[0] + k * 0.25 * ur;
							var b:Number = U[0] + (k + 1) * 0.25 * ur;
							if (!floors(a * k2, wm) || !floors(b * k2, wM)) {
								continue;
							}
							addWeight(out, n, Math.max(a, n.wmin), Math.min(b, n.wmax), R, dv);
							for each (var c:Array in cw) {
								addWeight(out, n, c[0] + k * 0.25 * (c[1] - c[0]), c[0] + (k + 1) * 0.25 * (c[1] - c[0]), R, dv);
							}
						}
					}
				}
			}
			if (out.length == 0) {
				out.push([n.smin - dv, n.smax + dv]);
			}
			return out;
		}

		// The game computes in single precision: at an integer boundary, either integer is possible.
		private static function floors(x:Number, n:int):Boolean {
			return Math.floor(x - 1e-4) == n || Math.floor(x + 1e-4) == n;
		}

		private static function addWeight(out:Array, n:Object, lo:Number, hi:Number, R:Number, dv:Number):void {
			lo = Math.max(lo, n.wmin);
			hi = Math.min(hi, n.wmax);
			if (lo > hi) {
				return;
			}
			var wr:Number = n.wmax - n.wmin;
			out.push([n.smin + (lo - n.wmin) / wr * R - dv, n.smin + (hi - n.wmin) / wr * R + dv]);
		}

		private static function unions(sp:Object):Array {
			var a:Number = Infinity, b:Number = -Infinity, ag:Number = Infinity, bg:Number = -Infinity;
			for each (var d:Object in sp.d) {
				if (d.wmax <= 0) {
					continue;
				}
				ag = Math.min(ag, d.wmin);
				bg = Math.max(bg, d.wmax);
				if (d.go == 0) {
					a = Math.min(a, d.wmin);
					b = Math.max(b, d.wmax);
				}
			}
			var out:Array = [[a, b]];
			if (ag < a || bg > b) {
				out.push([ag, bg]);
			}
			return out;
		}

		// Score ranges R of a gender: regular animals, and with the Great Ones.
		private static function ranges(sp:Object, g:int):Array {
			var n:Object = dist(sp, g, 0);
			var o:Object = dist(sp, g, 1);
			var out:Array = [n.smax - n.smin];
			if (o && o.smax > 0) {
				out.push(Math.max(n.smax, o.smax) - Math.min(n.smin, o.smin));
			}
			return out;
		}

		private static function posterior(sp:Object, genders:Array, m:int, M:int, useWeight:Boolean, wm:int, wM:int, knownWidth:Boolean):Object {
			var pieces:Array = []; // [lo, hi, weight, gender]
			var g:int;
			for each (g in genders) {
				var pr:Array = prior(sp, g, useWeight, wm, wM);
				for each (var q:Array in pr) {
					if (q[1] > q[0]) {
						pieces.push([q[0], q[1], 1 / (pr.length * (q[1] - q[0])), g]);
					}
				}
			}
			if (pieces.length == 0) {
				return null;
			}
			var lo:Number = Infinity, hi:Number = -Infinity;
			for each (q in pieces) {
				lo = Math.min(lo, q[0]);
				hi = Math.max(hi, q[1]);
			}
			lo = Math.max(lo, m - 0.001);
			hi = Math.min(hi, M + 1.001);
			if (hi <= lo) {
				return null;
			}
			var N:int = 400;
			var step:Number = (hi - lo) / N;
			var post:Array = [];
			var i:int;
			for (i = 0; i < N; i++) {
				post.push(0);
			}
			for each (g in genders) {
				var Ws:Array = [];
				if (knownWidth) {
					for each (var R:Number in ranges(sp, g)) {
						var w:Number = 0.2 * R;
						if (w > M - m - 1 && w < M - m + 1) {
							Ws.push(w);
						}
					}
				}
				if (Ws.length == 0) { // unknown width: every width the display allows
					for (i = 0; i < 8; i++) {
						w = M - m - 1 + (i + 0.5) * 0.25;
						if (w > 0) {
							Ws.push(w);
						}
					}
				}
				// weights and scores follow a bell curve over their range (mean 0.5, sd 0.15 in the saves)
				var nd:Object = dist(sp, g, 0);
				var s0:Number = nd ? nd.smin : 0;
				var sr:Number = nd && nd.smax > nd.smin ? nd.smax - nd.smin : 1;
				for each (w in Ws) {
					var L1:Number = Math.max(m, M - w);
					var L2:Number = Math.min(m + 1, M + 1 - w);
					if (L2 <= L1) {
						continue;
					}
					for (i = 0; i < N; i++) {
						var x:Number = lo + (i + 0.5) * step;
						var d:Number = 0;
						for each (q in pieces) {
							if (q[3] == g && x >= q[0] && x <= q[1]) {
								d += q[2];
							}
						}
						if (d > 0) {
							var sf:Number = ((x - s0) / sr - 0.5) / 0.15;
							post[i] += d * Math.exp(-0.5 * sf * sf) * Math.max(0, Math.min(x, L2) - Math.max(x - w, L1)) / w / Ws.length;
						}
					}
				}
			}
			var z:Number = 0, ps:Number = 0, pg:Number = 0, pd:Number = 0, a:Number = Infinity, b:Number = -Infinity;
			for (i = 0; i < N; i++) {
				var v:Number = post[i];
				if (v <= 0) {
					continue;
				}
				x = lo + (i + 0.5) * step;
				z += v;
				if (x >= sp.S) {
					ps += v;
				}
				if (x >= sp.G) {
					pg += v;
				}
				if (x >= sp.D) {
					pd += v;
				}
				a = Math.min(a, x - step / 2);
				b = Math.max(b, x + step / 2);
			}
			if (z <= 0) {
				return null;
			}
			return {pS: ps / z, pG: pg / z, pD: pd / z, lo: a, hi: b};
		}

		private static function readSettings(text:String):void {
			for each (var line:String in text.split("\n")) {
				var eq:int = line.indexOf("=");
				if (eq > 0 && trim(line).charAt(0) != "#") {
					settings[trim(line.substr(0, eq))] = trim(line.substr(eq + 1));
				}
			}
			// since=YYYY-MM-DD or YYYY-MM-DD HH:MM, local time
			var dt:Array = trim(String(settings.since || "")).split(" ");
			var m:Array = String(dt[0]).split("-");
			var hm:Array = dt.length > 1 ? String(dt[1]).split(":") : [0, 0];
			if (m.length == 3 && int(m[0]) > 2000) {
				since = new Date(int(m[0]), int(m[1]) - 1, int(m[2]), int(hm[0]), int(hm[1])).time / 1000;
			}
		}

		// cotwgc_species.txt: icon|hash|class|name key|translated name|other hashes used by the hunting log
		private static function readSpecies(text:String):void {
			for each (var line:String in text.split("\n")) {
				var f:Array = trim(line).split("|");
				if (f.length < 5) {
					continue;
				}
				var h:uint = uint(f[1]);
				hashByIcon[int(f[0])] = h;
				iconByHash[h] = int(f[0]);
				clsByHash[h] = int(f[2]);
				keyByHash[h] = f[3];
				nameByHash[h] = f[4];
				for each (var a:String in String(f[5] || "").split(",")) {
					if (a) {
						aliasOf[uint(a)] = h;
					}
				}
			}
		}

		// Starts loading every RELOAD_MS; in between, only collects loads still in progress.
		private static function tick():void {
			var now:int = getTimer();
			var start:Boolean = now >= nextLoad;
			if (start) {
				nextLoad = now + RELOAD_MS;
			}
			reload(start);
		}

		// Returns the file content, or null while it loads or when start is false and nothing is pending.
		// Loads usually complete synchronously; the first load of a file may take a few frames.
		private static function fetch(url:String, binary:Boolean, start:Boolean):* {
			var l:URLLoader = pending[url];
			if (l && l.data != null) {
				delete pending[url];
				return l.data;
			}
			if (!start) {
				return null;
			}
			l = new URLLoader();
			if (binary) {
				l.dataFormat = "binary";
			}
			pending[url] = l;
			l.load(new URLRequest(url));
			if (l.data != null) {
				delete pending[url];
				return l.data;
			}
			return null;
		}

		private static function reload(start:Boolean):void {
			// settings: read again every RELOAD_MS, a change on the settings page shows at once
			var text:* = fetch(SETTINGS_URL, false, start);
			if (text != null && String(text) != settingsText) {
				settingsText = String(text);
				settings = {};
				since = 0;
				readSettings(settingsText);
				settingsDone = true;
				settingsVersion++;
				shownKey = null;
			}
			if (!speciesDone) {
				if (!scoringDone) {
					var st:* = fetch(SCORING_URL, false, start);
					if (st != null) {
						readScoring(String(st));
						scoringDone = true;
					}
				}
				text = fetch(SPECIES_URL, false, start);
				if (text != null) {
					readSpecies(String(text));
					speciesDone = true;
					shownKey = null;
				}
			}
			if (!reservesDone) {
				text = fetch(RESERVES_URL, false, start);
				if (text != null) {
					readReserves(String(text));
					reservesDone = true;
					placeAgain = true; // reserves help to place harvests
				}
			}
			if (!regionsDone) {
				text = fetch(REGIONS_URL, false, start);
				if (text != null) {
					for each (var rl:String in String(text).split("\n")) {
						var rf:Array = trim(rl).split(" ");
						if (rf.length == 2) {
							regionReserve[uint(rf[0])] = int(rf[1]);
						}
					}
					regionsDone = true;
					placeAgain = true; // place again with the regions
				}
			}
			if (!worldLocked) {
				var world:ByteArray = fetch(WORLD_URL, true, start) as ByteArray;
				if (world && world.length > 40) {
					try {
						reserve = readReserve(world);
					} catch (e:Error) {
					}
				}
			}
			// history kept by the program: read before the save, which may be missing
			text = fetch(HISTORY_URL, false, start);
			if (text != null && String(text) != historyText) {
				historyText = String(text);
				placeAgain = true;
			}
			var raw:ByteArray = fetch(SAVE_URL, true, start) as ByteArray;
			if (raw && raw.length >= 40) {
				raw.endian = Endian.LITTLE_ENDIAN;
				raw.position = raw.length - 8;
				var tail:Number = raw.readUnsignedInt() * 4294967296 + raw.readUnsignedInt();
				if (raw.length != saveLength || tail != saveTail) {
					saveLength = raw.length;
					saveTail = tail;
					logRaw = raw;
					placeAgain = true;
				}
			} else if (saveLength < 0 && start && !historyText) {
				status = "no save (" + SAVE_URL + ")";
			}
			if (!placeAgain || !speciesDone) {
				return;
			}
			placeAgain = false;
			var log:Array = [];
			var logError:String = null;
			try {
				log = logRaw ? parse(logRaw) : [];
			} catch (pe:Error) {
				logError = pe.message; // the history still counts
			}
			entries = merge(log, historyText ? readHistory(historyText) : []);
			assignReserves(entries);
			bySpecies = {};
			for each (var he:Array in entries) {
				if (!bySpecies[he[0]]) {
					bySpecies[he[0]] = [];
				}
				bySpecies[he[0]].push(he);
			}
			version++;
			status = logError ? "log error: " + logError : entries.length + " entries, " + log.length + " in the log";
		}

		// cotwgc_history.txt, written by the program: "h species score rank time region" for each
		// harvest seen in the hunting log, "m species rank time reserve" for a trophy added by hand.
		private static function readHistory(text:String):Array {
			var out:Array = [];
			for each (var line:String in text.split("\n")) {
				var f:Array = trim(line).split(" ");
				var e:Array;
				if (f.length == 6 && f[0] == "h") {
					e = harvest(uint(f[1]), Number(f[2]), uint(f[3]), uint(f[4]), uint(f[5]));
				} else if (f.length == 5 && f[0] == "m") {
					e = harvest(uint(f[1]), 0, uint(f[2]), uint(f[3]), 0);
					e[5] = int(f[4]);
					e[7] = true;
				} else {
					continue;
				}
				out.push(e);
			}
			return out;
		}

		// Harvests of the log and of the history, without the ones seen twice.
		private static function merge(log:Array, hist:Array):Array {
			var seen:Object = {};
			var out:Array = [];
			for each (var e:Array in log.concat(hist)) {
				if (!e[7]) {
					if (seen[e[8]]) {
						continue;
					}
					seen[e[8]] = true;
				}
				out.push(e);
			}
			return out;
		}

		private static function readReserves(text:String):void {
			for each (var line:String in text.split("\n")) {
				var f:Array = trim(line).split("|");
				if (f.length >= 3) {
					var hashes:Array = [];
					for each (var h:String in String(f[2]).split(",")) {
						if (h) {
							hashes.push(uint(h));
						}
					}
					reserves[int(f[0])] = {name: f[1], hashes: hashes};
				}
			}
		}

		// Save file: "SAVE" header and zlib data, or plain ADF. Returns the ADF with the offset of its root instance.
		private static function adf(raw:ByteArray):Array {
			var d:ByteArray = raw;
			if (raw[0] == 0x53 && raw[1] == 0x41 && raw[2] == 0x56 && raw[3] == 0x45) {
				d = new ByteArray();
				d.writeBytes(raw, 32);
				d.uncompress();
			}
			d.endian = Endian.LITTLE_ENDIAN;
			var b:int = -1;
			for (var i:int = 0; i + 4 <= d.length; i++) {
				if (d[i] == 0x20 && d[i + 1] == 0x46 && d[i + 2] == 0x44 && d[i + 3] == 0x41) {
					b = i;
					break;
				}
			}
			if (b < 0) {
				throw new Error("ADF not found");
			}
			var inst:int = b + u32(d, b + 12);
			return [d, b + u32(d, inst + 8)];
		}

		// reserveworlddata_adf: root {AlreadyConverted, Reserve}.
		private static function readReserve(raw:ByteArray):int {
			var a:Array = adf(raw);
			return int(u32(a[0], a[1] + 4));
		}

		// hunting_log_adf: "SAVE" header, zlib data, then an ADF file.
		// Returns [speciesHash, score, rank, timestamp, regionHash] records.
		private static function parse(raw:ByteArray):Array {
			var r:Array = adf(raw);
			var d:ByteArray = r[0];
			var root:int = r[1];
			var i:int;
			var out:Array = [];
			var off:int = root + u32(d, root + 8);
			var n:int = u32(d, root + 16);
			for (i = 0; i < n; i++) {
				out.push(entry(d, off + i * 24));
			}
			off = root + u32(d, root + 24);
			n = u32(d, root + 32);
			for (i = 0; i < n; i++) {
				var o:int = off + i * 24;
				var po:int = root + u32(d, o + 8);
				var pn:int = u32(d, o + 16);
				for (var j:int = 0; j < pn; j++) {
					out.push(entry(d, po + j * 24));
				}
			}
			return out;
		}

		private static function u32(d:ByteArray, p:int):uint {
			d.position = p;
			return d.readUnsignedInt();
		}

		private static function entry(d:ByteArray, p:int):Array {
			d.position = p;
			var sp:uint = d.readUnsignedInt();
			var score:Number = d.readFloat();
			var rank:uint = d.readUnsignedInt();
			d.readUnsignedInt();
			return harvest(sp, score, rank, d.readUnsignedInt(), d.readUnsignedInt());
		}

		// [species hash, score, rank, time, region hash, reserve, Great One, added by hand, key].
		// Rank 0 diamond .. 3 bronze, 4 none; any other rank (Great One) counts as a diamond.
		// The key (species, time, raw rank) finds a harvest both in the log and in the history.
		private static function harvest(sp:uint, score:Number, rank:uint, ts:uint, reg:uint):Array {
			if (aliasOf[sp]) {
				sp = aliasOf[sp];
			}
			var out:Array = [sp, score, rank > 4 ? 0 : rank, ts, reg, -1, rank > 4, false];
			out[8] = sp + "/" + ts + "/" + rank;
			return out;
		}

		// Best rank (0 diamond .. 3 bronze, 4 none) and number of logged harvests of a species, since a date.
		// Reserve of each harvest (entry[5]): from its region; otherwise the only reserve of the species;
		// otherwise the reserve of the closest harvest in time, if within SAME_TRIP_S.
		private static function assignReserves(list:Array):void {
			var only:Object = {};
			for (var n:String in reserves) {
				for each (var h:uint in reserves[n].hashes) {
					only[h] = only[h] == null ? int(n) : -1;
				}
			}
			var known:Array = [];
			for each (var e:Array in list) {
				if (e[7]) {
					continue; // added by hand: its reserve is set, and it tells nothing about a trip
				}
				var r:* = regionReserve[e[4]];
				if (r == null && only[e[0]] != null && only[e[0]] >= 0) {
					r = only[e[0]];
				}
				e[5] = r == null ? -1 : int(r);
				if (e[5] >= 0) {
					known.push(e);
				}
			}
			for each (e in list) {
				if (e[5] >= 0) {
					continue;
				}
				var bestGap:Number = SAME_TRIP_S;
				for each (var k:Array in known) {
					var gap:Number = Math.abs(k[3] - e[3]);
					if (gap < bestGap) {
						bestGap = gap;
						e[5] = k[5];
					}
				}
			}
		}

		// Reserve used for completion: -1 when any reserve counts (100% mode).
		private static function scopeReserve():int {
			return settings.scope != "mega" ? currentReserve() : -1; // reserve mode by default
		}

		private static function currentReserve():int {
			return settings.reserve != null ? int(settings.reserve) : reserve;
		}

		private static function best(hash:uint, from:Number, inReserve:int = -1):Array {
			var rank:int = 4;
			var go:Boolean = false;
			var seen:Object = {};
			var count:int = 0;
			for each (var e:Array in bySpecies[hash] || []) {
				if (e[3] < from || (inReserve >= 0 && e[5] != inReserve)) {
					continue;
				}
				var key:String = e[3] + ":" + e[1];
				if (!seen[key]) {
					seen[key] = true;
					count++;
				}
				if (e[2] < rank) {
					rank = e[2];
				}
				if (e[6]) {
					go = true;
				}
			}
			return [rank, count, go];
		}

		// main_menu.gfx and change_reserve.gfx, reserve selection: called each time a reserve is shown.
		// The challenge progress of that reserve is drawn beside it.
		public static function reserveShown(panel:*, index:*):void {
			try {
				menuIndex = int(index);
				if (menuPanel != panel) {
					menuPanel = panel;
					panel.addEventListener("enterFrame", menuFrame);
				}
				menuKey = null;
				menuFrame(null);
			} catch (e:Error) {
			}
		}

		private static function menuFrame(e:*):void {
			try {
				tick();
				var p:* = menuPanel;
				var index:int = menuIndex;
				var item:* = p.m_ReserveData ? p.m_ReserveData[index] : null;
				var id:String = item && item.m_ReserveName != null ? String(item.m_ReserveName) : "";
				var rn:int = menuReserve(p, id, index);
				var key:String = rn + "/" + id + "/" + version + "/" + status + "/" + reservesDone + "/" + settingsVersion + "/" + speciesDone;
				if (key == menuKey) {
					return;
				}
				menuKey = key;
				var old:* = p.getChildByName(MENU_NAME);
				if (old) {
					p.removeChild(old);
				}
				// the condensed font of the reserve name, as the HUD panels
				var src:TextField = null;
				try {
					src = p.MCI_ReserveItem.MCI_Reserves.TXT_ReserveName as TextField;
				} catch (e1:Error) {
				}
				if (!src) {
					src = findTextField(p);
				}
				if (!src || !speciesDone || !reservesDone || !settingsDone) {
					return;
				}
				var built:Array = rn >= 0 ? buildReserve(src, rn, settings.scope != "mega" ? rn : -1, 0, "grid", false) : null;
				var box:Sprite = built ? built[0] : new Sprite();
				box.name = MENU_NAME;
				if (settings.debug == "1") {
					addText(src, box, "menu: " + index + " " + id + " -> " + rn + " · " + status, 0xFFFF66, 12, 0, box.height + 2, false, 0);
				}
				if (!box.numChildren) {
					return;
				}
				// left of the reserve card, as large as the HUD wall if it fits
				var st:* = p.stage;
				var card:* = p.MCI_ReserveItem;
				if (!st || !card) {
					return;
				}
				var c:Rectangle = card.getBounds(st);
				var m:* = p.transform.concatenatedMatrix;
				var w:Number = box.getBounds(box).right;
				var k:Number = Math.min(st.stageWidth / HUD_WIDTH, (c.left - 64) / w);
				box.scaleX = k / m.a;
				box.scaleY = k / m.d;
				var at:Point = p.globalToLocal(new Point(c.left - 24 - w * k, c.top));
				box.x = at.x;
				box.y = at.y;
				p.addChild(box);
			} catch (err:Error) {
				status = "menu error: " + err.message;
			}
		}

		// Reserve number of the reserve shown: its name matches the reserve file, else its position.
		private static function menuReserve(p:*, id:String, index:int):int {
			var name:String = id;
			try {
				var s:* = p.root.GetLocalizedStringFromString(id);
				if (s && String(s).indexOf("LOC:") != 0) {
					name = String(s);
				}
			} catch (e:Error) {
			}
			name = fold(trim(name));
			if (name) {
				for (var n:String in reserves) {
					if (fold(trim(reserves[n].name)) == name) {
						return int(n);
					}
				}
			}
			return reserves[index] ? index : -1;
		}

		// hud.gfx: called at the end of the hud constructor.
		public static function hudCreated(h:*):void {
			try {
				hud = h;
				h.addEventListener("enterFrame", hudFrame);
			} catch (e:Error) {
			}
		}

		// hud.gfx: called when the player enters a region, with the reserve number.
		public static function regionEntered(h:*, number:*):void {
			try {
				if (!hud) {
					hudCreated(h);
				}
				reserve = int(number);
				worldLocked = true;
			} catch (e:Error) {
			}
		}

		private static function hudFrame(e:*):void {
			try {
				tick();
				pollToggle();
				showHud();
				festFrame();
			} catch (err:Error) {
				status = "error: " + err.message;
			}
		}

		// Reads the shortcut state every TOGGLE_MS.
		private static function pollToggle():void {
			var now:int = getTimer();
			if (toggleLoader && toggleLoader.data != null) {
				var f:Array = trim(String(toggleLoader.data)).split(" ");
				CONFIG::preview {
					trace("COTWGC toggle " + f.join("/") + " off " + wallOff);
				}
				toggleLoader = null;
				if (f.length >= 2 && f[0] != toggleBeat) {
					toggleBeat = f[0];
					toggleSeen = now;
					if (wallOff != (f[1] == "0")) {
						wallOff = !wallOff;
						wallChanged = true;
					}
					if (namesOn != (f[2] == "1")) {
						namesOn = !namesOn;
						wallChanged = true;
					}
					if (opaqueOn != (f[3] == "1")) {
						opaqueOn = !opaqueOn;
						wallChanged = true;
					}
				}
			}
			if ((wallOff || namesOn || opaqueOn) && now - toggleSeen > TOGGLE_STALE_MS) {
				wallOff = false;
				namesOn = false;
				opaqueOn = false;
				wallChanged = true;
			}
			if (now < nextToggle || (toggleLoader && now < toggleStarted + TOGGLE_STALE_MS)) {
				return; // a load still running is given time
			}
			nextToggle = now + TOGGLE_MS;
			toggleStarted = now;
			var l:URLLoader = new URLLoader();
			l.addEventListener("ioError", function(e:*):void {
				if (toggleLoader == e.target) {
					toggleLoader = null;
				}
			});
			toggleLoader = l;
			l.load(new URLRequest(TOGGLE_URL));
			if (l.data != null) {
				pollToggle();
			}
		}

		// Species wall of the current reserve, in the HUD. Settings:
		// wall=grid|list|0, wallX, wallY, iconSize, perRow, classes=0|1, sort=reserve|class, missing=0|1.
		private static function showHud():void {
			var key:String = reserve + "/" + version + "/" + status + "/" + reservesDone + "/" + settingsVersion + "/" + speciesDone + "/" + wallOff + "/" + namesOn + "/" + opaqueOn;
			var root:* = hud;
			var old:Sprite = root.getChildByName(HUD_NAME) as Sprite;
			if (old && key == hudKey) {
				return;
			}
			hudKey = key;
			if (old) {
				root.removeChild(old);
			}
			placed = {};
			recording = true;
			var built:Array = null;
			try {
				if (namesOn) {
					built = speciesDone ? buildReserve(findTextField(root), currentReserve(), scopeReserve(), 0, "list", true, true) : null;
				} else {
					built = wallOff ? null : buildWall(findTextField(root), 0);
				}
			} finally {
				recording = false;
			}
			if (!built) {
				return;
			}
			root.addChild(built[0]);
			// Better medals since the last wall of the same challenge and reserve: celebrated.
			var rules:String = settings.tier + "/" + settings.higher + "/" + settings.scope + "/" + since + "/" + currentReserve();
			var now:int = getTimer();
			var n:int = 0;
			// every better medal is celebrated, more or less festively (see festLook)
			if (doneBefore && rules == doneRules && version != doneVersion) {
				for (var h:String in placed) {
					var q:Object = placed[h];
					var was:Object = doneBefore[h];
					if (q.rank < 4 && (!was || q.rank < was.rank || (q.go && !was.go))) {
						fests.push({hash: uint(h), m: q.go ? -1 : q.rank, t0: now + 450 * n++});
					}
				}
			}
			CONFIG::preview {
				if (!doneBefore) {
					for (var th:String in placed) {
						if (placed[th].ok) {
							testFest = uint(th);
							break;
						}
					}
				}
			}
			doneBefore = {};
			for (h in placed) {
				doneBefore[h] = {rank: placed[h].rank, go: placed[h].go};
			}
			doneRules = rules;
			doneVersion = version;
		}

		// HUD wall icons: position and validation of each species, for the celebrations.
		private static function record(sp:Object, ic:DisplayObject, x:Number, y:Number, w:Number, h:Number):void {
			if (recording) {
				placed[sp.hash] = {ic: ic, ok: sp.ok, cx: x + w / 2, cy: y + h / 2, w: w, sx: ic.scaleX, x: ic.x, y: ic.y,
					alpha: ic.alpha, filters: ic.filters, rank: sp.rank, go: sp.go};
			}
		}

		// A better medal: the icon pops with a flash and hexagon shockwaves, then settles back
		// with a slight bounce. Small for bronze and silver, larger for gold, festive for diamonds and
		// Great Ones. Drawn from the time elapsed only, so a wall rebuilt meanwhile picks the animation
		// up where it was.
		private static const GO_COLORS:Array = [0xFFC42E, 0x9EE9FF, 0xFFFFFF, 0xD6A8FF];

		// m: -1 Great One, 0 diamond, 1 gold, 2 silver, 3 bronze.
		private static function festLook(m:int):Object {
			var W:uint = 0xFFFFFF;
			if (m < 0) {
				var gc:uint = 0xFFC42E;
				return {dur: 3600, peak: 2.6, glow: 3000, radius: 3.6,
					color: gc,
					rings: [[60, gc, 7], [200, 0x9EE9FF, 3.5], [360, W, 3], [560, 0xD6A8FF, 2.5], [800, gc, 2], [1050, 0x9EE9FF, 1.5]]};
			}
			if (m == 0) {
				var dc:uint = 0x9EE9FF;
				return {dur: 3200, peak: 2.4, glow: 2600, radius: 3.2,
					color: dc,
					rings: [[60, dc, 6], [220, W, 3], [380, dc, 2], [700, W, 2], [900, dc, 1.5]]};
			}
			if (m == 1) {
				var oc:uint = 0xFFC42E;
				return {dur: 2400, peak: 2.1, glow: 1800, radius: 2.6,
					color: oc, rings: [[60, oc, 5], [240, W, 2.5], [420, oc, 1.5]]};
			}
			var c:uint = TINTS[m == 2 ? 2 : 3];
			return {dur: 1700, peak: 1.55, glow: 900, radius: 1.9,
				color: c, rings: [[40, c, 3], [200, W, 1.5]]};
		}

		private static function festFrame():void {
			var now:int = getTimer();
			CONFIG::preview {
				if (testFest && now > testNext) {
					testNext = now + 4000;
					fests.push({hash: testFest, m: testM, t0: now});
					testM = testM == -1 ? 3 : testM - 1;
				}
			}
			for (var i:int = fests.length - 1; i >= 0; i--) {
				var f:Object = fests[i];
				var L:Object = f.look || (f.look = festLook(f.m));
				var p:Object = placed[f.hash];
				var e:Number = now - f.t0;
				if (!p || e >= L.dur) {
					if (p) {
						festIcon(p, 1, 0, false);
					}
					if (f.fx && f.fx.parent) {
						f.fx.parent.removeChild(f.fx);
					}
					fests.splice(i, 1);
					continue;
				}
				if (e < 0) {
					continue;
				}
				var body:DisplayObjectContainer = p.ic.parent;
				if (!body) {
					continue;
				}
				if (!f.fx || f.fx.parent != body) {
					if (f.fx && f.fx.parent) {
						f.fx.parent.removeChild(f.fx);
					}
					f.fx = new Sprite();
					f.fx.mouseEnabled = false;
				}
				body.addChild(p.ic); // above its neighbours
				body.addChild(f.fx);
				// icon: up to the peak in 220 ms, back to 1 with an overshoot by 850 ms
				var up:Number = L.peak - 1;
				var k:Number = 1;
				if (e < 220) {
					k = 1 + up * easeOut(e / 220);
				} else if (e < 850) {
					var t:Number = (e - 220) / 630;
					var c:Number = 2.2;
					k = L.peak - up * (1 + (c + 1) * Math.pow(t - 1, 3) + c * Math.pow(t - 1, 2));
				}
				var glowColor:uint = f.m < 0 ? GO_COLORS[int(e / 180) % 2] : L.color;
				festIcon(p, k, Math.max(0, 1 - e / 380), e < L.glow, glowColor, e);
				// shockwaves
				var g:Graphics = f.fx.graphics;
				g.clear();
				for each (var rg:Array in L.rings) {
					ring(g, p, (e - rg[0]) / 750, rg[1], rg[2], L.radius);
				}
			}
		}

		private static function festIcon(p:Object, k:Number, flash:Number, glow:Boolean, color:uint = 0, e:Number = 0):void {
			var ic:DisplayObject = p.ic;
			ic.scaleX = ic.scaleY = p.sx * k;
			ic.x = p.cx - (p.cx - p.x) * k;
			ic.y = p.cy - (p.cy - p.y) * k;
			var o:Number = 255 * flash;
			ic.transform.colorTransform = new ColorTransform(1, 1, 1, 1, o, o, o, 0);
			if (glow) {
				var pulse:Number = 0.6 + 0.4 * Math.cos(e / 140);
				ic.filters = [new GlowFilter(color, 0.9, 10 + 8 * pulse, 10 + 8 * pulse, 2 + pulse), shadow()];
				ic.alpha = 1;
			} else {
				ic.filters = p.filters;
				ic.alpha = p.alpha;
			}
		}

		// Hexagon ring growing from the icon and fading, for u from 0 to 1.
		private static function ring(g:Graphics, p:Object, u:Number, color:uint, width:Number, grow:Number):void {
			if (u <= 0 || u >= 1) {
				return;
			}
			var r:Number = p.w * 0.62 * (1 + grow * easeOut(u));
			g.lineStyle(Math.max(0.5, width * (1 - u)), color, 1 - u);
			for (var i:int = 0; i <= 6; i++) {
				var a:Number = Math.PI / 2 + i * Math.PI / 3;
				var px:Number = p.cx + Math.cos(a) * r;
				var py:Number = p.cy + Math.sin(a) * r * 1.05;
				if (i == 0) {
					g.moveTo(px, py);
				} else {
					g.lineTo(px, py);
				}
			}
			g.lineStyle();
		}

		private static function easeOut(t:Number):Number {
			return 1 - Math.pow(1 - t, 3);
		}

		// The wall of the settings, with the aimed species highlighted: [wall, its icon or null], or null.
		private static function buildWall(src:TextField, aim:uint):Array {
			var mode:String = settings.wall || "grid";
			if (!src || mode == "0" || !speciesDone) {
				return null;
			}
			if (scopeReserve() < 0 && settings.megaWall != "0" && reservesDone) {
				return buildMega(src, aim, null);
			}
			if (mode == "list") {
				return buildReserve(src, currentReserve(), scopeReserve(), aim, mode, true); // reserve= forces one (tests)
			}
			// the species of the reserve played, in the same layout
			var r:Object = reserves[currentReserve()];
			return r ? buildMega(src, aim, r) : null;
		}

		// Wall of one reserve, with the harvests made anywhere or only in that reserve (inReserve >= 0).
		// onlyMissing: the missing species only, whatever the settings (names key held).
		private static function buildReserve(src:TextField, number:int, inReserve:int, aim:uint, mode:String, onHud:Boolean, onlyMissing:Boolean = false):Array {
			var r:Object = reserves[number];
			if (!r) {
				return null;
			}
			var tiers:Array = String(settings.tiers || "DIAMOND,GOLD,SILVER,BRONZE").split(",");
			var target:int = settings.tier != null ? int(settings.tier) : 1;
			var higher:Boolean = settings.higher != "0";
			// laid out at 44 and scaled to iconSize: texts and banner keep their proportions
			var size:Number = 44;
			var perRow:int = int(settings.perRow || 10);
			var classes:Boolean = settings.classes == "1";
			var missing:Boolean = onlyMissing || settings.missing == "1";

			var list:Array = [];
			var done:int = 0;
			for (var i:int = 0; i < r.hashes.length; i++) {
				var h:uint = r.hashes[i];
				var bst:Array = best(h, since, inReserve);
				var rank:int = bst[0];
				var ok:Boolean = rank == target || (higher && rank < target);
				if (ok) {
					done++;
				}
				if (!(ok && missing)) {
					list.push({hash: h, rank: rank, go: bst[2], ok: ok, cls: clsByHash[h], order: i});
				}
			}
			if (settings.sort == "class" || classes) {
				list.sortOn(["cls", "order"], Array.NUMERIC);
			}

			var wall:Sprite = new Sprite();
			wall.name = HUD_NAME;
			wall.mouseEnabled = false;
			wall.mouseChildren = false;
			var right:String = tiers[target] + " " + done + "/" + r.hashes.length;
			if (missing && list.length) {
				right += "  ·  " + (settings.leftText || "left") + " " + list.length;
			}
			if (settings.total == "1") {
				// overall progress: every species, harvested in any reserve
				var seen:Object = {};
				var all:int = 0;
				var allDone:int = 0;
				for (var rn:String in reserves) {
					for each (var ah:uint in reserves[rn].hashes) {
						if (seen[ah]) {
							continue;
						}
						seen[ah] = true;
						all++;
						var ar:int = best(ah, since)[0];
						if (ar == target || (higher && ar < target)) {
							allDone++;
						}
					}
				}
				right += "  ·  " + (settings.totalText || "GLOBAL") + " " + allDone + "/" + all;
			}
			var iconClass:Class = null;
			try {
				iconClass = COTWGoldChallengeIcons;
			} catch (e:Error) {
			}
			// Icons first, to know the width.
			var body:Sprite = new Sprite();
			var y:Number = 8;
			var x:Number = 10;
			var maxX:Number = 0;
			var col:int = 0;
			var lastCls:int = -1;
			var names:Array = [];
			var aimIcon:Sprite = null;
			for each (var sp:Object in list) {
				if (mode == "list") {
					addIcon(iconClass, body, sp, 10, y, 30);
					var n:TextField = addText(src, body, speciesName(sp.hash), 0xFFFFFF, 17, 50, y + 5, false, 0);
					names.push([sp, y]);
					maxX = Math.max(maxX, n.x + n.width + 90);
					y += 36;
					continue;
				}
				if (col == perRow) {
					col = 0;
					x = 10;
					y += size * 1.112 + 6;
				}
				if (classes && sp.cls != lastCls) {
					var t:TextField = addText(src, body, String(sp.cls), 0xE8ECF2, 15, x + 2, y + size * 0.3, false, 0);
					x += t.textWidth + 8;
					lastCls = sp.cls;
				}
				if (sp.hash == aim) {
					aimIcon = padded(iconClass, body, sp, x, y, size, size * 1.112, 0.45);
					aimLook(aimIcon, x, y, size, size * 1.112);
				} else {
					record(sp, padded(iconClass, body, sp, x, y, size, size * 1.112, 0.45), x, y, size, size * 1.112);
				}
				x += size + 6;
				maxX = Math.max(maxX, x);
				col++;
			}
			if (mode != "list" && list.length) {
				y += size * 1.112 + 8;
			}
			var probe:TextField = addText(src, body, String(r.name).toUpperCase() + "    " + right, 0xFFFFFF, 20, 0, 0, false, 0);
			body.removeChild(probe);
			var width:Number = Math.max(mode == "list" ? 320 : 300, maxX + 4, probe.width + 20);
			for each (var nm:Array in names) {
				var rk:int = nm[0].rank;
				addText(src, body, rk < 4 ? tiers[rk] : "–", rk < 4 ? COLORS[rk] : 0x888888, 16, width - 10, nm[1] + 6, true, 0);
			}
			if (settings.debug == "1") {
				addText(src, body, status + " · reserve " + reserve, 0xAAAAAA, 12, 10, y, false, 0);
				y += 18;
			}
			// Banner and body, as the game panels.
			wall.graphics.beginFill(0xFF9800, 1);
			wall.graphics.drawRect(0, 0, width, 34);
			wall.graphics.endFill();
			wall.graphics.beginFill(0x000000, 0.55);
			wall.graphics.drawRect(0, 34, width, y);
			wall.graphics.endFill();
			addText(src, wall, String(r.name).toUpperCase(), 0xFFFFFF, 20, 10, 4, false, 0);
			addText(src, wall, right, 0xFFFFFF, 20, width - 10, 4, true, 0);
			body.y = 34;
			if (aimIcon) {
				body.addChild(aimIcon); // on top of its neighbours
			}
			wall.addChild(body);
			shadowTexts(wall);
			if (onHud) {
				wall.scaleX = wall.scaleY = Number(settings.iconSize || 28) / 44;
				place(wall);
			}
			return [wall, aimIcon];
		}

		// Overlay with the completion badge: every species by class (100% mode, the species of the reserve
		// played highlighted), or only those of a reserve (only). Settings: megaSize, megaGap, megaRows,
		// reserveSize, perRow, attenue, avant=0|1, classes=0|1, gaugeSize, iconsAlign, bg, wallX, wallY.
		private static function buildMega(src:TextField, aim:uint, only:Object):Array {
			var target:int = settings.tier != null ? int(settings.tier) : 1;
			var higher:Boolean = settings.higher != "0";
			var size:Number = Number(settings.megaSize || 20);
			var gap:Number = Number(settings.megaGap || 3);
			var clsSize:Number = Number(settings.classSize || 9);
			if (only) {
				// reserve mode: fewer species (19 at most), larger icons; the gauge keeps its size
				var rk:Number = Number(settings.reserveSize || 30) / Math.max(1, size);
				size *= rk;
				gap *= rk;
				clsSize *= rk;
			}
			var rows:int = int(settings.megaRows || 5);
			var attenue:Number = opaqueOn ? 1 : settings.attenue != null ? Number(settings.attenue) : 0.55;
			var showCls:Boolean = settings.classes != "0";
			var here:Object = {};
			var cur:Object = reserves[currentReserve()];
			var avant:Boolean = settings.avant != "0" && cur != null && !only;
			if (cur) {
				for each (var ch:uint in cur.hashes) {
					here[ch] = true;
				}
			}
			var list:Array = [];
			var seen:Object = {};
			var done:int = 0;
			var total:int = 0;
			var aimIcon:Sprite = null;
			var inReserve:int = only ? scopeReserve() : -1;
			for (var rn:String in reserves) {
				if (only && reserves[rn] != only) {
					continue;
				}
				for each (var h:uint in reserves[rn].hashes) {
					if (seen[h]) {
						continue;
					}
					seen[h] = true;
					total++;
					var b:Array = best(h, since, inReserve);
					var rank:int = b[0];
					var ok:Boolean = rank == target || (higher && rank < target);
					if (ok) {
						done++;
					}
					if (!(only && ok && settings.missing == "1")) {
						list.push({hash: h, rank: rank, go: b[2], ok: ok, cls: int(clsByHash[h]) || 99, name: fold(speciesName(h))});
					}
				}
			}
			list.sortOn(["cls", "name"], [Array.NUMERIC, 0]);

			var wall:Sprite = new Sprite();
			wall.name = HUD_NAME;
			wall.mouseEnabled = false;
			wall.mouseChildren = false;
			var gsize:Number = Number(settings.gaugeSize || 52);
			var pct:int = total ? Math.round(100 * done / total) : 0;
			// Gauge with the reserve name under it, in a column of fixed width (a long name is shrunk):
			// the layout must not depend on the reserve, which the binoculars movie may know later.
			var rname:TextField = null;
			var colW:Number = gsize * 1.2;
			var gcol:Sprite = new Sprite(); // badge column: gauge, reserve name, overall progress
			wall.addChild(gcol);
			if (cur) {
				rname = addText(src, gcol, String(cur.name).toUpperCase(), 0xFFFFFF, Number(settings.nameSize || 6), 0, 0, false, 0);
				rname.filters = [shadow()];
				if (rname.width > colW) {
					rname.scaleX = rname.scaleY = colW / rname.width;
				}
			}
			drawGauge(gcol, src, 6 + (colW - gsize) / 2, 6, gsize, pct, done + "/" + total);
			var gy:Number = 6 + gsize * 1.112 + 2;
			if (rname) {
				rname.x = 6 + (colW - rname.width) / 2;
				rname.y = gy;
				gy += rname.height;
			}
			if (only && settings.total == "1") {
				// overall progress under the name: every species, harvested in any reserve
				var aseen:Object = {};
				var all:int = 0;
				var allDone:int = 0;
				for (var an:String in reserves) {
					for each (var ah:uint in reserves[an].hashes) {
						if (!aseen[ah]) {
							aseen[ah] = true;
							all++;
							var ar:int = best(ah, since)[0];
							if (ar == target || (higher && ar < target)) {
								allDone++;
							}
						}
					}
				}
				var gt:TextField = addText(src, gcol, (settings.totalText || "GLOBAL") + " " + allDone + "/" + all, 0xFFFFFF, Number(settings.nameSize || 6), 0, 0, false, 0);
				gt.filters = [shadow()];
				if (gt.width > colW) {
					gt.scaleX = gt.scaleY = colW / gt.width;
				}
				gt.x = 6 + (colW - gt.width) / 2;
				gt.y = gy;
				gy += gt.height;
			}
			// Icons in rows of equal length.
			var body:Sprite = new Sprite();
			var perRow:int = only ? Math.max(1, int(settings.perRow || 10)) : Math.ceil(list.length / Math.max(1, rows));
			var x:Number = 0;
			var y:Number = 0;
			var col:int = 0;
			var lastCls:int = -1;
			var iconClass:Class = null;
			try {
				iconClass = COTWGoldChallengeIcons;
			} catch (e:Error) {
			}
			var maxX:Number = 0;
			var h1:Number = size * 1.112;
			for each (var sp:Object in list) {
				if (col == perRow) {
					col = 0;
					x = 0;
					y += h1 + gap + 3;
				}
				if (showCls && sp.cls != lastCls) {
					// class number centred in its own slot, between the previous icon and the next one
					var t:TextField = addText(src, body, String(sp.cls), 0xE8ECF2, clsSize, 0, 0, false, 0);
					t.alpha = 0.8;
					t.filters = [shadow()];
					var slot:Number = Math.max(t.textWidth + gap, size * 0.45);
					t.x = x + (slot - gap) / 2 - t.width / 2;
					t.y = y + h1 / 2 - t.height / 2;
					x += slot;
					lastCls = sp.cls;
				}
				// each icon in an unscaled holder padded around it, filters on the holder: the game clips
				// filters to the bounds of the filtered object, which cut the top of the glow
				var ic:Sprite = padded(iconClass, body, sp, x, y, size, h1, 1); // no extra dimming
				ic.filters = sp.rank <= 1 ? [new GlowFilter(sp.rank == 0 ? 0x9EE9FF : 0xFFC42E, sp.rank == 0 ? 0.7 : 0.45, 8, 8, 2), shadow()] : [shadow()];
				if (avant) {
					var cx:Number = x + size / 2;
					var cy:Number = y + h1 / 2;
					var k:Number = here[sp.hash] ? 1.12 : 0.9;
					ic.scaleX *= k;
					ic.scaleY *= k;
					ic.x = cx - (cx - ic.x) * k;
					ic.y = cy - (cy - ic.y) * k;
					if (!here[sp.hash]) {
						ic.alpha *= attenue;
					}
				}
				if (sp.hash == aim) {
					aimLook(ic, x, y, size, h1);
					aimIcon = ic;
				}
				record(sp, ic, x, y, size, h1);
				x += size + gap;
				maxX = Math.max(maxX, x);
				col++;
			}
			if (aimIcon) {
				body.addChild(aimIcon); // on top of its neighbours
			}
			body.x = 6 + colW + 10;
			body.y = 6;
			wall.addChild(body);
			// icons aligned with the top, centre or bottom of the badge (iconsAlign), not of the texts under it
			var colH:Number = gsize * 1.112;
			var bodyH:Number = list.length ? y + h1 : 0;
			var align:String = settings.iconsAlign || "auto";
			if (align == "auto") {
				align = only ? "center" : "top"; // 100% mode: top, reserve mode: centre
			}
			var shift:Number = align == "top" ? 0 : (colH - bodyH) * (align == "bottom" ? 1 : 0.5);
			if (shift > 0) {
				body.y += shift;
			} else {
				gcol.y -= shift;
			}
			var w:Number = body.x + maxX + 6;
			var hh:Number = Math.max(gcol.y + gy, body.y + y + h1 + 8) + 4;
			if (settings.bg == "1") { // optional dark background, off by default
				wall.graphics.beginFill(0x000000, settings.bgAlpha != null ? Number(settings.bgAlpha) : 0.35);
				wall.graphics.drawRect(0, 0, w, hh);
				wall.graphics.endFill();
			}
			place(wall);
			return [wall, aimIcon];
		}

		// Top right corner of the HUD, taken by the binoculars and calls panels (HUD units).
		private static const PANELS_LEFT:Number = 960;
		private static const PANELS_BOTTOM:Number = 380;

		// Places the wall at wallX, wallY, moved out of the panels corner and kept on screen if needed.
		private static function place(wall:Sprite):void {
			wall.x = Number(settings.wallX || 40);
			wall.y = Number(settings.wallY || 40);
			var b:Rectangle = wall.getBounds(wall);
			var right:Number = b.right * wall.scaleX;
			var bottom:Number = b.bottom * wall.scaleY;
			if (wall.y < PANELS_BOTTOM && wall.x + right > PANELS_LEFT) {
				if (PANELS_LEFT - right >= 0) {
					wall.x = PANELS_LEFT - right;
				} else {
					wall.y = PANELS_BOTTOM;
				}
			}
			// and on screen (the HUD is 1280 x 720)
			wall.x = Math.max(0, Math.min(wall.x, 1280 - right));
			wall.y = Math.max(0, Math.min(wall.y, 720 - bottom));
		}

		// Drop shadow of the texts and icons (0 2px 3px black).
		// Texts shadowed as in the game panels.
		private static function shadowTexts(c:DisplayObjectContainer):void {
			for (var i:int = 0; i < c.numChildren; i++) {
				var o:DisplayObject = c.getChildAt(i);
				if (o is TextField) {
					o.filters = [shadow()];
				} else if (o is DisplayObjectContainer) {
					shadowTexts(DisplayObjectContainer(o));
				}
			}
		}

		private static function shadow():DropShadowFilter {
			return new DropShadowFilter(2, 90, 0x000000, 0.9, 3, 3, 1);
		}

		// Icon in an unscaled holder padded around it, for filters on the holder: the game clips
		// filters to the bounds of the filtered object, which cut the top of the glow.
		private static function padded(iconClass:Class, body:Sprite, sp:Object, x:Number, y:Number, w:Number, h:Number, noneAlpha:Number):Sprite {
			var ic:Sprite = new Sprite();
			addIcon(iconClass, ic, sp, x, y, w, noneAlpha);
			var pad:Number = 14;
			ic.graphics.beginFill(0x000000, 0.004);
			ic.graphics.drawRect(x - pad, y - pad, w + pad * 2, h + pad * 2);
			ic.graphics.endFill();
			body.addChild(ic);
			return ic;
		}

		// Species aimed at with the binoculars: larger, full opacity, white hexagon ring behind it.
		private static function aimLook(ic:Sprite, x:Number, y:Number, w:Number, h:Number):void {
			var k:Number = Number(settings.aimScale || 1.35);
			var cx:Number = x + w / 2;
			var cy:Number = y + h / 2;
			var o:Number = w * 0.07;
			var g:* = ic.graphics;
			g.lineStyle(Math.max(1.2, w * 0.07), 0xFFFFFF, 1);
			var pts:Array = [[0.5, 0], [1, 0.25], [1, 0.75], [0.5, 1], [0, 0.75], [0, 0.25]];
			g.moveTo(x - o + (w + 2 * o) * 0.5, y - o);
			for (var i:int = 1; i <= 6; i++) {
				var p:Array = pts[i % 6];
				g.lineTo(x - o + (w + 2 * o) * p[0], y - o + (h + 2 * o) * p[1]);
			}
			g.lineStyle();
			grow = {ic: ic, cx: cx, cy: cy, from: ic.scaleX, to: k, f: 0, at: k, out: false};
			scaleAround(ic, cx, cy, k);
			ic.alpha = 1;
		}

		private static function scaleAround(ic:DisplayObject, cx:Number, cy:Number, k:Number):void {
			ic.scaleX = ic.scaleY = k;
			ic.x = cx - cx * k;
			ic.y = cy - cy * k;
		}

		// Binoculars movie: the aimed species of the wall, drawn at its place over the HUD movie.
		// Returns whether it is drawn.
		private static function aimOverlay(panel:*, hash:uint):Boolean {
			var root:* = panel.root;
			var st:* = panel.stage;
			if (!root || !st) {
				return false;
			}
			if (!cluePanel) {
				root.addEventListener("enterFrame", overlayFrame);
			}
			cluePanel = panel;
			var key:String = hash + "/" + version + "/" + status + "/" + reservesDone + "/" + settingsVersion + "/" + currentReserve() + "/" + opaqueOn + "/" + st.stageWidth + "x" + st.stageHeight;
			if (key != overlayKey) {
				overlayKey = key;
				if (overlay && overlay.parent) {
					overlay.parent.removeChild(overlay);
				}
				overlay = null;
				var built:Array = hash ? buildWall(findTextField(root), hash) : null;
				if (built && built[1]) {
					var wall:Sprite = built[0];
					var ic:Sprite = built[1];
					var body:Sprite = ic.parent as Sprite;
					wall.graphics.clear();
					body.graphics.clear();
					for (var i:int = wall.numChildren - 1; i >= 0; i--) {
						if (wall.getChildAt(i) != body) {
							wall.removeChildAt(i);
						}
					}
					for (i = body.numChildren - 1; i >= 0; i--) {
						if (body.getChildAt(i) != ic) {
							body.removeChildAt(i);
						}
					}
					var m:* = root.transform.concatenatedMatrix;
					overlay = new Sprite();
					overlay.mouseEnabled = false;
					overlay.mouseChildren = false;
					overlay.scaleX = st.stageWidth / HUD_WIDTH / m.a;
					overlay.scaleY = st.stageHeight / HUD_HEIGHT / m.d;
					CONFIG::preview {
						// outside the game the wall is drawn in this movie
						overlay.scaleX = 1 / m.a;
						overlay.scaleY = 1 / m.d;
					}
					overlay.x = -m.tx / m.a;
					overlay.y = -m.ty / m.d;
					overlay.addChild(wall);
					overlay.visible = false; // overlayFrame starts the growth
					root.addChild(overlay);
				}
			}
			overlayFrame(null);
			return overlay != null;
		}

		// The binoculars movie keeps running once shown: hide the overlay with the panel.
		// Each time it appears, the icon grows from its size in the wall with a slight overshoot;
		// when the animal is no longer aimed at, it shrinks back and fades into the wall.
		private static function overlayFrame(e:*):void {
			pollToggle();
			if (wallChanged) {
				wallChanged = false;
				if (shownPanel) {
					try {
						show(shownPanel, shownIcon);
					} catch (err:Error) {
					}
				}
			}
			if (e && heardClip && cluePanel == heardClip && heardClip.stage && visibleChain(heardClip)) {
				tick();
				aimOverlay(heardClip, uint(hashByIcon[heardIcon]));
			}
			if (!overlay) {
				return;
			}
			var shown:Boolean = cluePanel && cluePanel.stage && visibleChain(cluePanel) && !wallOff && !namesOn;
			if (!grow || grow.ic.parent == null) {
				overlay.visible = shown;
				return;
			}
			// counted in frames, not in time: the movie may not be drawn during its first frames
			var frames:Number = Number(settings.aimAnim != null ? settings.aimAnim : 14);
			if (shown && (!overlay.visible || grow.out)) { // appears
				grow.out = false;
				grow.f = 0;
				overlay.visible = true;
			} else if (!shown && overlay.visible && !grow.out) { // leaves
				grow.out = true;
				grow.f = 0;
				grow.at = grow.ic.scaleX;
			}
			if (!overlay.visible) {
				return;
			}
			if (e) {
				grow.f++;
			}
			var t:Number = frames > 0 ? Math.min(1, grow.f / frames) : 1;
			if (grow.out) {
				var q:Number = t * t; // ease in
				scaleAround(grow.ic, grow.cx, grow.cy, grow.at + (grow.from - grow.at) * q);
				grow.ic.alpha = 1 - q;
				if (t >= 1) {
					overlay.visible = false;
				}
				return;
			}
			var c:Number = 1.7;
			var ease:Number = 1 + (c + 1) * Math.pow(t - 1, 3) + c * Math.pow(t - 1, 2); // ease out back
			scaleAround(grow.ic, grow.cx, grow.cy, grow.from + (grow.to - grow.from) * ease);
			grow.ic.alpha = 1;
		}

		private static function visibleChain(d:*):Boolean {
			for (; d; d = d.parent) {
				if (!d.visible || d.alpha <= 0.01) {
					return false;
				}
			}
			return true;
		}

		// Hexagon gauge filled from the bottom, with the percentage and the count.
		private static function drawGauge(box:Sprite, src:TextField, x:Number, y:Number, w:Number, pct:int, sub:String):void {
			var h:Number = w * 1.112;
			var pts:Array = [[0.5, 0], [1, 0.25], [1, 0.75], [0.5, 1], [0, 0.75], [0, 0.25]];
			var hex:Function = function(g:*, inset:Number):void {
				g.moveTo(x + inset + (w - 2 * inset) * pts[0][0], y + inset + (h - 2 * inset) * pts[0][1]);
				for (var i:int = 1; i <= 6; i++) {
					var p:Array = pts[i % 6];
					g.lineTo(x + inset + (w - 2 * inset) * p[0], y + inset + (h - 2 * inset) * p[1]);
				}
			};
			var back:Shape = new Shape();
			back.graphics.beginFill(0x000000, 0.72);
			hex(back.graphics, 0);
			back.graphics.endFill();
			box.addChild(back);
			var fill:Shape = new Shape();
			var top:Number = y + h * (1 - pct / 100);
			fill.graphics.beginFill(0xFFC42E, 0.9);
			fill.graphics.drawRect(x, top, w, y + h - top);
			fill.graphics.endFill();
			var mask:Shape = new Shape();
			mask.graphics.beginFill(0xFFFFFF, 1);
			hex(mask.graphics, 0);
			mask.graphics.endFill();
			box.addChild(fill);
			box.addChild(mask);
			fill.mask = mask;
			var frame:Shape = new Shape();
			frame.graphics.lineStyle(w * 0.06, 0x7D5C04, 1);
			hex(frame.graphics, w * 0.03);
			frame.graphics.lineStyle(w * 0.025, 0xFFC42E, 1);
			hex(frame.graphics, w * 0.03);
			box.addChild(frame);
			var t:TextField = addText(src, box, pct + "%", 0xFFFFFF, Math.round(h * 0.27), 0, 0, false, 0);
			t.filters = [new GlowFilter(0x000000, 0.75, 4, 4, 3)];
			t.x = x + w / 2 - t.width / 2;
			t.y = y + h * 0.47 - t.height / 2;
			var s:TextField = addText(src, box, sub, 0xFFFFFF, Math.round(h * 0.13), 0, 0, false, 0);
			s.filters = [new GlowFilter(0x000000, 0.75, 3, 3, 3)];
			s.x = x + w / 2 - s.width / 2;
			s.y = y + h * 0.70 - s.height / 2;
		}

		// Lower case without accents, to sort names.
		private static function fold(s:String):String {
			var from:String = "àáâäãåçèéêëìíîïñòóôöõùúûüýÿœæ’";
			var to:Array = ["a", "a", "a", "a", "a", "a", "c", "e", "e", "e", "e", "i", "i", "i", "i", "n", "o", "o", "o", "o", "o", "u", "u", "u", "u", "y", "y", "oe", "ae", "'"];
			var out:String = "";
			s = s.toLowerCase();
			for (var i:int = 0; i < s.length; i++) {
				var j:int = from.indexOf(s.charAt(i));
				out += j >= 0 ? to[j] : s.charAt(i);
			}
			return out;
		}

		private static function addIcon(iconClass:Class, wall:Sprite, sp:Object, x:Number, y:Number, size:Number, noneAlpha:Number = 0.45):DisplayObject {
			var c:uint = sp.rank < 4 ? TINTS[sp.rank] : 0xB4B4B4;
			if (iconClass) {
				var ic:MovieClip = new iconClass() as MovieClip;
				ic.gotoAndStop(int(iconByHash[sp.hash]) + 1);
				var k:Number = size / Math.max(1, ic.width);
				ic.scaleX = ic.scaleY = k;
				var b:Rectangle = ic.getBounds(ic);
				ic.x = x - b.x * k;
				ic.y = y - b.y * k;
				ic.transform.colorTransform = new ColorTransform(((c >> 16) & 255) / 255, ((c >> 8) & 255) / 255, (c & 255) / 255, sp.rank < 4 ? 1 : noneAlpha);
				wall.addChild(ic);
				return ic;
			}
			var dot:Shape = new Shape();
			dot.graphics.beginFill(c, sp.rank < 4 ? 1 : noneAlpha);
			dot.graphics.drawCircle(x + size / 2, y + size / 2, size / 2 - 2);
			dot.graphics.endFill();
			wall.addChild(dot);
			return dot;
		}

		private static function speciesName(hash:uint):String {
			if (nameByHash[hash]) {
				return nameByHash[hash];
			}
			var k:String = keyByHash[hash];
			try {
				var s:* = hud.GetLocalizedStringFromString(k);
				if (s && String(s).indexOf("LOC:") != 0) {
					return String(s);
				}
			} catch (e:Error) {
			}
			k = k.replace("animal_", "").replace("_name", "");
			return k.charAt(0).toUpperCase() + k.substr(1);
		}

		private static function addText(src:TextField, box:Sprite, text:String, color:uint, size:Number, x:Number, y:Number, alignRight:Boolean, dummy:int):TextField {
			var tf:TextField = copyTextField(src);
			var fmt:TextFormat = tf.getTextFormat();
			fmt.size = size;
			fmt.color = color;
			tf.defaultTextFormat = fmt;
			tf.text = text;
			tf.setTextFormat(fmt);
			tf.width = tf.textWidth + 6;
			tf.height = tf.textHeight + 4;
			tf.x = alignRight ? x - tf.width : x;
			tf.y = y;
			box.addChild(tf);
			return tf;
		}

		private static function findTextField(c:*):TextField {
			for (var i:int = 0; i < c.numChildren; i++) {
				var o:* = c.getChildAt(i);
				if (o is TextField && TextField(o).embedFonts) {
					return o;
				}
				if (o is DisplayObjectContainer) {
					var t:TextField = findTextField(o);
					if (t) {
						return t;
					}
				}
			}
			return null;
		}

		private static function show(panel:*, icon:int):void {
			if (!speciesDone) {
				return;
			}
			shownPanel = panel;
			shownIcon = icon;
			// a hidden panel leaves the wall to the audio clue panel
			var wall:Boolean = (panel.stage && visibleChain(panel) ? aimOverlay(panel, uint(hashByIcon[icon])) : overlay != null) && !wallOff;
			var key:String = icon + "/" + version + "/" + status + "/" + scopeReserve() + "/" + wall + "/" + (settings.debug == "1" ? estInfo : "");
			var log:Sprite = panel.getChildByName(LOG_NAME) as Sprite;
			// "harvested" line: line=1 always, line=0 never, default only when no wall shows the species
			var line:String = settings.line || "auto";
			if (settings.debug != "1" && (line == "0" || (line != "1" && wall))) {
				if (log) {
					panel.removeChild(log);
				}
				return;
			}
			var gauge:DisplayObject = panel.getChildByName(GAUGE_NAME);
			if (log && key == shownKey) {
				if (gauge) {
					log.y = gauge.y + gauge.height;
				}
				return;
			}
			shownKey = key;
			if (log) {
				panel.removeChild(log);
			}
			log = new Sprite();
			log.name = LOG_NAME;
			log.x = -352.1;
			log.y = gauge ? gauge.y + gauge.height : 169;
			var hash:uint = hashByIcon[icon];
			var parts:Array = [];
			var colors:Array = [];
			var tiers:Array = String(settings.tiers || "DIAMOND,GOLD,SILVER,BRONZE").split(",");
			var label:String = settings.label || "HARVESTED";
			var none:String = settings.none || "none";
			if (hash) {
				// Best medal since the start of the challenge (since=), or ever when no date is set.
				var got:Array = best(hash, since, scopeReserve());
				parts.push(label + "  ");
				colors.push(0xFFFFFF);
				parts.push(got[0] < 4 ? tiers[got[0]] : none);
				colors.push(got[0] < 4 ? COLORS[got[0]] : 0xAAAAAA);
			} else {
				parts.push("species " + icon + " unknown");
				colors.push(0xAAAAAA);
			}
			var y:Number = 3;
			y = addLine(panel, log, parts, colors, y, 13);
			if (settings.debug == "1") {
				y = addLine(panel, log, [status + " · icon " + icon + " · hash " + hash + " · overlay " + wall], [0xAAAAAA], y, 11);
				y = addLine(panel, log, ["estimate " + estInfo], [0xAAAAAA], y, 11);
			}
			log.graphics.beginFill(0x000000, 0.65);
			log.graphics.drawRect(0, 0, 352.1, y + 3);
			log.graphics.endFill();
			panel.addChild(log);
		}

		private static function addLine(panel:*, log:Sprite, parts:Array, colors:Array, y:Number, size:Number):Number {
			return addLineFrom(panel.tf_difficulty, log, parts, colors, y, size, panel.MCI_score.tf_score.filters);
		}

		private static function addLineFrom(src:TextField, log:Sprite, parts:Array, colors:Array, y:Number, size:Number, filters:Array = null):Number {
			var x:Number = 8;
			var h:Number = 0;
			for (var i:int = 0; i < parts.length; i++) {
				var tf:TextField = copyTextField(src);
				if (filters) {
					tf.filters = filters;
				}
				var fmt:TextFormat = tf.getTextFormat();
				fmt.size = size;
				fmt.color = colors[i];
				tf.defaultTextFormat = fmt;
				tf.text = parts[i];
				tf.setTextFormat(fmt);
				tf.width = tf.textWidth + 6;
				tf.x = x;
				tf.y = y;
				log.addChild(tf);
				x += tf.textWidth;
				h = Math.max(h, tf.textHeight);
			}
			return y + h + 2;
		}

		// Same as the panel's own copyTextField, which is not public.
		private static function copyTextField(src:TextField):TextField {
			var tf:TextField = new TextField();
			tf.width = src.width;
			tf.height = src.height;
			tf.multiline = false;
			tf.wordWrap = false;
			tf.embedFonts = src.embedFonts;
			tf.antiAliasType = src.antiAliasType;
			tf.autoSize = src.autoSize;
			tf.defaultTextFormat = src.getTextFormat();
			tf.selectable = false;
			return tf;
		}

		private static function trim(s:String):String {
			var a:int = 0;
			var b:int = s.length;
			while (a < b && s.charCodeAt(a) <= 32) {
				a++;
			}
			while (b > a && s.charCodeAt(b - 1) <= 32) {
				b--;
			}
			return s.substring(a, b);
		}
	}
}
