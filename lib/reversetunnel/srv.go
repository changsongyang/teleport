/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reversetunnel

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gravitational/teleport"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/limiter"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/sshutils"
	"github.com/gravitational/teleport/lib/utils"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/configure/cstrings"
	"github.com/gravitational/roundtrip"
	"github.com/gravitational/trace"
	"github.com/mailgun/oxy/forward"
	"golang.org/x/crypto/ssh"
)

// RemoteSite represents remote teleport site that can be accessed via
// teleport tunnel or directly by proxy
type RemoteSite interface {
	// ConnectToServer allows to SSH into remote teleport server
	ConnectToServer(addr, user string, auth []ssh.AuthMethod) (*ssh.Client, error)
	// DialServer dials teleport server and returns connection
	DialServer(addr string) (net.Conn, error)
	// Dial dials any address withing reach of remote site's servers
	Dial(network, addr string) (net.Conn, error)
	// GetLastConnected returns last time the remote site was seen connected
	GetLastConnected() time.Time
	// GetName returns site name (identified by authority domain's name)
	GetName() string
	// GetStatus returns status of this site (either offline or connected)
	GetStatus() string
	// GetClient returns client connected to remote auth server
	GetClient() (auth.ClientI, error)
	// GetHangoutInfo returns hangout info (used only if the site is in hangout mode
	GetHangoutInfo() (*HangoutSiteInfo, error)
}

type HangoutSiteInfo struct {
	HostKey  *services.CertAuthority
	OSUser   string
	AuthPort string
	NodePort string
}

// Server represents server connected to one or many remote sites
type Server interface {
	// GetSites returns a list of connected remote sites
	GetSites() []RemoteSite
	// GetSite returns remote site this node belongs to
	GetSite(name string) (RemoteSite, error)
	// FindSimilarSite returns site that matches domain name
	FindSimilarSite(name string) (RemoteSite, error)
	// Start starts server
	Start() error
	// Wait waits for server to close all outstanding operations
	Wait()
}

type server struct {
	sync.RWMutex

	localAuth       auth.ClientI
	hostCertChecker ssh.CertChecker
	userCertChecker ssh.CertChecker
	l               net.Listener
	srv             *sshutils.Server
	timeout         time.Duration

	tunnelSites []*tunnelSite
	directSites []*directSite
}

// ServerOption sets reverse tunnel server options
type ServerOption func(s *server)

// ServerTimeout sets server timeout for read and write operations
func ServerTimeout(duration time.Duration) ServerOption {
	return func(s *server) {
		s.timeout = duration
	}
}

// DirectSite instructs server to proxy access to this site not using
// reverse tunnel
func DirectSite(domainName string, clt auth.ClientI) ServerOption {
	return func(s *server) {
		s.directSites = append(s.directSites, newDirectSite(domainName, clt))
	}
}

// NewServer returns an unstarted server
func NewServer(addr utils.NetAddr, hostSigners []ssh.Signer,
	clt auth.ClientI, limiter *limiter.Limiter, opts ...ServerOption) (Server, error) {
	srv := &server{
		directSites: []*directSite{},
		tunnelSites: []*tunnelSite{},
		localAuth:   clt,
	}

	for _, o := range opts {
		o(srv)
	}
	if srv.timeout == 0 {
		srv.timeout = teleport.DefaultServerTimeout
	}

	s, err := sshutils.NewServer(
		addr,
		srv,
		hostSigners,
		sshutils.AuthMethods{
			PublicKey: srv.keyAuth,
		},
		limiter,
	)
	if err != nil {
		return nil, err
	}
	srv.hostCertChecker = ssh.CertChecker{IsAuthority: srv.isHostAuthority}
	srv.userCertChecker = ssh.CertChecker{IsAuthority: srv.isUserAuthority}
	srv.srv = s
	return srv, nil
}

func (s *server) Wait() {
	s.srv.Wait()
}

func (s *server) Addr() string {
	return s.srv.Addr()
}

func (s *server) Start() error {
	return s.srv.Start()
}

func (s *server) Close() error {
	return s.srv.Close()
}

