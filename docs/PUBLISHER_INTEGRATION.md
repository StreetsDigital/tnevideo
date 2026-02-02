# TNE Publisher Integration Guide

## Overview

TNEVideo provides a self-contained JavaScript integration that connects your page to the TNE Catalyst Prebid Server. A single `<script>` tag handles:

1. **Identity Resolution** -- Loads Prebid.js User ID modules to resolve visitor identifiers (SharedID, ID5, UID2, etc.)
2. **Server-Side Bidding** -- Sends auction requests to TNE Catalyst, which runs parallel bidding across SSPs server-side
3. **GAM Delivery** -- Passes bid responses back to Google Ad Manager via targeting key-values, supporting both banner (Universal Creative) and video (VAST redirect) formats

```
Publisher Page                TNE Catalyst               SSPs
     |                            |                       |
     |  1. tnevideo.js loads      |                       |
     |     resolves user IDs      |                       |
     |                            |                       |
     |  2. POST /openrtb2/auction |                       |
     |  ───────────────────────>  |  3. Parallel bids     |
     |     (with user.eids)       |  ──────────────────>  |
     |                            |  <──────────────────  |
     |  4. Bid response           |                       |
     |  <───────────────────────  |                       |
     |     (with targeting keys)  |                       |
     |                            |                       |
     |  5. Set targeting on GAM   |                       |
     |  6. GAM renders ad         |                       |
```

## Quick Start

### 1. Add the script to your page

```html
<script src="https://your-cdn.com/tnevideo.js"></script>
```

### 2. Initialize with your config

```html
<script>
  TNEVideo.init({
    serverUrl: 'https://pbs.yourdomain.com',
    publisherId: 'pub-12345',
    gamNetworkId: '/19968336',
    adUnits: [
      {
        code: 'banner-1',
        mediaTypes: { banner: { sizes: [[300, 250]] } },
        bids: {
          appnexus: { placementId: 13144370 }
        }
      }
    ]
  });
</script>
```

### 3. Add a div for each ad slot

```html
<div id="banner-1"></div>
```

That's it. TNEVideo handles loading Prebid.js and GPT, running the auction, and rendering ads.

---

## Configuration Reference

### Required Fields

