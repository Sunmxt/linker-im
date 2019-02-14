package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/gomodule/redigo/redis"
	"time"
)

func (g *Gate) sendRoute(rconn redis.Conn, conn *Connection) (int, error) {
	var err error
	var timeout int

	infoKey := g.config.RedisPrefix.Value + "{clientinfo-" + conn.key + "}"
	if err = rconn.Send("HSET", infoKey, "proto", conn.Meta.Proto); err != nil {
		return 0, err
	}
	if err = rconn.Send("HSET", infoKey, "remote", conn.Meta.Remote); err != nil {
		return 1, err
	}
	if err = rconn.Send("HSET", infoKey, "gate", g.ID.String()); err != nil {
		return 2, err
	}
	if conn.Meta.Timeout <= 0 {
		if g.config.RouteTimeout.Value > 0 {
			timeout = int(g.config.RouteTimeout.Value)
		} else {
			timeout = -1
		}
	} else {
		timeout = conn.Meta.Timeout
	}
	if timeout > 0 {
		timeout /= 500
		if timeout < 1 {
			timeout = 1
		}
		if err = rconn.Send("EXPIRE", infoKey, timeout); err != nil {
			return 3, err
		}
		return 4, nil
	}
	return 3, nil
}

func (g *Gate) Routing() {
	var (
		rconn redis.Conn
		err   error
	)
	log.Info0("Start client publishing.")

	count, flushTick := 0, time.Tick(time.Millisecond*1000)
	for {
		rconn = g.Redis.Get()
		count = 0
		g.Hub.Visit(func(key string, conn *Connection) bool {
			cnt, ierr := g.sendRoute(rconn, conn)
			count += cnt
			if ierr != nil {
				err = ierr
				for len(flushTick) > 0 {
					<-flushTick
				}
				<-flushTick
				return false
			}
			return true
		})

	RoutePublish:
		for err == nil {
			if count > 0 {
				if err = rconn.Flush(); err != nil {
					break
				}
				for count > 0 {
					if _, err = rconn.Receive(); err != nil {
						break RoutePublish
					}
					count--
				}
			}

		ConnectionReceive:
			for {
				select {
				case conn := <-g.Hub.sigRoute:
					cnt, ierr := g.sendRoute(rconn, conn)
					count += cnt
					if ierr != nil {
						err = ierr
						break ConnectionReceive
					}
				case <-flushTick:
					break ConnectionReceive
				}
			}
		}
		rconn.Close()
		if err != nil {
			log.Error("Route sending failure: " + err.Error())
		}
	}
}