func (s *server) HandleNewChan(conn net.Conn, sconn *ssh.ServerConn, nch ssh.NewChannel) {
	log.Infof("got new channel request: %v", nch.ChannelType())
	switch nch.ChannelType() {
	case chanHeartbeat:
		log.Infof("got heartbeat request from agent: %v", sconn)
		var site *tunnelSite
		var err error

		switch sconn.Permissions.Extensions[extCertType] {
		case extCertTypeHost:
			site, err = s.upsertRegularSite(conn, sconn)
		case extCertTypeUser:
			site, err = s.upsertHangoutSite(conn, sconn)
		default:
			err = trace.Wrap(
				teleport.BadParameter(
					"certType", "can't retrieve certificate type"))
		}

		if err != nil {
			log.Errorf("failed to upsert site: %v", err)
			nch.Reject(ssh.ConnectionFailed, "failed to upsert site")
			return
		}

		ch, req, err := nch.Accept()
		if err != nil {
			log.Errorf("failed to accept channel: %v", err)
			sconn.Close()
			return
		}
		go site.handleHeartbeat(ch, req)
	}
}

// isHostAuthority is called during checking the client key, to see if the signing
// key is the real host CA authority key.
func (s *server) isHostAuthority(auth ssh.PublicKey) bool {
	keys, err := s.getTrustedCAKeys(services.HostCA)
	if err != nil {
		log.Errorf("failed to retrieve trusted keys, err: %v", err)
		return false
	}
	for _, k := range keys {
		if sshutils.KeysEqual(k, auth) {
			return true
		}
	}
	return false
}

// isUserAuthority is called during checking the client key, to see if the signing
// key is the real user CA authority key.
func (s *server) isUserAuthority(auth ssh.PublicKey) bool {
	keys, err := s.getTrustedCAKeys(services.UserCA)
	if err != nil {
		log.Errorf("failed to retrieve trusted keys, err: %v", err)
		return false
	}
	for _, k := range keys {
		if sshutils.KeysEqual(k, auth) {
			return true
		}
	}
	return false
}

func (s *server) getTrustedCAKeys(CertType services.CertAuthType) ([]ssh.PublicKey, error) {
	cas, err := s.localAuth.GetCertAuthorities(CertType)
	if err != nil {
		return nil, err
	}
	out := []ssh.PublicKey{}
	for _, ca := range cas {
		checkers, err := ca.Checkers()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		out = append(out, checkers...)
	}
	return out, nil
}

func (s *server) checkTrustedKey(CertType services.CertAuthType, domainName string, key ssh.PublicKey) error {
	cas, err := s.localAuth.GetCertAuthorities(CertType)
	if err != nil {
		return trace.Wrap(err)
	}
	for _, ca := range cas {
		if ca.DomainName != domainName {
			continue
		}
		checkers, err := ca.Checkers()
		if err != nil {
			return trace.Wrap(err)
		}
		for _, checker := range checkers {
			if sshutils.KeysEqual(key, checker) {
				return nil
			}
		}
	}
	return trace.Wrap(&teleport.NotFoundError{
		Message: fmt.Sprintf("authority domain %v not found or has no mathching keys", domainName)})
}