| Field | Type | Description |
|---|---|---|
| `serverUrl` | string | TNE Catalyst Prebid Server URL (e.g. `https://pbs.yourdomain.com`) |
| `publisherId` | string | Your publisher account ID registered in Catalyst |
| `adUnits` | array | Array of ad unit configurations (see [Ad Units](#ad-units)) |

### Optional Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `gamNetworkId` | string | `''` | GAM network code (e.g. `'/19968336'`). Used to auto-generate GAM ad unit paths. |
| `timeout` | number | `1500` | Total auction timeout in milliseconds |
| `s2sTimeout` | number | `1000` | Server-side auction timeout in milliseconds |
| `s2sBidders` | array | `['appnexus', 'rubicon', 'pubmatic']` | Bidders routed through Catalyst server-side |
| `enableSendAllBids` | boolean | `false` | When `true`, sends all bids to GAM (not just the winner). See [Send All Bids vs Send Top Price](#send-all-bids-vs-send-top-price). |
| `prebidUrl` | string | jsdelivr CDN | URL to load Prebid.js from. Override to use a custom build. |
| `userIds` | array | SharedID + PubProvided | User ID module configurations. See [User ID Modules](#user-id-modules). |
| `consentManagement` | object | GDPR + USP via IAB CMP | Consent management config. See [Privacy & Consent](#privacy--consent). |
| `video` | object | See below | Default video parameters merged into all video ad units |
| `debug` | boolean | `false` | Enable console logging with `[TNEVideo]` prefix |

### Callbacks

| Field | Type | Description |
|---|---|---|
| `onAuctionEnd` | function(bids) | Called when all bids are received |
| `onBidWon` | function(bid) | Called when a bid wins the GAM auction |
| `onError` | function(error) | Called on initialization or auction errors |

---

## Ad Units

Each ad unit describes one placement on the page. The `code` must match the `id` of a `<div>` on the page (for banner) or be used as a reference (for video).

### Banner Ad Unit

```javascript
{
  code: 'banner-1',                              // Matches <div id="banner-1">
  mediaTypes: {
    banner: {
      sizes: [[300, 250], [300, 600]]            // Accepted creative sizes
    }
  },
  bids: {
    appnexus: { placementId: 13144370 },         // Bidder-specific params
    rubicon: { accountId: 1001, siteId: 113932, zoneId: 535510 }
  }
}
```

### Instream Video Ad Unit

```javascript
{
  code: 'video-preroll',
  gamAdUnitPath: '/19968336/preroll',             // Optional: explicit GAM path
  mediaTypes: {
    video: {
      context: 'instream',                        // Required for video player flow
      playerSize: [640, 360],
      mimes: ['video/mp4'],
      protocols: [2, 5],                          // VAST 2.0, 3.0
      maxduration: 30,
      linearity: 1,
      api: [2],                                   // VPAID 2.0
      skip: 1,                                    // Skippable
      skipafter: 5                                // Skip button after 5 seconds
    }
  },
  bids: {
    appnexus: { placementId: 13232385 },
    rubicon: { accountId: 1001, siteId: 113932, zoneId: 535512 }
  }
}
```

### Outstream Video Ad Unit

```javascript
{
  code: 'outstream-1',                            // Matches <div id="outstream-1">
  mediaTypes: {
    video: {
      context: 'outstream',                       // Renders in a banner slot
      playerSize: [640, 360],
      mimes: ['video/mp4'],
      protocols: [2, 5],
      maxduration: 30
    }
  },
  bids: {
    appnexus: { placementId: 13232386 }
  }
}
```

### Multi-Format Ad Unit

```javascript
{
  code: 'multi-1',
  mediaTypes: {
    banner: { sizes: [[300, 250]] },
    video: {
      context: 'outstream',
      playerSize: [300, 250],
      mimes: ['video/mp4']
    }
  },
  bids: {
    appnexus: { placementId: 13144370 }
  }
}
```

---

## Instream Video Player Integration

For instream video (pre-roll, mid-roll, post-roll), the video player needs a VAST URL from GAM. Use `TNEVideo.buildVideoUrl()` in the `onAuctionEnd` callback:

```javascript
TNEVideo.init({
  serverUrl: 'https://pbs.yourdomain.com',
  publisherId: 'pub-12345',
  adUnits: [
    {
      code: 'preroll',
      gamAdUnitPath: '/19968336/preroll',
      mediaTypes: {
        video: {
          context: 'instream',
          playerSize: [640, 360],
          mimes: ['video/mp4'],
          protocols: [2, 5],
          maxduration: 30
        }
      },
      bids: {
        appnexus: { placementId: 13232385 }
      }
    }
  ],

  onAuctionEnd: function (bids) {
    var vastUrl = TNEVideo.buildVideoUrl({
      adUnit: { code: 'preroll', mediaTypes: { video: { context: 'instream' } } },
      iu: '/19968336/preroll'
    });

    if (vastUrl) {
      // Pass to your video player
      player.src({ src: vastUrl, type: 'video/mp4' });
      // or for Video.js:  videojs('my-player').ima({ adTagUrl: vastUrl });
      // or for JW Player:  jwplayer('player').setup({ advertising: { tag: vastUrl } });
    }
  }
});
```

### How the Video Flow Works

```
1. tnevideo.js runs auction → Catalyst returns bids with VAST XML
2. Prebid.js caches VAST XML at /cache → receives a UUID
3. buildVideoUrl() constructs a GAM VAST tag URL with hb_uuid in the query string
4. Video player calls the GAM URL
5. GAM matches a line item (based on hb_pb price bucket)
6. GAM returns a VAST redirect pointing to /cache?uuid=<hb_uuid>
7. Video player follows the redirect, gets the cached VAST XML, plays the ad
```

---

## Targeting Key-Values

Every bid from Catalyst includes targeting key-values in `bid.ext.prebid.targeting`. These are set on GAM ad slots automatically by `tnevideo.js`.

All bidder codes are prefixed with `tne_` (e.g. `tne_appnexus`, `tne_rubicon`) to namespace TNE-routed demand.

### Core Keys (Always Present)

| Key | Example Value | Description |
|---|---|---|
| `hb_pb` | `5.00` | Price bucket (used for GAM line item targeting) |
| `hb_bidder` | `tne_appnexus` | Winning bidder name |
| `hb_format` | `video` | Media type: `banner`, `video`, `native`, or `audio` |

### Conditional Keys

| Key | Example Value | When Present |
|---|---|---|
| `hb_size` | `300x250` | Banner bids with valid dimensions |
| `hb_deal` | `deal123` | Private marketplace deal bids |
| `hb_adid` | `abc123` | Creative ID (from bid CRID or AdID) |
| `hb_adomain` | `ford.com` | Advertiser domain (from bid adomain[]) |

### Cache Keys (When Prebid Cache Is Active)

| Key | Example Value | Description |
|---|---|---|
| `hb_uuid` | `16c887cf-58db-4f0a-a1e3-...` | Cache UUID for VAST retrieval |
| `hb_cache_id` | `16c887cf-58db-4f0a-a1e3-...` | Same as hb_uuid (Prebid Server convention) |
| `hb_cache_host` | `https://pbs.yourdomain.com` | Cache server host |
| `hb_cache_path` | `/cache` | Cache endpoint path |

### Bidder-Suffixed Keys

In Send All Bids mode, every key also gets a bidder-suffixed variant:

```
hb_pb_tne_appnexus = 5.00
hb_bidder_tne_appnexus = tne_appnexus
hb_format_tne_appnexus = video
hb_adomain_tne_appnexus = ford.com
```

### Price Bucket Granularity

Prices are bucketed using medium granularity:

| Price Range | Increment | Example |
|---|---|---|
| $0.00 - $5.00 | $0.01 | $2.53 |
| $5.00 - $10.00 | $0.05 | $7.25 |
| $10.00 - $20.00 | $0.50 | $15.50 |
| $20.00+ | Capped | $20.00 |

---

## GAM Creative Setup

### Banner / Outstream: Universal Creative

Create a **Third-Party** creative in GAM with this HTML:

```html
<script src="https://cdn.jsdelivr.net/npm/prebid-universal-creative@latest/dist/creative.js"></script>
<script>
  var ucTagData = {};
  ucTagData.adServerDomain = "";
  ucTagData.pubUrl = "%%PATTERN:url%%";
  ucTagData.targetingMap = %%PATTERN:TARGETINGMAP%%;
  ucTagData.hbPb = "%%PATTERN:hb_pb%%";
  ucTagData.requestAllAssets = true;
  try {
    ucTag.renderAd(document, ucTagData);
  } catch (e) {
    console.log(e);
  }
</script>
```

**Creative settings:**
- Size: 1x1 (serves to all inventory sizes)
- "Serve into a SafeFrame" can be checked or unchecked

### Instream Video: VAST Redirect

Create a **VAST redirect** creative in GAM:

```
VAST Tag URL: https://pbs.yourdomain.com/cache?uuid=%%PATTERN:hb_uuid%%
```

**Creative settings:**
- Duration: Set to your max expected ad length (e.g. 30s)
- GAM will warn that VAST fetch failed during setup -- this is expected since the UUID macro hasn't resolved yet

### Line Item Targeting

Create line items with key-value targeting on `hb_pb`:

| Line Item | Price | Targeting |
|---|---|---|
| TNE_$0.50 | $0.50 | `hb_pb = 0.50` |
| TNE_$1.00 | $1.00 | `hb_pb = 1.00` |
| TNE_$1.50 | $1.50 | `hb_pb = 1.50` |
| ... | ... | ... |
| TNE_$20.00 | $20.00 | `hb_pb = 20.00` |

**Register these key-values in GAM** (Inventory > Key-values):

| Key | Values | Type |
|---|---|---|
| `hb_pb` | Predefined: `0.50`, `1.00`, ... `20.00` | Predefined |
| `hb_bidder` | Dynamic | Free-form |
| `hb_format` | Predefined: `banner`, `video`, `native` | Predefined |
| `hb_size` | Dynamic (e.g. `300x250`) | Free-form |
| `hb_uuid` | Dynamic | Free-form |
| `hb_adomain` | Dynamic (e.g. `ford.com`) | Free-form |
| `hb_deal` | Dynamic | Free-form |

---

## Send All Bids vs Send Top Price

| Mode | `enableSendAllBids` | What GAM Sees | When to Use |
|---|---|---|---|
| **Send Top Price** (default) | `false` | Only the winning bid's targeting keys (e.g. `hb_pb`, `hb_bidder`) | Most publishers. Simpler GAM setup. |
| **Send All Bids** | `true` | Every bidder's keys with bidder suffix (e.g. `hb_pb_tne_appnexus`, `hb_pb_tne_rubicon`) | When you want per-bidder line items or reporting in GAM. |

Send Top Price requires one set of line items. Send All Bids requires one set per bidder.

---

## User ID Modules

User ID modules resolve visitor identifiers client-side before the auction request. These IDs are sent to Catalyst as `user.eids` in the OpenRTB request, improving match rates and bid performance.

### Default Configuration

```javascript
userIds: [
  {
    name: 'sharedId',
    storage: { name: '_sharedid', type: 'cookie', expires: 365 }
  },
  { name: 'pubProvidedId' }
]
```

### Adding More ID Providers

```javascript
userIds: [
  // SharedID (free, first-party)
  {
    name: 'sharedId',
    storage: { name: '_sharedid', type: 'cookie', expires: 365 }
  },

  // Unified ID 2.0
  {
    name: 'unifiedId',
    params: { partner: 'your-partner-id' },
    storage: { name: 'unifiedid', type: 'cookie', expires: 30 }
  },

  // ID5
  {
    name: 'id5Id',
    params: { partner: 1234 },
    storage: { name: 'id5id', type: 'html5', expires: 90 }
  },

  // LiveRamp / RampID
  {
    name: 'identityLink',
    params: { pid: '123' },
    storage: { name: 'idl_env', type: 'cookie', expires: 30 }
  },

  // Publisher-provided (pass your own first-party ID)
  {
    name: 'pubProvidedId',
    params: {
      eids: [{
        source: 'yourdomain.com',
        uids: [{ id: 'your-first-party-id', atype: 1 }]
      }]
    }
  }
]
```

### Which Prebid.js Build Do I Need?

If you use the default CDN Prebid.js, all common modules are included. For a custom build (recommended for production -- smaller file size), build with the modules you need:

```bash
gulp build --modules=prebidServerBidAdapter,dfpAdServerVideo,userId,sharedIdSystem,id5IdSystem,unifiedIdSystem,consentManagement,consentManagementUsp
```

Override the URL:

```javascript
TNEVideo.init({
  prebidUrl: 'https://your-cdn.com/prebid-custom.js',
  ...
});
```

---

## Privacy & Consent

TNEVideo integrates with IAB Transparency & Consent Framework (TCF) and US Privacy String (USP / CCPA).

### Default (IAB CMP Auto-Detection)

```javascript
consentManagement: {
  gdpr: {
    cmpApi: 'iab',        // Auto-detect IAB-compliant CMP on the page
    timeout: 3000,         // Max wait for CMP response
    defaultGdprScope: true // Assume GDPR applies if CMP doesn't respond
  },
  usp: {
    cmpApi: 'iab',
    timeout: 1000
  }
}
```

### Static Consent String (Testing)

```javascript
consentManagement: {
  gdpr: {
    cmpApi: 'static',
    consentData: {
      getTCData: {
        tcString: 'your-base64-tcf-string',
        gdprApplies: true
      }
    }
  }
}
```

### Disable Consent (Non-GDPR Markets)

```javascript
consentManagement: null
```

When consent data is present, it flows through to Catalyst as `regs.ext.gdpr` and `user.ext.consent` in the OpenRTB request. Catalyst enforces privacy at the server level -- filtering bidders, removing EIDs, and blocking personalized ads when required.

---

## API Reference

### `TNEVideo.init(config)`

Initialize the integration. Loads Prebid.js and GPT, configures everything, runs the first auction automatically.

### `TNEVideo.refresh([options])`

Trigger a new auction. Useful for infinite scroll or single-page apps.

```javascript
TNEVideo.refresh();
TNEVideo.refresh({ timeout: 2000 });   // Override timeout for this auction
```

### `TNEVideo.addAdUnits(adUnits)`

Add ad units after initialization.

```javascript
TNEVideo.addAdUnits({
  code: 'sidebar-1',
  mediaTypes: { banner: { sizes: [[160, 600]] } },
  bids: { appnexus: { placementId: 13144372 } }
});
```

### `TNEVideo.buildVideoUrl(params)`

Build a GAM VAST tag URL for instream video. Returns `null` if not ready.

```javascript
var url = TNEVideo.buildVideoUrl({
  adUnit: { code: 'preroll', mediaTypes: { video: { context: 'instream' } } },
  iu: '/19968336/preroll',
  custParams: { section: 'sports' }   // Optional: extra GAM targeting
});
```

### `TNEVideo.buildAdpodVideoUrl(params)`

Build a GAM VAST tag URL for ad pods (long-form video with multiple ad breaks).

```javascript
TNEVideo.buildAdpodVideoUrl({
  iu: '/19968336/longform',
  descriptionUrl: 'https://example.com/video-page',
  callback: function (err, vastUrl) {
    if (vastUrl) player.src(vastUrl);
  }
});
```

---

## Troubleshooting

### Enable Debug Mode

```javascript
TNEVideo.init({ debug: true, ... });
```

This logs all activity to the browser console with `[TNEVideo]` prefix.

### Common Issues

**"serverUrl is required"**
You didn't pass `serverUrl` to `TNEVideo.init()`.

**No bids returned**
- Check that your bidder params are correct (placementId, accountId, etc.)
- Verify the Catalyst server is reachable: `curl https://pbs.yourdomain.com/status`
- Check the browser Network tab for the `/openrtb2/auction` request and response
- Ensure your publisher ID is registered in Catalyst

**GAM not showing ads**
- Verify line items exist with `hb_pb` targeting
- Check that key-values are registered in GAM (Inventory > Key-values)
- For video: ensure the VAST redirect creative uses `%%PATTERN:hb_uuid%%`
- For banner: ensure the Universal Creative HTML is correctly pasted

**Video player not playing ads**
- Verify `buildVideoUrl()` returns a non-null URL
- Check that the video ad unit has `context: 'instream'`
- Test the VAST URL directly in a [VAST Inspector](https://vastinspector.com)

### Checking Bid Response

Open browser DevTools > Network, find the POST to `/openrtb2/auction`, and inspect the response. Each bid should have:

```json
{
  "ext": {
    "prebid": {
      "targeting": {
        "hb_pb": "5.00",
        "hb_bidder": "tne_appnexus",
        "hb_format": "video",
        "hb_adomain": "ford.com",
        "hb_uuid": "16c887cf-..."
      }
    }
  }
}
```

If `targeting` is empty or missing, the issue is server-side. If targeting is present but GAM isn't matching, the issue is in GAM line item setup.
