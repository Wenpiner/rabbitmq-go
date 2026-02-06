package rabbitmq

import (
	"fmt"
	"sync"
)

// ConnectionState 连接状态
type ConnectionState int

const (
	// StateDisconnected 已断开连接
	StateDisconnected ConnectionState = iota
	// StateConnecting 正在连接
	StateConnecting
	// StateConnected 已连接
	StateConnected
	// StateReconnecting 正在重连
	StateReconnecting
	// StateClosed 已关闭（终态）
	StateClosed
)

// String 返回状态名称
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateReconnecting:
		return "Reconnecting"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// ConsumerState 消费者状态
type ConsumerState int

const (
	// ConsumerIdle 空闲
	ConsumerIdle ConsumerState = iota
	// ConsumerStarting 正在启动
	ConsumerStarting
	// ConsumerRunning 运行中
	ConsumerRunning
	// ConsumerStopping 正在停止
	ConsumerStopping
	// ConsumerStopped 已停止
	ConsumerStopped
)

// String 返回消费者状态名称
func (s ConsumerState) String() string {
	switch s {
	case ConsumerIdle:
		return "Idle"
	case ConsumerStarting:
		return "Starting"
	case ConsumerRunning:
		return "Running"
	case ConsumerStopping:
		return "Stopping"
	case ConsumerStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// Event 状态事件类型
type Event int

const (
	// EventConnect 连接事件
	EventConnect Event = iota
	// EventConnectSuccess 连接成功
	EventConnectSuccess
	// EventConnectFailed 连接失败
	EventConnectFailed
	// EventDisconnect 断开连接
	EventDisconnect
	// EventReconnect 重连
	EventReconnect
	// EventClose 关闭
	EventClose
)

// String 返回事件名称
func (e Event) String() string {
	switch e {
	case EventConnect:
		return "Connect"
	case EventConnectSuccess:
		return "ConnectSuccess"
	case EventConnectFailed:
		return "ConnectFailed"
	case EventDisconnect:
		return "Disconnect"
	case EventReconnect:
		return "Reconnect"
	case EventClose:
		return "Close"
	default:
		return "Unknown"
	}
}

// StateTransition 状态转换定义
type StateTransition struct {
	From  ConnectionState
	Event Event
	To    ConnectionState
}

// StateMachine 状态机
type StateMachine struct {
	mu          sync.RWMutex
	current     ConnectionState
	transitions map[ConnectionState]map[Event]ConnectionState
	listeners   []StateChangeListener
}

// StateChangeListener 状态变化监听器
type StateChangeListener func(from, to ConnectionState, event Event)

// NewStateMachine 创建状态机
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		current:     StateDisconnected,
		transitions: make(map[ConnectionState]map[Event]ConnectionState),
		listeners:   make([]StateChangeListener, 0),
	}
	sm.initTransitions()
	return sm
}

// initTransitions 初始化状态转换规则
func (sm *StateMachine) initTransitions() {
	// 定义所有合法的状态转换
	transitions := []StateTransition{
		// 从断开状态
		{StateDisconnected, EventConnect, StateConnecting},
		{StateDisconnected, EventClose, StateClosed},

		// 从连接中状态
		{StateConnecting, EventConnectSuccess, StateConnected},
		{StateConnecting, EventConnectFailed, StateDisconnected},
		{StateConnecting, EventClose, StateClosed},

		// 从已连接状态
		{StateConnected, EventDisconnect, StateReconnecting},
		{StateConnected, EventClose, StateClosed},

		// 从重连状态
		{StateReconnecting, EventConnectSuccess, StateConnected},
		{StateReconnecting, EventConnectFailed, StateDisconnected},
		{StateReconnecting, EventClose, StateClosed},
	}

	for _, t := range transitions {
		if sm.transitions[t.From] == nil {
			sm.transitions[t.From] = make(map[Event]ConnectionState)
		}
		sm.transitions[t.From][t.Event] = t.To
	}
}

// Current 获取当前状态
func (sm *StateMachine) Current() ConnectionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// CanTransition 检查是否可以进行状态转换
func (sm *StateMachine) CanTransition(event Event) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, ok := sm.transitions[sm.current][event]
	return ok
}

// Transition 执行状态转换
func (sm *StateMachine) Transition(event Event) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	to, ok := sm.transitions[sm.current][event]
	if !ok {
		return fmt.Errorf("invalid transition: %s -> %s", sm.current, event)
	}

	from := sm.current
	sm.current = to

	// 通知所有监听器
	for _, listener := range sm.listeners {
		go listener(from, to, event)
	}

	return nil
}

// OnStateChange 添加状态变化监听器
func (sm *StateMachine) OnStateChange(listener StateChangeListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// IsConnected 检查是否已连接
func (sm *StateMachine) IsConnected() bool {
	return sm.Current() == StateConnected
}

// IsClosed 检查是否已关闭
func (sm *StateMachine) IsClosed() bool {
	return sm.Current() == StateClosed
}