func (s *server) keyAuth(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	logger := log.WithFields(log.Fields{
		"remote": conn.RemoteAddr(),
		"user":   conn.User(),
	})

	logger.Infof("auth attempt with key %v", key.Type())

	cert, ok := key.(*ssh.Certificate)
	if !ok {
		logger.Warningf("server doesn't support provided key type")
		return nil, trace.Wrap(
			teleport.BadParameter("key", "server doesn't support provided key type"))
	}

	switch cert.CertType {
	case ssh.HostCert:
		authDomain, ok := cert.Extensions[utils.CertExtensionAuthority]
		if !ok || authDomain == "" {
			err := trace.Wrap(teleport.BadParameter("domainName", "missing authority domain"))
			logger.Warningf("failed authenticate host, err: %v", err)
			return nil, err
		}
		err := s.hostCertChecker.CheckHostKey(conn.User(), conn.RemoteAddr(), key)
		if err != nil {
			logger.Warningf("failed authenticate host, err: %v", err)
			return nil, trace.Wrap(err)
		}
		if err := s.hostCertChecker.CheckCert(conn.User(), cert); err != nil {
			logger.Warningf("failed to authenticate host err: %v", err)
			return nil, trace.Wrap(err)
		}
		// this fixes possible injection attack
		// when we have 2 trusted remote sites, and one can simply
		// pose as another. so we have to check that authority
		// matches by some other way (in absence of x509 chains)
		if err := s.checkTrustedKey(services.HostCA, authDomain, cert.SignatureKey); err != nil {
			logger.Warningf("this claims to be signed as authDomain %v, but no matching signing keys found")
			return nil, trace.Wrap(err)
		}
		return &ssh.Permissions{
			Extensions: map[string]string{
				extHost:      conn.User(),
				extCertType:  extCertTypeHost,
				extAuthority: authDomain,
			},
		}, nil
	case ssh.UserCert:
		_, err := s.userCertChecker.Authenticate(conn, key)
		if err != nil {
			logger.Warningf("failed to authenticate user, err: %v", err)
			return nil, err
		}

		if err := s.userCertChecker.CheckCert(conn.User(), cert); err != nil {
			logger.Warningf("failed to authenticate user err: %v", err)
			return nil, trace.Wrap(err)
		}

		return &ssh.Permissions{
			Extensions: map[string]string{
				extHost:     conn.User(),
				extCertType: extCertTypeUser,
			},
		}, nil
	default:
		return nil, trace.Wrap(
			teleport.BadParameter("cert",
				fmt.Sprintf("unsupported cert type: %v", cert.CertType)))
	}
}

func (s *server) upsertRegularSite(conn net.Conn, sshConn *ssh.ServerConn) (*tunnelSite, error) {
	domainName := sshConn.Permissions.Extensions[extAuthority]
	if !cstrings.IsValidDomainName(domainName) {
		return nil, trace.Wrap(teleport.BadParameter(
			"authDomain", fmt.Sprintf("'%v' is a bad domain name", domainName)))
	}
	var site *tunnelSite
	for _, st := range s.tunnelSites {
		if st.domainName == domainName {
			site = st
			break
		}
	}
	log.Infof("found authority domain: %v", domainName)

	s.Lock()
	defer s.Unlock()

	var err error
	if site != nil {
		if err := site.addConn(conn, sshConn); err != nil {
			return nil, trace.Wrap(err)
		}
	} else {
		site, err = newRemoteSite(s, domainName)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		if err := site.addConn(conn, sshConn); err != nil {
			return nil, trace.Wrap(err)
		}
		s.tunnelSites = append(s.tunnelSites, site)
	}
	return site, nil
}

func (s *server) upsertHangoutSite(conn net.Conn, sshConn ssh.Conn) (*tunnelSite, error) {
	s.Lock()
	defer s.Unlock()

	hangoutID := sshConn.User()
	for _, st := range s.tunnelSites {
		if st.domainName == hangoutID {
			return nil, trace.Errorf("Hangout ID is already used")
		}
	}

	site, err := newRemoteSite(s, hangoutID)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	err = site.addConn(conn, sshConn)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	clt, err := site.GetClient()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	hangoutCertAuthorities, err := clt.GetCertAuthorities(services.HostCA)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if len(hangoutCertAuthorities) != 1 {
		return nil, trace.Errorf("can't retrieve hangout Certificate Authority")
	}
	site.hangoutInfo = &HangoutSiteInfo{
		HostKey: hangoutCertAuthorities[0],
	}

	proxyUserCertAuthorities, err := s.localAuth.GetCertAuthorities(services.UserCA)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	for _, ca := range proxyUserCertAuthorities {
		err := clt.UpsertCertAuthority(*ca, 0)
		if err != nil {
			return nil, trace.Wrap(err)
		}
	}

	// receiving hangoutInfo using sessions just as storage
	sess, err := clt.GetSessions()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if len(sess) != 1 {
		return nil, trace.Wrap(&teleport.NotFoundError{
			Message: fmt.Sprintf("hangout %v not found", hangoutID),
		})
	}
	hangoutInfo, err := utils.UnmarshalHangoutInfo(sess[0].ID)
	if err != nil {
		return nil, err
	}
	site.domainName = hangoutInfo.HangoutID

	site.hangoutInfo.OSUser = hangoutInfo.OSUser
	site.hangoutInfo.AuthPort = hangoutInfo.AuthPort
	site.hangoutInfo.NodePort = hangoutInfo.NodePort

	s.tunnelSites = append(s.tunnelSites, site)
	return site, nil
}

