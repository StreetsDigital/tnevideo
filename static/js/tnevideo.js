/**
 * TNE Video - Self-Contained Publisher Integration
 *
 * A single JS file that:
 *   1. Loads Prebid.js and Google Publisher Tag (GPT)
 *   2. Configures User ID modules for identity resolution
 *   3. Sends server-side bid requests to the TNE Catalyst Prebid Server
 *   4. Passes bid responses to Google Ad Manager (GAM) via targeting keys
 *   5. Supports GAM Universal Creative for banner/outstream rendering
 *   6. Supports VAST redirect creatives for instream video
 *
 * Usage:
 *   <script src="https://your-cdn.com/tnevideo.js"></script>
 *   <script>
 *     TNEVideo.init({
 *       publisherId: 'pub-12345',
 *       serverUrl: 'https://pbs.yourdomain.com',
 *       adUnits: [ ... ],
 *       gamNetworkId: '/19968336',
 *     });
 *   </script>
 *
 * @version 1.0.0
 * @license Apache-2.0
 */
(function (window, document) {
  'use strict';

  // ─── Default Configuration ───────────────────────────────────────────
  var DEFAULTS = {
    // Prebid.js CDN (publisher can override with a custom build)
    prebidUrl: 'https://cdn.jsdelivr.net/npm/prebid.js@latest/dist/not-for-prod/prebid.js',

    // Server-side bidding config
    serverUrl: '',          // Required: TNE Catalyst Prebid Server URL
    publisherId: '',        // Required: Publisher account ID
    timeout: 1500,          // Total auction timeout (ms)
    s2sTimeout: 1000,       // Server-side auction timeout (ms)

    // Bidders routed server-side (these run on Catalyst, not in-browser)
    s2sBidders: ['appnexus', 'rubicon', 'pubmatic'],

    // GAM config
    gamNetworkId: '',       // Required: GAM network code, e.g. '/19968336'
    enableSendAllBids: false,

    // User ID modules to activate
    userIds: [
      {
        name: 'sharedId',
        storage: { name: '_sharedid', type: 'cookie', expires: 365 }
      },
      {
        name: 'pubProvidedId'
      }
    ],

    // GDPR / US Privacy (CMP integration)
    consentManagement: {
      gdpr: {
        cmpApi: 'iab',
        timeout: 3000,
        defaultGdprScope: true
      },
      usp: {
        cmpApi: 'iab',
        timeout: 1000
      }
    },

    // Video defaults
    video: {
      mimes: ['video/mp4', 'video/webm'],
      protocols: [2, 3, 5, 6],     // VAST 2.0, 3.0, 5.0, 6.0
      maxduration: 30,
      minduration: 5,
      playbackmethod: [2],          // Auto-play muted
      linearity: 1,                 // Linear (in-stream)
      api: [2]                      // VPAID 2.0
    },

    // Callbacks
    onAuctionEnd: null,       // function(bids) {}
    onBidWon: null,           // function(bid) {}
    onError: null,            // function(error) {}

    // Debug
    debug: false
  };

  // ─── TNEVideo Module ─────────────────────────────────────────────────
  var TNEVideo = {
    version: '1.0.0',
    _config: {},
    _adUnits: [],
    _ready: false,
    _prebidLoaded: false,
    _gptLoaded: false,
    _queue: [],

    /**
     * Initialize TNEVideo with publisher configuration.
     * Loads Prebid.js + GPT, configures ID modules, s2sConfig, and GAM.
     *
     * @param {Object} config - Publisher configuration
     */
    init: function (config) {
      var self = this;
      this._config = mergeDeep({}, DEFAULTS, config || {});

      // Validate required fields
      if (!this._config.serverUrl) {
        this._error('serverUrl is required');
        return;
      }
      if (!this._config.publisherId) {
        this._error('publisherId is required');
        return;
      }
      if (!this._config.adUnits || !this._config.adUnits.length) {
        this._error('adUnits array is required');
        return;
      }

      this._adUnits = this._config.adUnits;
      this._log('Initializing TNEVideo v' + this.version);
      this._log('Server: ' + this._config.serverUrl);
      this._log('Publisher: ' + this._config.publisherId);

      // Load GPT and Prebid.js in parallel
      this._loadGPT(function () {
        self._gptLoaded = true;
        self._checkReady();
      });

      this._loadPrebid(function () {
        self._prebidLoaded = true;
        self._configurePrebid();
        self._checkReady();
      });
    },

    /**
     * Add additional ad units after init.
     * @param {Array} adUnits
     */
    addAdUnits: function (adUnits) {
      if (!Array.isArray(adUnits)) adUnits = [adUnits];
      this._adUnits = this._adUnits.concat(adUnits);
      if (this._ready) {
        window.pbjs.que.push(function () {
          window.pbjs.addAdUnits(adUnits);
        });
      }
    },

    /**
     * Trigger a new auction for the configured ad units.
     * Handles both video and banner flows.
     * @param {Object} [options] - Override options for this auction
     */
    refresh: function (options) {
      var self = this;
      options = options || {};

      if (!this._ready) {
        this._queue.push(function () { self.refresh(options); });
        return;
      }

      this._log('Starting auction refresh');

      window.pbjs.que.push(function () {
        window.pbjs.requestBids({
          timeout: options.timeout || self._config.timeout,
          bidsBackHandler: function (bidResponses) {
            self._log('Bids received', bidResponses);
            self._handleBidsBack(bidResponses, options);
          }
        });
      });
    },

    /**
     * Build a video VAST URL for a specific ad unit (for video player integration).
     *
     * @param {Object} params
     * @param {Object} params.adUnit - The video ad unit config
     * @param {string} params.iu     - GAM ad unit path (e.g. '/19968336/preroll')
     * @param {Object} [params.custParams] - Additional custom targeting params
     * @returns {string|null} GAM VAST tag URL with Prebid targeting
     */
    buildVideoUrl: function (params) {
      if (!this._ready || !window.pbjs) return null;

      return window.pbjs.adServers.dfp.buildVideoUrl({
        adUnit: params.adUnit,
        params: {
          iu: params.iu,
          cust_params: params.custParams || {},
          output: 'vast'
        }
      });
    },

    /**
     * Build a video VAST URL for ad pods (long-form video).
     *
     * @param {Object} params
     * @param {string} params.iu     - GAM ad unit path
     * @param {number} params.num_ads - Number of ads in the pod
     * @returns {string|null} GAM VAST tag URL for ad pod
     */
    buildAdpodVideoUrl: function (params) {
      if (!this._ready || !window.pbjs || !window.pbjs.adServers.dfp.buildAdpodVideoUrl) {
        return null;
      }

      return window.pbjs.adServers.dfp.buildAdpodVideoUrl({
        codes: this._adUnits.filter(function (u) { return u.mediaTypes && u.mediaTypes.video; })
          .map(function (u) { return u.code; }),
        params: {
          iu: params.iu,
          description_url: params.descriptionUrl || window.location.href
        },
        callback: params.callback
      });
    },

    // ─── Internal Methods ────────────────────────────────────────────

    /**
     * Load Google Publisher Tag
     */
    _loadGPT: function (callback) {
      if (window.googletag && window.googletag.apiReady) {
        callback();
        return;
      }

      window.googletag = window.googletag || { cmd: [] };
      loadScript('https://securepubads.g.doubleclick.net/tag/js/gpt.js', callback);
    },

    /**
     * Load Prebid.js from CDN or custom URL
     */
    _loadPrebid: function (callback) {
      if (window.pbjs) {
        callback();
        return;
      }

      window.pbjs = window.pbjs || { que: [] };
      loadScript(this._config.prebidUrl, callback);
    },

    /**
     * Configure Prebid.js with s2sConfig, userId, consent, and cache settings
     */
    _configurePrebid: function () {
      var self = this;
      var cfg = this._config;

      window.pbjs.que.push(function () {
        // ── Server-to-Server Config ──
        // Routes specified bidders through TNE Catalyst Prebid Server
        var s2sConfig = {
          accountId: cfg.publisherId,
          bidders: cfg.s2sBidders,
          adapter: 'prebidServer',
          enabled: true,
          timeout: cfg.s2sTimeout,
          endpoint: {
            p1Consent: cfg.serverUrl + '/openrtb2/auction',
            noP1Consent: cfg.serverUrl + '/openrtb2/auction'
          },
          syncEndpoint: {
            p1Consent: cfg.serverUrl + '/cookie_sync',
            noP1Consent: cfg.serverUrl + '/cookie_sync'
          },
          extPrebid: {
            cache: {
              vastxml: { returnCreative: false }
            },
            targeting: {
              includewinners: true,
              includebidderkeys: !cfg.enableSendAllBids
            }
          }
        };

        // ── Prebid Cache ──
        // Tells Prebid.js to cache VAST XML on our server for video ads
        var cacheConfig = {
          url: cfg.serverUrl + '/cache'
        };

        // ── User ID Modules ──
        // Resolves identifiers client-side before sending to Prebid Server
        var userSyncConfig = {
          userIds: cfg.userIds,
          syncDelay: 3000,
          auctionDelay: 200
        };

        // ── Apply All Config ──
        window.pbjs.setConfig({
          debug: cfg.debug,
          s2sConfig: s2sConfig,
          cache: cacheConfig,
          userSync: userSyncConfig,
          enableSendAllBids: cfg.enableSendAllBids,
          targetingControls: {
            allowTargetingKeys: [
              'BIDDER', 'AD_ID', 'PRICE_BUCKET', 'SIZE',
              'DEAL', 'SOURCE', 'FORMAT', 'UUID',
              'CACHE_ID', 'CACHE_HOST', 'CACHE_PATH',
              'ADOMAIN'
            ]
          }
        });

        // ── Consent Management ──
        if (cfg.consentManagement) {
          window.pbjs.setConfig({
            consentManagement: cfg.consentManagement
          });
        }

        // ── Register Ad Units ──
        var prebidAdUnits = self._buildPrebidAdUnits(cfg.adUnits);
        window.pbjs.addAdUnits(prebidAdUnits);

        // ── Event Hooks ──
        if (cfg.onBidWon) {
          window.pbjs.onEvent('bidWon', cfg.onBidWon);
        }
        if (cfg.onAuctionEnd) {
          window.pbjs.onEvent('auctionEnd', cfg.onAuctionEnd);
        }

        self._log('Prebid.js configured');
      });
    },

    /**
     * Transform publisher ad unit configs into Prebid.js ad unit format
     */
    _buildPrebidAdUnits: function (adUnits) {
      var self = this;
      return adUnits.map(function (unit) {
        var pbUnit = {
          code: unit.code,
          mediaTypes: {},
          bids: []
        };

        // Banner
        if (unit.mediaTypes && unit.mediaTypes.banner) {
          pbUnit.mediaTypes.banner = unit.mediaTypes.banner;
        }

        // Video
        if (unit.mediaTypes && unit.mediaTypes.video) {
          pbUnit.mediaTypes.video = mergeDeep(
            {}, self._config.video, unit.mediaTypes.video
          );
        }

        // Native
        if (unit.mediaTypes && unit.mediaTypes.native) {
          pbUnit.mediaTypes.native = unit.mediaTypes.native;
        }

        // Server-side bidders (routed through Catalyst)
        self._config.s2sBidders.forEach(function (bidder) {
          var bidderParams = (unit.bids && unit.bids[bidder]) || {};
          pbUnit.bids.push({
            bidder: bidder,
            params: bidderParams
          });
        });

        // Any additional client-side bidders the publisher defined
        if (unit.bids) {
          Object.keys(unit.bids).forEach(function (bidder) {
            if (self._config.s2sBidders.indexOf(bidder) === -1) {
              pbUnit.bids.push({
                bidder: bidder,
                params: unit.bids[bidder]
              });
            }
          });
        }

        return pbUnit;
      });
    },

    /**
     * Handle bid responses: set targeting on GAM slots and refresh
     */
    _handleBidsBack: function (bidResponses, options) {
      var self = this;
      var cfg = this._config;

      window.googletag.cmd.push(function () {
        // Set Prebid targeting on all GPT ad slots
        window.pbjs.que.push(function () {
          window.pbjs.setTargetingForGPTAsync();

          self._log('Targeting set on GAM slots');

          // For video ad units, the publisher uses buildVideoUrl() directly
          // with their video player. For display, we refresh GAM.
          var displaySlots = self._getDisplaySlots();
          if (displaySlots.length > 0) {
            window.googletag.pubads().refresh(displaySlots);
            self._log('GAM display slots refreshed: ' + displaySlots.length);
          }

          if (cfg.onAuctionEnd) {
            cfg.onAuctionEnd(bidResponses);
          }
        });
      });
    },

    /**
     * Set up GAM slots from ad unit config (called once on ready)
     */
    _setupGAMSlots: function () {
      var self = this;
      var cfg = this._config;

      window.googletag.cmd.push(function () {
        self._adUnits.forEach(function (unit) {
          var gamPath = unit.gamAdUnitPath || (cfg.gamNetworkId + '/' + unit.code);

          if (unit.mediaTypes && unit.mediaTypes.video &&
              unit.mediaTypes.video.context === 'instream') {
            // Instream video: no GPT slot needed (player handles it)
            self._log('Instream video unit (no GPT slot): ' + unit.code);
            return;
          }

          // Determine sizes
          var sizes = [];
          if (unit.mediaTypes && unit.mediaTypes.banner && unit.mediaTypes.banner.sizes) {
            sizes = unit.mediaTypes.banner.sizes;
          }

          // Create display or outstream slot
          var slot = window.googletag.defineSlot(gamPath, sizes, unit.code);
          if (slot) {
            slot.addService(window.googletag.pubads());
            self._log('GAM slot defined: ' + gamPath + ' -> #' + unit.code);
          }
        });

        // Enable single request mode for efficiency
        window.googletag.pubads().enableSingleRequest();

        // Enable services
        window.googletag.enableServices();

        // Display all defined slots
        self._adUnits.forEach(function (unit) {
          if (unit.mediaTypes && unit.mediaTypes.video &&
              unit.mediaTypes.video.context === 'instream') {
            return;
          }
          window.googletag.display(unit.code);
        });
      });
    },

    /**
     * Get the GPT slots for display ad units (excludes instream video)
     */
    _getDisplaySlots: function () {
      var displayCodes = this._adUnits
        .filter(function (u) {
          return !(u.mediaTypes && u.mediaTypes.video &&
                   u.mediaTypes.video.context === 'instream');
        })
        .map(function (u) { return u.code; });

      var allSlots = window.googletag.pubads().getSlots();
      return allSlots.filter(function (slot) {
        return displayCodes.indexOf(slot.getSlotElementId()) !== -1;
      });
    },

    /**
     * Both Prebid.js and GPT are loaded -- fire initial auction
     */
    _checkReady: function () {
      if (!this._prebidLoaded || !this._gptLoaded) return;

      this._ready = true;
      this._log('TNEVideo ready');

      // Set up GAM slots
      this._setupGAMSlots();

      // Run initial auction
      this.refresh();

      // Process queued calls
      while (this._queue.length) {
        this._queue.shift()();
      }
    },

    _log: function () {
      if (!this._config.debug) return;
      var args = Array.prototype.slice.call(arguments);
      args.unshift('[TNEVideo]');
      console.log.apply(console, args);
    },

    _error: function (msg) {
      console.error('[TNEVideo] ' + msg);
      if (this._config.onError) {
        this._config.onError(new Error(msg));
      }
    }
  };

  // ─── Utility Functions ───────────────────────────────────────────────

  /**
   * Load an external script asynchronously
   */
  function loadScript(url, callback) {
    var script = document.createElement('script');
    script.src = url;
    script.async = true;
    script.onload = function () {
      if (callback) callback();
    };
    script.onerror = function () {
      console.error('[TNEVideo] Failed to load: ' + url);
      if (callback) callback();
    };
    (document.head || document.getElementsByTagName('head')[0]).appendChild(script);
  }

  /**
   * Deep merge objects (simple version, no circular refs)
   */
  function mergeDeep(target) {
    for (var i = 1; i < arguments.length; i++) {
      var source = arguments[i];
      if (!source) continue;
      Object.keys(source).forEach(function (key) {
        if (isObject(source[key]) && isObject(target[key])) {
          mergeDeep(target[key], source[key]);
        } else {
          target[key] = source[key];
        }
      });
    }
    return target;
  }

  function isObject(item) {
    return item && typeof item === 'object' && !Array.isArray(item);
  }

  // ─── Expose ──────────────────────────────────────────────────────────
  window.TNEVideo = TNEVideo;

})(window, document);
