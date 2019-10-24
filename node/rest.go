package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service"
)

const (
	UrlSystem   = "/system"
	UrlStats    = "/stats"
	UrlChain    = "/chain"
	ParamNID    = "nid"
	UrlChainRes = "/:" + ParamNID
)

type Rest struct {
	n *Node
}

type SystemView struct {
	BuildVersion string `json:"buildVersion"`
	BuildTags    string `json:"buildTags"`
	Setting      struct {
		Address       string `json:"address"`
		P2PAddr       string `json:"p2p"`
		P2PListenAddr string `json:"p2pListen"`
		RPCAddr       string `json:"rpcAddr"`
		RPCDump       bool   `json:"rpcDump"`
	} `json:"setting"`
	Config interface{} `json:"config"`
}

type StatsView struct {
	Chains    []map[string]interface{} `json:"chains"`
	Timestamp time.Time                `json:"timestamp"`
}

type ChainView struct {
	NID       common.HexInt32 `json:"nid"`
	Channel   string          `json:"channel"`
	State     string          `json:"state"`
	Height    int64           `json:"height"`
	LastError string          `json:"lastError"`
}

type ChainInspectView struct {
	*ChainView
	GenesisTx json.RawMessage `json:"genesisTx"`
	Config    *ChainConfig    `json:"config"`
	// TODO [TBD] define structure each module for inspect
	Module map[string]interface{} `json:"module"`
}

type ChainConfig struct {
	DBType           string `json:"dbType"`
	SeedAddr         string `json:"seedAddress"`
	Role             uint   `json:"role"`
	ConcurrencyLevel int    `json:"concurrencyLevel,omitempty"`
	NormalTxPoolSize int    `json:"normalTxPool,omitempty"`
	PatchTxPoolSize  int    `json:"patchTxPool,omitempty"`
	MaxBlockTxBytes  int    `json:"maxBlockTxBytes,omitempty"`
	NodeCache        string `json:"nodeCache,omitempty"`
	Channel          string `json:"channel"`
	SecureSuites     string `json:"secureSuites"`
	SecureAeads      string `json:"secureAeads"`
}

type ChainImportParam struct {
	DBPath string `json:"dbPath"`
	Height int64  `json:"height"`
}

type ConfigureParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TODO [TBD]move to module.Chain ?
type LastErrorReportor interface {
	LastError() error
}

func NewChainView(c *Chain) *ChainView {
	v := &ChainView{
		NID:     common.HexInt32{Value: int32(c.NID())},
		Channel: c.Channel(),
		State:   c.State(),
	}
	if r, ok := c.Chain.(LastErrorReportor); ok && r.LastError() != nil {
		v.LastError = r.LastError().Error()
	}

	if bm := c.BlockManager(); bm != nil {
		if b, err := bm.GetLastBlock(); err == nil {
			v.Height = b.Height()
		}
	}
	return v
}

type InspectFunc func(c module.Chain, informal bool) map[string]interface{}

var (
	inspectFuncs = make(map[string]InspectFunc)
)

func NewChainInspectView(c *Chain) *ChainInspectView {
	v := &ChainInspectView{
		ChainView: NewChainView(c),
		GenesisTx: c.Genesis(),
		Config:    NewChainConfig(c.cfg),
	}
	return v
}

func NewChainConfig(cfg *chain.Config) *ChainConfig {
	v := &ChainConfig{
		DBType:           cfg.DBType,
		SeedAddr:         cfg.SeedAddr,
		Role:             cfg.Role,
		ConcurrencyLevel: cfg.ConcurrencyLevel,
		NormalTxPoolSize: cfg.NormalTxPoolSize,
		PatchTxPoolSize:  cfg.PatchTxPoolSize,
		MaxBlockTxBytes:  cfg.MaxBlockTxBytes,
		NodeCache:        cfg.NodeCache,
		Channel:          cfg.Channel,
		SecureSuites:     cfg.SecureSuites,
		SecureAeads:      cfg.SecureAeads,
	}
	return v
}

func RegisterInspectFunc(name string, f InspectFunc) error {
	if _, ok := inspectFuncs[name]; ok {
		return fmt.Errorf("already exist function name:%s", name)
	}
	inspectFuncs[name] = f
	return nil
}

func RegisterRest(n *Node) {
	r := Rest{n}
	ag := n.srv.AdminEchoGroup()
	r.RegisterChainHandlers(ag.Group(UrlChain))
	r.RegisterSystemHandlers(ag.Group(UrlSystem))

	r.RegisterChainHandlers(n.cliSrv.e.Group(UrlChain))
	r.RegisterSystemHandlers(n.cliSrv.e.Group(UrlSystem))
	r.RegisterStatsHandlers(n.cliSrv.e.Group(UrlStats))

	_ = RegisterInspectFunc("metrics", metric.Inspect)
	_ = RegisterInspectFunc("network", network.Inspect)
	_ = RegisterInspectFunc("service", service.Inspect)
}

