//////////////////////////////////////////////////////////////////////
//
// Given is a SessionManager that stores session information in
// memory. The SessionManager itself is working, however, since we
// keep on adding new sessions to the manager our program will
// eventually run out of memory.
//
// Your task is to implement a session cleaner routine that runs
// concurrently in the background and cleans every session that
// hasn't been updated for more than 5 seconds (of course usually
// session times are much longer).
//
// Note that we expect the session to be removed anytime between 5 and
// 7 seconds after the last update. Also, note that you have to be
// very careful in order to prevent race conditions.
//

package main

import (
	"container/heap"
	"errors"
	"log"
	"sync"
	"time"
)

type queueItem struct {
	key       string
	expiresAt time.Time
	index     int
}

type queue []*queueItem

func (pq queue) Len() int { return len(pq) }

func (pq queue) Less(i, j int) bool {
	// minimum is always first
	return pq[i].expiresAt.Before(pq[j].expiresAt)
}

func (pq queue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *queue) Push(x any) {
	n := len(*pq)
	item := x.(*queueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *queue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *queue) update(item *queueItem, key string, expiresAt time.Time) {
	item.key = key
	item.expiresAt = expiresAt
	heap.Fix(pq, item.index)
}

// SessionManager keeps track of all sessions from creation, updating
// to destroying.
type SessionManager struct {
	mu            sync.RWMutex
	sessions      map[string]Session
	evictionQueue queue
}

// Session stores the session's data
type Session struct {
	Data map[string]interface{}
	item *queueItem
}

// NewSessionManager creates a new sessionManager
func NewSessionManager() *SessionManager {
	m := &SessionManager{
		sessions: make(map[string]Session),
	}
	heap.Init(&m.evictionQueue)

	go func() {
		for {
			time.Sleep(5 * time.Second)

			m.mu.Lock()
			for m.evictionQueue.Len() > 0 && m.evictionQueue[0].expiresAt.Before(time.Now()) {
				it := m.evictionQueue.Pop()
				delete(m.sessions, it.(*queueItem).key)
			}
			m.mu.Unlock()
		}
	}()

	return m
}

// CreateSession creates a new session and returns the sessionID
func (m *SessionManager) CreateSession() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID, err := MakeSessionID()
	if err != nil {
		return "", err
	}

	item := &queueItem{
		key:       sessionID,
		expiresAt: time.Now().Add(5 * time.Second),
	}
	m.sessions[sessionID] = Session{
		Data: make(map[string]interface{}),
		item: item,
	}
	heap.Push(&m.evictionQueue, item)

	return sessionID, nil
}

// ErrSessionNotFound returned when sessionID not listed in
// SessionManager
var ErrSessionNotFound = errors.New("SessionID does not exists")

// GetSessionData returns data related to session if sessionID is
// found, errors otherwise
func (m *SessionManager) GetSessionData(sessionID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session.Data, nil
}

// UpdateSessionData overwrites the old session data with the new one
func (m *SessionManager) UpdateSessionData(sessionID string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	// Hint: you should renew expiry of the session here
	m.evictionQueue.update(val.item, sessionID, time.Now().Add(5*time.Second))
	m.sessions[sessionID] = Session{
		Data: data,
		item: val.item,
	}

	return nil
}

func main() {
	// Create new sessionManager and new session
	m := NewSessionManager()
	sID, err := m.CreateSession()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Created new session with ID", sID)

	// Update session data
	data := make(map[string]interface{})
	data["website"] = "longhoang.de"

	err = m.UpdateSessionData(sID, data)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Update session data, set website to longhoang.de")

	// Retrieve data from manager again
	updatedData, err := m.GetSessionData(sID)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Get session data:", updatedData)
}
