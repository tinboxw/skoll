package memcached

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/cache/local"
)

type Adapter struct {
	addr     string
	fallback *local.LRUCache
}

func NewAdapter(addr string) (*Adapter, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("memcached addr is required")
	}
	cleanAddr := strings.TrimSpace(addr)
	return &Adapter{
		addr:     cleanAddr,
		fallback: local.NewLRUCache(4096),
	}, nil
}

func (a *Adapter) Address() string { return a.addr }

func (a *Adapter) Set(key string, value []byte, ttl time.Duration) {
	if a == nil {
		return
	}
	err := a.setRemote(key, value, ttl)
	if err != nil {
		a.fallback.Set(key, value, ttl)
		return
	}
	a.fallback.Set(key, value, ttl)
}

func (a *Adapter) Get(key string) ([]byte, bool) {
	if a == nil {
		return nil, false
	}
	value, ok, err := a.getRemote(key)
	if err == nil {
		if ok {
			return value, true
		}
		return nil, false
	}
	return a.fallback.Get(key)
}

func (a *Adapter) Delete(key string) {
	if a == nil {
		return
	}
	_ = a.deleteRemote(key)
	a.fallback.Delete(key)
}

func (a *Adapter) setRemote(key string, value []byte, ttl time.Duration) error {
	if !isSafeKey(key) {
		return fmt.Errorf("invalid memcached key")
	}
	conn, err := net.DialTimeout("tcp", a.addr, 300*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(300 * time.Millisecond))

	expiration := int(ttl / time.Second)
	if expiration < 0 {
		expiration = 0
	}
	cmd := fmt.Sprintf("set %s 0 %d %d\r\n", key, expiration, len(value))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return err
	}
	if _, err := conn.Write(value); err != nil {
		return err
	}
	if _, err := conn.Write([]byte("\r\n")); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != "STORED" {
		return fmt.Errorf("memcached set failed: %s", strings.TrimSpace(line))
	}
	return nil
}

func (a *Adapter) getRemote(key string) ([]byte, bool, error) {
	if !isSafeKey(key) {
		return nil, false, fmt.Errorf("invalid memcached key")
	}
	conn, err := net.DialTimeout("tcp", a.addr, 300*time.Millisecond)
	if err != nil {
		return nil, false, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(300 * time.Millisecond))

	if _, err := conn.Write([]byte("get " + key + "\r\n")); err != nil {
		return nil, false, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, false, err
	}
	line = strings.TrimSpace(line)
	if line == "END" {
		return nil, false, nil
	}
	parts := strings.Split(line, " ")
	if len(parts) < 4 || parts[0] != "VALUE" {
		return nil, false, fmt.Errorf("unexpected memcached response: %s", line)
	}
	bytesLen, err := strconv.Atoi(parts[3])
	if err != nil || bytesLen < 0 {
		return nil, false, fmt.Errorf("invalid memcached bytes length: %s", parts[3])
	}
	value := make([]byte, bytesLen+2)
	if _, err := io.ReadFull(reader, value); err != nil {
		return nil, false, err
	}
	endLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(endLine) != "END" {
		return nil, false, fmt.Errorf("unexpected memcached trailer: %s", strings.TrimSpace(endLine))
	}
	return value[:bytesLen], true, nil
}

func (a *Adapter) deleteRemote(key string) error {
	if !isSafeKey(key) {
		return fmt.Errorf("invalid memcached key")
	}
	conn, err := net.DialTimeout("tcp", a.addr, 300*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(300 * time.Millisecond))

	if _, err := conn.Write([]byte("delete " + key + "\r\n")); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(line)
	if trimmed != "DELETED" && trimmed != "NOT_FOUND" {
		return fmt.Errorf("unexpected memcached delete response: %s", trimmed)
	}
	return nil
}

func isSafeKey(key string) bool {
	v := strings.TrimSpace(key)
	if v == "" || len(v) > 250 {
		return false
	}
	return !strings.ContainsAny(v, " \r\n\t")
}