func (s *server) GetSites() []RemoteSite {
	s.RLock()
	defer s.RUnlock()
	out := make([]RemoteSite, 0, len(s.tunnelSites)+len(s.directSites))
	for i := range s.tunnelSites {
		out = append(out, s.tunnelSites[i])
	}
	for i := range s.directSites {
		out = append(out, s.directSites[i])
	}
	return out
}

func (s *server) GetSite(domainName string) (RemoteSite, error) {
	s.RLock()
	defer s.RUnlock()
	for i := range s.tunnelSites {
		if s.tunnelSites[i].domainName == domainName {
			return s.tunnelSites[i], nil
		}
	}

	for i := range s.directSites {
		if s.directSites[i].domainName == domainName {
			return s.directSites[i], nil
		}
	}
	return nil, trace.Wrap(teleport.NotFound(
		fmt.Sprintf("site '%v' not found", domainName)))
}

// FindSimilarSite finds the site that is the most similar to domain.
// Returns nil if no sites with such domain name.
func (s *server) FindSimilarSite(domainName string) (RemoteSite, error) {
	sites := s.GetSites()

	s.RLock()
	defer s.RUnlock()

	result := -1
	resultSimilarity := 1

	domainName1 := strings.Split(domainName, ".")
	log.Infof("Find matching domain: %v", domainName)

	for i, site := range sites {
		log.Infof(site.GetName())
		domainName2 := strings.Split(site.GetName(), ".")
		similarity := 0
		for j := 1; (j <= len(domainName1)) && (j <= len(domainName2)); j++ {
			if domainName1[len(domainName1)-j] != domainName2[len(domainName2)-j] {
				break
			}
			similarity++
		}
		if (similarity > resultSimilarity) || (result == -1) {
			result = i
			resultSimilarity = similarity
		}
	}

	if result != -1 {
		return sites[result], nil
	}
	return nil, trace.Wrap(teleport.NotFound(
		fmt.Sprintf("no site matching '%v' found", domainName)))
}

type remoteConn struct {
	sshConn ssh.Conn
	conn    net.Conn
	invalid int32
	log     *log.Entry
	counter int32
}

func (rc *remoteConn) setDeadline(d time.Duration) {
	atomic.AddInt32(&rc.counter, 1)
	rc.conn.SetDeadline(time.Now().Add(d))
}

func (rc *remoteConn) resetDeadline() {
	// this is to ensure that only last request wil reset the deadline
	// after setting it, otherwise we may reset deadline for
	// a concurrent waiting request. With this logic, we will just
	// make it wait slightly longer
	val := atomic.AddInt32(&rc.counter, -1)
	if val == 0 {
		rc.log.Infof("COUNTER(%v): reseting deadline...", val)
		rc.conn.SetDeadline(time.Time{})
	} else {
		rc.log.Infof("COUNTER(%v): still have waiters", val)
	}
}

func (rc *remoteConn) Close() error {
	return rc.sshConn.Close()
}

func (rc *remoteConn) markInvalid() {
	rc.log.Infof("%v is marked is invalid", rc.conn.RemoteAddr())
	atomic.StoreInt32(&rc.invalid, 1)
}

func (rc *remoteConn) isInvalid() bool {
	return atomic.LoadInt32(&rc.invalid) == 1
}

func newRemoteConn(log *log.Entry, conn net.Conn, sshConn ssh.Conn) (*remoteConn, error) {
	return &remoteConn{
		sshConn: sshConn,
		conn:    conn,
		log:     log,
	}, nil
}

