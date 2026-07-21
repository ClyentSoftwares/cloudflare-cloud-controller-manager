package cfapi

// LoadBalancer is a zone scoped cloudflare load balancer.
type LoadBalancer struct {
	ID             string
	Name           string
	FallbackPoolID string
	DefaultPoolIDs []string
	Proxied        bool
}

// Pool is an account scoped origin pool.
type Pool struct {
	ID        string
	Name      string
	MonitorID string
	Enabled   bool
	Origins   []Origin
}

// Origin is a single member of a pool.
type Origin struct {
	Name    string
	Address string
	Enabled bool
	Weight  float64
}

// Monitor is an account scoped health monitor. Monitors have no name field on
// cloudflare, the description is used as the identifier.
type Monitor struct {
	ID              string
	Description     string
	Type            string
	Method          string
	Path            string
	Port            int64
	ExpectedCodes   string
	Interval        int64
	Timeout         int64
	Retries         int64
	FollowRedirects bool
	AllowInsecure   bool
	ProbeZone       string
	Header          map[string][]string
}
