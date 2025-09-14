package flyreplay

import (
	"strconv"
	
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(FlyReplay{})
	httpcaddyfile.RegisterHandlerDirective("fly_replay", parseCaddyfile)
}

// CaddyModule returns the Caddy module information.
func (FlyReplay) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.fly_replay",
		New: func() caddy.Module { return new(FlyReplay) },
	}
}

// Provision implements caddy.Provisioner.
func (f *FlyReplay) Provision(ctx caddy.Context) error {
	if f.Apps == nil {
		f.Apps = make(map[string]AppConfig)
	}
	
	// Initialize cache if enabled
	if f.EnableCache {
		f.cache = NewPathCache()
	}
	
	// Set default cache TTL if not specified
	if f.CacheTTL == 0 {
		f.CacheTTL = 300 // 5 minutes default
	}
	
	return nil
}

// Validate implements caddy.Validator.
func (f *FlyReplay) Validate() error {
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new FlyReplay.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var f FlyReplay
	err := f.UnmarshalCaddyfile(h.Dispenser)
	return &f, err
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (f *FlyReplay) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	f.Apps = make(map[string]AppConfig)
	
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "enable_cache":
				if !d.NextArg() {
					return d.ArgErr()
				}
				f.EnableCache = d.Val() == "true"
				
			case "cache_dir":
				if !d.NextArg() {
					return d.ArgErr()
				}
				f.CacheDir = d.Val()
				
			case "cache_ttl":
				if !d.NextArg() {
					return d.ArgErr()
				}
				ttl, err := strconv.Atoi(d.Val())
				if err != nil {
					return err
				}
				f.CacheTTL = ttl
				
			case "debug":
				if !d.NextArg() {
					return d.ArgErr()
				}
				f.Debug = d.Val() == "true"
				
			case "apps":
				for d.NextBlock(1) {
					appName := d.Val()
					var app AppConfig
					
					for d.NextBlock(2) {
						switch d.Val() {
						case "domain":
							if !d.NextArg() {
								return d.ArgErr()
							}
							app.Domain = d.Val()
						default:
							return d.Errf("unknown app property: %s", d.Val())
						}
					}
					
					if app.Domain == "" {
						return d.Errf("app %s must have a domain", appName)
					}
					
					f.Apps[appName] = app
				}
				
			default:
				return d.Errf("unknown directive: %s", d.Val())
			}
		}
	}
	
	return nil
}