func newRemoteSite(srv *server, domainName string) (*tunnelSite, error) {
	remoteSite := &tunnelSite{
		srv:        srv,
		domainName: domainName,
		log: log.WithFields(log.Fields{
			teleport.Component: teleport.ComponentReverseTunnel,
			teleport.ComponentFields: map[string]string{
				"domainName": domainName,
				"side":       "server",
			},
		}),
	}
	// transport uses connection do dial out to the remote address
	remoteSite.transport = &http.Transport{
		Dial: remoteSite.dialAccessPoint,
	}
	clt, err := auth.NewClient(
		"http://stub:0",
		roundtrip.HTTPClient(&http.Client{
			Transport: remoteSite.transport,
		}))
	if err != nil {
		return nil, trace.Wrap(err)
	}
	remoteSite.clt = clt
	return remoteSite, nil
}

// tunnelSite is a site accessed via SSH reverse tunnel that established
// between proxy and remote site
type tunnelSite struct {
	sync.Mutex

	log         *log.Entry
	domainName  string
	connections []*remoteConn
	lastUsed    int
	lastActive  time.Time
	srv         *server

	transport *http.Transport
	clt       *auth.Client

	hangoutInfo *HangoutSiteInfo
}

func (s *tunnelSite) GetClient() (auth.ClientI, error) {
	return s.clt, nil
}

func (s *tunnelSite) String() string {
	return fmt.Sprintf("remoteSite(%v)", s.domainName)
}

func (s *tunnelSite) nextConn() (*remoteConn, error) {
	s.Lock()
	defer s.Unlock()

	for {
		if len(s.connections) == 0 {
			return nil, trace.Wrap(
				&teleport.NotFoundError{
					Message: "no active connections"})
		}
		s.lastUsed = (s.lastUsed + 1) % len(s.connections)
		remoteConn := s.connections[s.lastUsed]
		if !remoteConn.isInvalid() {
			s.log.Infof("return connection %v", s.lastUsed)
			return remoteConn, nil
		}
		s.connections = append(s.connections[:s.lastUsed], s.connections[s.lastUsed+1:]...)
		s.lastUsed = 0
		go remoteConn.Close()
	}
}

func (s *tunnelSite) addConn(conn net.Conn, sshConn ssh.Conn) error {
	remoteConn, err := newRemoteConn(s.log, conn, sshConn)
	if err != nil {
		return trace.Wrap(err)
	}
	s.Lock()
	defer s.Unlock()
	s.connections = append(s.connections, remoteConn)
	s.lastUsed = 0
	return nil
}

func (s *tunnelSite) GetStatus() string {
	diff := time.Now().Sub(s.lastActive)
	if diff > 2*heartbeatPeriod {
		return RemoteSiteStatusOffline
	}
	return RemoteSiteStatusOnline
}

func (s *tunnelSite) handleHeartbeat(ch ssh.Channel, reqC <-chan *ssh.Request) {
	go func() {
		for {
			req := <-reqC
			if req == nil {
				s.log.Infof("agent disconnected")
				return
			}
			s.log.Debugf("ping")
			s.lastActive = time.Now()
		}
	}()
}

func (s *tunnelSite) GetName() string {
	return s.domainName
}

func (s *tunnelSite) GetLastConnected() time.Time {
	return s.lastActive
}

func (s *tunnelSite) timeout() time.Duration {
	return s.srv.timeout
}

func (s *tunnelSite) ConnectToServer(server, user string, auth []ssh.AuthMethod) (*ssh.Client, error) {
	s.log.Infof("ConnectToServer(server=%v, user=%v)", server, user)
	remoteConn, err := s.nextConn()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	remoteConn.setDeadline(s.timeout())
	defer remoteConn.resetDeadline()
	ch, _, err := remoteConn.sshConn.OpenChannel(chanTransport, nil)
	if err != nil {
		remoteConn.markInvalid()
		return nil, trace.Wrap(err)
	}
	// ask remote channel to dial
	dialed, err := ch.SendRequest(chanTransportDialReq, true, []byte(server))
	if err != nil {
		remoteConn.markInvalid()
		return nil, trace.Wrap(err)
	}
	if !dialed {
		return nil, trace.Errorf("remote server %v is not available", server)
	}
	transportConn := utils.NewChConn(remoteConn.sshConn, ch)
	conn, chans, reqs, err := ssh.NewClientConn(
		transportConn, server,
		&ssh.ClientConfig{
			User: user,
			Auth: auth,
		})
	if err != nil {
		s.log.Infof("connectToServer %v", err)
		return nil, trace.Wrap(err)
	}
	return ssh.NewClient(conn, chans, reqs), nil
}

