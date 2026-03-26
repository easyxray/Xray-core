package kicker

import (
	"context"
	"sync"
)

var (
	// sessions 结构: Email -> { ConnectionID -> CancelFunc }
	sessions = make(map[string]map[uint32]context.CancelFunc)
	mutex    sync.Mutex
	counter  uint32
)

// Register 记录连接。在 Dispatcher 层调用。
func Register(email string, ctx context.Context) context.Context {
	if email == "" {
		return ctx
	}

	// 创建可取消的 Context
	newCtx, cancel := context.WithCancel(ctx)

	mutex.Lock()
	if _, ok := sessions[email]; !ok {
		sessions[email] = make(map[uint32]context.CancelFunc)
	}
	id := counter
	counter++
	sessions[email][id] = cancel
	mutex.Unlock()

	// 协程监听：连接结束时自动清理，防止内存泄漏
	go func() {
		<-newCtx.Done()
		mutex.Lock()
		if userSessions, ok := sessions[email]; ok {
			delete(userSessions, id)
			if len(userSessions) == 0 {
				delete(sessions, email)
			}
		}
		mutex.Unlock()
	}()

	return newCtx
}

// Kick 物理断开连接。由 gRPC Handler 调用。
func Kick(email string) {
	mutex.Lock()
	defer mutex.Unlock()
	if userSessions, ok := sessions[email]; ok {
		for _, cancel := range userSessions {
			cancel() // 执行 Context 取消
		}
		// 清空该用户所有记录
		delete(sessions, email)
	}
}