func (r *Rest) RegisterChainHandlers(g *echo.Group) {
	g.GET("", r.GetChains)
	g.POST("", r.JoinChain)

	g.GET(UrlChainRes, r.GetChain, r.ChainInjector)
	g.DELETE(UrlChainRes, r.LeaveChain, r.ChainInjector)
	// TODO update chain configuration ex> Channel, Seed, ConcurrencyLevel ...
	// g.PUT(UrlChainRes, r.UpdateChain, r.ChainInjector)
	g.POST(UrlChainRes+"/start", r.StartChain, r.ChainInjector)
	g.POST(UrlChainRes+"/stop", r.StopChain, r.ChainInjector)
	g.POST(UrlChainRes+"/reset", r.ResetChain, r.ChainInjector)
	g.POST(UrlChainRes+"/verify", r.VerifyChain, r.ChainInjector)
	g.POST(UrlChainRes+"/import", r.ImportChain, r.ChainInjector)
	g.GET(UrlChainRes+"/configure", r.GetChainConfig, r.ChainInjector)
	g.POST(UrlChainRes+"/configure", r.ConfigureChain, r.ChainInjector)
}

func (r *Rest) ChainInjector(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		var c *Chain
		p := ctx.Param(ParamNID)
		if nid, err := strconv.ParseInt(p, 0, 32); err == nil {
			c = r.n.GetChain(int(nid))
		}
		if c == nil {
			c = r.n.GetChainByChannel(p)
		}

		if c == nil {
			return ctx.String(http.StatusNotFound,
				fmt.Sprintf("Chain(%s: nid or channel) not found", p))
		}
		ctx.Set("chain", c)
		return next(ctx)
	}
}

func (r *Rest) GetChains(ctx echo.Context) error {
	l := make([]*ChainView, 0)
	for _, c := range r.n.GetChains() {
		v := NewChainView(c)
		l = append(l, v)
	}
	return ctx.JSON(http.StatusOK, l)
}

func GetJsonMultipart(ctx echo.Context, ptr interface{}) error {
	jsonStr := ctx.FormValue("json")
	if err := json.Unmarshal([]byte(jsonStr), ptr); err != nil {
		return err
	}
	return nil
}

func GetFileMultipart(ctx echo.Context, fieldname string) ([]byte, error) {
	ff, err := ctx.FormFile(fieldname)
	if err != nil {
		return nil, err
	}
	f, err := ff.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Rest) JoinChain(ctx echo.Context) error {
	p := &ChainConfig{}

	if err := GetJsonMultipart(ctx, p); err != nil {
		return errors.Wrap(err, "fail to get 'json' from multipart")
	}

	genesis, err := GetFileMultipart(ctx, "genesisZip")
	if err != nil {
		return errors.Wrap(err, "fail to get 'genesisZip' from multipart")
	}

	c, err := r.n.JoinChain(p, genesis)
	if err != nil {
		if we, ok := err.(errors.Unwrapper); ok {
			switch we.Unwrap() {
			case ErrAlreadyExists:
				return ctx.String(http.StatusConflict, err.Error())
			}
		}
		return errors.Wrap(err, "fail to join")
	}
	return ctx.String(http.StatusOK, fmt.Sprintf("%#x", c.NID()))
}

var (
	defaultJsonTemplate = NewJsonTemplate("default")
)

func (r *Rest) GetChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	v := NewChainInspectView(c)

	informal, _ := strconv.ParseBool(ctx.QueryParam("informal"))
	v.Module = make(map[string]interface{})
	for name, f := range inspectFuncs {
		if m := f(c, informal); m != nil {
			v.Module[name] = m
		}
	}
	format := ctx.QueryParam("format")
	if format != "" {
		return defaultJsonTemplate.Response(format, v, ctx.Response())
	}
	return ctx.JSON(http.StatusOK, v)
}

func (r *Rest) LeaveChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.LeaveChain(c.NID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) StartChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.StartChain(c.NID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) StopChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.StopChain(c.NID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) ResetChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.ResetChain(c.NID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) VerifyChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.VerifyChain(c.NID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) ImportChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	param := &ChainImportParam{}
	if err := ctx.Bind(param); err != nil {
		return echo.ErrBadRequest
	}
	if err := r.n.ImportChain(c.NID(), param.DBPath, param.Height); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) GetChainConfig(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	return ctx.JSON(http.StatusOK, NewChainConfig(c.cfg))
}