func (s *tunnelSite) tryDialAccessPoint(network, addr string) (net.Conn, error) {
	s.log.Infof("tryDialAccessPoint(net=%v, addr=%v)", network, addr)
	remoteConn, err := s.nextConn()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	remoteConn.setDeadline(s.timeout())
	defer remoteConn.resetDeadline()

	ch, _, err := remoteConn.sshConn.OpenChannel(chanAccessPoint, nil)
	if err != nil {
		remoteConn.markInvalid()
		s.log.Infof("%v marking connection invalid, conn err: %v", remoteConn.conn.RemoteAddr(), err)
		return nil, trace.Wrap(err)
	}
	return utils.NewChConn(remoteConn.sshConn, ch), nil
}

func (s *tunnelSite) dialAccessPoint(network, addr string) (net.Conn, error) {
	s.log.Infof("dialAccessPoint(net=%v, addr=%v)", network, addr)

	for {
		conn, err := s.tryDialAccessPoint(network, addr)
		if err != nil {
			if teleport.IsNotFound(err) {
				return nil, trace.Wrap(err)
			}
			continue
		}
		return conn, nil
	}
}

func (s *tunnelSite) tryDial(net, addr string) (net.Conn, error) {
	s.log.Infof("tryDial(net=%v, addr=%v)", net, addr)
	remoteConn, err := s.nextConn()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	remoteConn.setDeadline(s.timeout())
	defer remoteConn.resetDeadline()
	var ch ssh.Channel
	ch, _, err = remoteConn.sshConn.OpenChannel(chanTransport, nil)
	if err != nil {
		remoteConn.markInvalid()
		return nil, trace.Wrap(err)
	}
	// ask remote channel to dial
	var dialed bool
	dialed, err = ch.SendRequest(chanTransportDialReq, true, []byte(addr))
	if err != nil {
		remoteConn.markInvalid()
		return nil, trace.Wrap(err)
	}
	if !dialed {
		remoteConn.markInvalid()
		return nil, trace.Wrap(
			teleport.ConnectionProblem(
				fmt.Sprintf("remote server %v is not available", addr), nil))
	}
	return utils.NewChConn(remoteConn.sshConn, ch), nil
}

func (s *tunnelSite) Dial(net string, addr string) (net.Conn, error) {
	s.log.Infof("Dial(net=%v, addr=%v)", net, addr)
	for {
		conn, err := s.tryDial(net, addr)
		if err != nil {
			s.log.Infof("got error: %v", err)
			// we interpret it as a "out of connections and will try again"
			if teleport.IsNotFound(err) {
				return nil, trace.Wrap(err)
			}
			continue
		}
		return conn, nil
	}
}

func (s *tunnelSite) DialServer(addr string) (net.Conn, error) {
	s.log.Infof("DialServer(addr=%v)", addr)
	clt, err := s.GetClient()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	knownServers, err := clt.GetServers()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	server, err := findServer(addr, knownServers)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return s.Dial("tcp", server.Addr)
}

func (s *tunnelSite) handleAuthProxy(w http.ResponseWriter, r *http.Request) {
	s.log.Infof("handleAuthProxy()")

	fwd, err := forward.New(forward.RoundTripper(s.transport), forward.Logger(s.log))
	if err != nil {
		roundtrip.ReplyJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	r.URL.Scheme = "http"
	r.URL.Host = "stub"
	fwd.ServeHTTP(w, r)
}

func (s *tunnelSite) GetHangoutInfo() (*HangoutSiteInfo, error) {
	if s.hangoutInfo == nil {
		return nil, trace.Errorf("No hangout info")
	}
	return s.hangoutInfo, nil
}

const (
	extHost         = "host@teleport"
	extCertType     = "certtype@teleport"
	extAuthority    = "auth@teleport"
	extCertTypeHost = "host"
	extCertTypeUser = "user"
)
