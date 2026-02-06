package rabbitmq

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "Disconnected"},
		{StateConnecting, "Connecting"},
		{StateConnected, "Connected"},
		{StateReconnecting, "Reconnecting"},
		{StateClosed, "Closed"},
		{ConnectionState(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("ConnectionState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

func TestConsumerState_String(t *testing.T) {
	tests := []struct {
		state    ConsumerState
		expected string
	}{
		{ConsumerIdle, "Idle"},
		{ConsumerStarting, "Starting"},
		{ConsumerRunning, "Running"},
		{ConsumerStopping, "Stopping"},
		{ConsumerStopped, "Stopped"},
		{ConsumerState(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("ConsumerState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

func TestEvent_String(t *testing.T) {
	tests := []struct {
		event    Event
		expected string
	}{
		{EventConnect, "Connect"},
		{EventConnectSuccess, "ConnectSuccess"},
		{EventConnectFailed, "ConnectFailed"},
		{EventDisconnect, "Disconnect"},
		{EventReconnect, "Reconnect"},
		{EventClose, "Close"},
		{Event(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.event.String(); got != tt.expected {
			t.Errorf("Event(%d).String() = %s, want %s", tt.event, got, tt.expected)
		}
	}
}

func TestNewStateMachine(t *testing.T) {
	sm := NewStateMachine()

	if sm.Current() != StateDisconnected {
		t.Errorf("初始状态应为 Disconnected, got %s", sm.Current())
	}

	if sm.IsConnected() {
		t.Error("新建状态机不应处于已连接状态")
	}

	if sm.IsClosed() {
		t.Error("新建状态机不应处于已关闭状态")
	}
}

func TestStateMachine_ValidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		events    []Event
		wantState ConnectionState
	}{
		{
			name:      "连接成功流程",
			events:    []Event{EventConnect, EventConnectSuccess},
			wantState: StateConnected,
		},
		{
			name:      "连接失败流程",
			events:    []Event{EventConnect, EventConnectFailed},
			wantState: StateDisconnected,
		},
		{
			name:      "断开后重连成功",
			events:    []Event{EventConnect, EventConnectSuccess, EventDisconnect, EventConnectSuccess},
			wantState: StateConnected,
		},
		{
			name:      "直接关闭",
			events:    []Event{EventClose},
			wantState: StateClosed,
		},
		{
			name:      "连接中关闭",
			events:    []Event{EventConnect, EventClose},
			wantState: StateClosed,
		},
		{
			name:      "已连接后关闭",
			events:    []Event{EventConnect, EventConnectSuccess, EventClose},
			wantState: StateClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()

			for _, event := range tt.events {
				if err := sm.Transition(event); err != nil {
					t.Errorf("Transition(%s) 失败: %v", event, err)
				}
			}

			if sm.Current() != tt.wantState {
				t.Errorf("最终状态 = %s, want %s", sm.Current(), tt.wantState)
			}
		})
	}
}

func TestStateMachine_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name        string
		setupEvents []Event
		badEvent    Event
	}{
		{
			name:        "断开状态不能直接成功",
			setupEvents: []Event{},
			badEvent:    EventConnectSuccess,
		},
		{
			name:        "断开状态不能断开",
			setupEvents: []Event{},
			badEvent:    EventDisconnect,
		},
		{
			name:        "已连接不能再连接",
			setupEvents: []Event{EventConnect, EventConnectSuccess},
			badEvent:    EventConnect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()

			for _, event := range tt.setupEvents {
				_ = sm.Transition(event)
			}

			if err := sm.Transition(tt.badEvent); err == nil {
				t.Errorf("期望 Transition(%s) 失败，但成功了", tt.badEvent)
			}
		})
	}
}

func TestStateMachine_CanTransition(t *testing.T) {
	sm := NewStateMachine()

	// 初始状态可以连接
	if !sm.CanTransition(EventConnect) {
		t.Error("断开状态应该可以连接")
	}

	// 初始状态不能断开
	if sm.CanTransition(EventDisconnect) {
		t.Error("断开状态不应该可以断开")
	}

	// 转换到连接中
	_ = sm.Transition(EventConnect)

	// 连接中可以成功或失败
	if !sm.CanTransition(EventConnectSuccess) {
		t.Error("连接中状态应该可以成功")
	}
	if !sm.CanTransition(EventConnectFailed) {
		t.Error("连接中状态应该可以失败")
	}
}

func TestStateMachine_StateChangeListener(t *testing.T) {
	sm := NewStateMachine()

	var called int32
	var gotFrom, gotTo ConnectionState
	var gotEvent Event

	done := make(chan struct{})
	sm.OnStateChange(func(from, to ConnectionState, event Event) {
		atomic.AddInt32(&called, 1)
		gotFrom = from
		gotTo = to
		gotEvent = event
		close(done)
	})

	_ = sm.Transition(EventConnect)

	// 等待监听器被调用
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("监听器未被调用")
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("监听器调用次数 = %d, want 1", called)
	}
	if gotFrom != StateDisconnected {
		t.Errorf("from = %s, want Disconnected", gotFrom)
	}
	if gotTo != StateConnecting {
		t.Errorf("to = %s, want Connecting", gotTo)
	}
	if gotEvent != EventConnect {
		t.Errorf("event = %s, want Connect", gotEvent)
	}
}

func TestStateMachine_MultipleListeners(t *testing.T) {
	sm := NewStateMachine()

	var count1, count2 int32
	var wg sync.WaitGroup
	wg.Add(2)

	sm.OnStateChange(func(from, to ConnectionState, event Event) {
		atomic.AddInt32(&count1, 1)
		wg.Done()
	})

	sm.OnStateChange(func(from, to ConnectionState, event Event) {
		atomic.AddInt32(&count2, 1)
		wg.Done()
	})

	_ = sm.Transition(EventConnect)

	// 等待两个监听器都被调用
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("监听器未全部被调用")
	}

	if atomic.LoadInt32(&count1) != 1 || atomic.LoadInt32(&count2) != 1 {
		t.Errorf("监听器调用次数不正确: count1=%d, count2=%d", count1, count2)
	}
}

func TestStateMachine_ConcurrentAccess(t *testing.T) {
	sm := NewStateMachine()

	var wg sync.WaitGroup
	iterations := 100

	// 并发读取状态
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sm.Current()
			_ = sm.IsConnected()
			_ = sm.IsClosed()
			_ = sm.CanTransition(EventConnect)
		}()
	}

	wg.Wait()
}

func TestStateMachine_IsConnected(t *testing.T) {
	sm := NewStateMachine()

	if sm.IsConnected() {
		t.Error("初始状态不应该是已连接")
	}

	_ = sm.Transition(EventConnect)
	if sm.IsConnected() {
		t.Error("连接中状态不应该是已连接")
	}

	_ = sm.Transition(EventConnectSuccess)
	if !sm.IsConnected() {
		t.Error("连接成功后应该是已连接")
	}
}

func TestStateMachine_IsClosed(t *testing.T) {
	sm := NewStateMachine()

	if sm.IsClosed() {
		t.Error("初始状态不应该是已关闭")
	}

	_ = sm.Transition(EventClose)
	if !sm.IsClosed() {
		t.Error("关闭后应该是已关闭")
	}
}

func TestStateMachine_ClosedIsTerminal(t *testing.T) {
	sm := NewStateMachine()
	_ = sm.Transition(EventClose)

	// 关闭后不能进行任何转换
	events := []Event{EventConnect, EventConnectSuccess, EventConnectFailed, EventDisconnect, EventReconnect}
	for _, event := range events {
		if sm.CanTransition(event) {
			t.Errorf("关闭状态不应该可以转换到 %s", event)
		}
	}
}