func (r *Rest) ConfigureChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	p := &ConfigureParam{}
	if err := ctx.Bind(p); err != nil {
		return err
	}
	if err := r.n.ConfigureChain(c.NID(), p.Key, p.Value); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RegisterSystemHandlers(g *echo.Group) {
	g.GET("", r.GetSystem)
	g.GET("/configure", r.GetSystemConfig)
	g.POST("/configure", r.ConfigureSystem)
}

func (r *Rest) GetSystem(ctx echo.Context) error {
	v := &SystemView{
		BuildVersion: r.n.cfg.BuildVersion,
		BuildTags:    r.n.cfg.BuildTags,
	}
	v.Setting.Address = r.n.w.Address().String()
	v.Setting.P2PAddr = r.n.nt.Address()
	v.Setting.P2PListenAddr = r.n.nt.GetListenAddress()
	v.Setting.RPCAddr = r.n.cfg.RPCAddr
	v.Setting.RPCDump = r.n.cfg.RPCDump
	v.Config = r.n.rcfg

	format := ctx.QueryParam("format")
	if format != "" {
		return defaultJsonTemplate.Response(format, v, ctx.Response())
	}
	return ctx.JSON(http.StatusOK, v)
}

func (r *Rest) GetSystemConfig(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, r.n.rcfg)
}

func (r *Rest) ConfigureSystem(ctx echo.Context) error {
	p := &ConfigureParam{}
	if err := ctx.Bind(p); err != nil {
		return err
	}

	if err := r.n.Configure(p.Key, p.Value); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RegisterStatsHandlers(g *echo.Group) {
	g.GET("", r.StreamStats)
}

func (r *Rest) StreamStats(ctx echo.Context) error {
	intervalSec := 1
	param := ctx.QueryParam("interval")
	if param != "" {
		var err error
		intervalSec, err = strconv.Atoi(param)
		if err != nil {
			return err
		}
	}

	streaming := true
	param = ctx.QueryParam("stream")
	if param != "" {
		var err error
		streaming, err = strconv.ParseBool(param)
		if err != nil {
			return err
		}
	}
	// chains := ctx.QueryParam("chains")
	// strings.Split(chains,",")

	resp := ctx.Response()
	resp.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	resp.WriteHeader(http.StatusOK)
	if err := r.ResponseStatsView(resp); err != nil {
		return err
	}
	resp.Flush()

	tick := time.NewTicker(time.Duration(intervalSec) * time.Second)
	for streaming {
		select {
		case <-tick.C:
			if err := r.ResponseStatsView(resp); err != nil {
				return err
			}
			resp.Flush()
		}
	}
	return nil
}

func (r *Rest) ResponseStatsView(resp *echo.Response) error {
	v := StatsView{
		Chains:    make([]map[string]interface{}, 0),
		Timestamp: time.Now(),
	}
	for _, c := range r.n.GetChains() {
		m := metric.Inspect(c, false)
		if c.State() != chain.StateStopped.String() {
			m["nid"] = common.HexInt32{Value: int32(c.NID())}
			m["channel"] = c.Channel()
			v.Chains = append(v.Chains, m)
		}
	}
	if err := json.NewEncoder(resp).Encode(&v); err != nil {
		if EqualsSyscallErrno(err, syscall.EPIPE) {
			// ignore 'write: broken pipe' error
			// close by client
			return nil
		}
		return err
	}
	return nil
}

func EqualsSyscallErrno(err error, sen syscall.Errno) bool {
	if oe, ok := err.(*net.OpError); ok {
		if se, ok := oe.Err.(*os.SyscallError); ok {
			if en, ok := se.Err.(syscall.Errno); ok && en == sen {
				return true
			}
		}
	}
	return false
}

type JsonTemplate struct {
	*template.Template
}

func NewJsonTemplate(name string) *JsonTemplate {
	tmpl := &JsonTemplate{template.New(name)}
	tmpl.Option("missingkey=error")
	tmpl.Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
	})
	return tmpl
}

func (t *JsonTemplate) Response(format string, v interface{}, resp *echo.Response) error {
	nt, err := t.Clone()
	if err != nil {
		return err
	}
	nt, err = nt.Parse(format)
	if err != nil {
		return err
	}
	err = nt.Execute(resp, v)
	if err != nil {
		return err
	}

	// resp.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	resp.Header().Set(echo.HeaderContentType, echo.MIMETextPlain)
	resp.WriteHeader(http.StatusOK)
	return nil
}
