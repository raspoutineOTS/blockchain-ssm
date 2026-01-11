// Copyright Blockchain SSM Lightning Integration 2025
// License: Apache-2.0

package main

import (
	"fmt"
	"sync"
)

// SessionRegistry manages session counters for transaction number encoding
type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[string]uint16 // sessionID -> counter
	reverse  map[uint16]string // counter -> sessionID
	nextID   uint16
}

// NewSessionRegistry creates a new session registry
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[string]uint16),
		reverse:  make(map[uint16]string),
		nextID:   1, // Start from 1, reserve 0 for special cases
	}
}

// RegisterSession assigns a counter to a session ID
func (sr *SessionRegistry) RegisterSession(sessionID string) (uint16, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Check if already registered
	if counter, exists := sr.sessions[sessionID]; exists {
		return counter, nil
	}

	// Check if we've exceeded max sessions (65535)
	if sr.nextID == 0 {
		return 0, fmt.Errorf("session registry full: maximum 65535 sessions reached")
	}

	// Assign new counter
	counter := sr.nextID
	sr.sessions[sessionID] = counter
	sr.reverse[counter] = sessionID
	sr.nextID++

	return counter, nil
}

// GetSessionCounter retrieves the counter for a session ID
func (sr *SessionRegistry) GetSessionCounter(sessionID string) (uint16, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	counter, exists := sr.sessions[sessionID]
	return counter, exists
}

// GetSessionID retrieves the session ID for a counter
func (sr *SessionRegistry) GetSessionID(counter uint16) (string, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	sessionID, exists := sr.reverse[counter]
	return sessionID, exists
}

// Count returns the number of registered sessions
func (sr *SessionRegistry) Count() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	return len(sr.sessions)
}

// Global session registry instance
var globalSessionRegistry = NewSessionRegistry()

// GetSessionCounter is a helper to get counter from global registry
func GetSessionCounter(sessionID string) uint16 {
	counter, exists := globalSessionRegistry.GetSessionCounter(sessionID)
	if !exists {
		// Auto-register if not found
		counter, _ = globalSessionRegistry.RegisterSession(sessionID)
	}
	return counter
}

// StateEncoder provides encoding/decoding utilities for SSM states
type StateEncoder struct{}

// EncodeStateTransition creates a compact representation of a state transition
func (se *StateEncoder) EncodeStateTransition(from, to int, role, action string) []byte {
	// Encode transition as compact binary format
	// Format: [1 byte: from][1 byte: to][1 byte: role_hash][1 byte: action_code]

	roleHash := hashStringToByte(role)
	actionCode := GetActionCode(action)

	return []byte{
		byte(from),
		byte(to),
		roleHash,
		actionCode,
	}
}

// DecodeStateTransition decodes a compact state transition
func (se *StateEncoder) DecodeStateTransition(data []byte) (from, to int, roleHash byte, actionCode uint8) {
	if len(data) < 4 {
		return 0, 0, 0, 0
	}

	from = int(data[0])
	to = int(data[1])
	roleHash = data[2]
	actionCode = data[3]

	return
}

// hashStringToByte creates a deterministic 1-byte hash of a string
func hashStringToByte(s string) byte {
	hash := byte(0)
	for i := 0; i < len(s); i++ {
		hash = hash*31 + s[i]
	}
	return hash
}

// CompactStateMetadata contains minimal metadata for state verification
type CompactStateMetadata struct {
	Version   uint8  `json:"v"`   // Protocol version
	Type      uint8  `json:"t"`   // State type (SSM, Grant, etc.)
	Flags     uint8  `json:"f"`   // Feature flags
	Reserved  uint8  `json:"r"`   // Reserved for future use
}

// EncodeMetadata encodes metadata into 4 bytes
func (se *StateEncoder) EncodeMetadata(meta *CompactStateMetadata) uint32 {
	return uint32(meta.Version)<<24 |
		uint32(meta.Type)<<16 |
		uint32(meta.Flags)<<8 |
		uint32(meta.Reserved)
}

// DecodeMetadata decodes metadata from 4 bytes
func (se *StateEncoder) DecodeMetadata(encoded uint32) *CompactStateMetadata {
	return &CompactStateMetadata{
		Version:  uint8(encoded >> 24),
		Type:     uint8((encoded >> 16) & 0xFF),
		Flags:    uint8((encoded >> 8) & 0xFF),
		Reserved: uint8(encoded & 0xFF),
	}
}

// BitFlags for compact state representation
const (
	FLAG_HAS_LIMIT      = 1 << 0 // State has iteration limit
	FLAG_HAS_PRIVATE    = 1 << 1 // State has private data
	FLAG_IS_FINAL       = 1 << 2 // State is in final/acceptance state
	FLAG_IS_ANCHORED    = 1 << 3 // State is anchored on Bitcoin
	FLAG_IS_MINTED      = 1 << 4 // Taproot Asset has been minted
	FLAG_IS_TRANSFERRED = 1 << 5 // Asset has been transferred
	FLAG_RESERVED_6     = 1 << 6 // Reserved
	FLAG_RESERVED_7     = 1 << 7 // Reserved
)

// SetFlag sets a flag in the flags byte
func SetFlag(flags uint8, flag uint8) uint8 {
	return flags | flag
}

// ClearFlag clears a flag in the flags byte
func ClearFlag(flags uint8, flag uint8) uint8 {
	return flags &^ flag
}

// HasFlag checks if a flag is set
func HasFlag(flags uint8, flag uint8) bool {
	return (flags & flag) != 0
}

// StateFlags represents the flags for a state
type StateFlags struct {
	HasLimit      bool
	HasPrivate    bool
	IsFinal       bool
	IsAnchored    bool
	IsMinted      bool
	IsTransferred bool
}

// ToUint8 converts StateFlags to uint8
func (sf *StateFlags) ToUint8() uint8 {
	var flags uint8 = 0

	if sf.HasLimit {
		flags = SetFlag(flags, FLAG_HAS_LIMIT)
	}
	if sf.HasPrivate {
		flags = SetFlag(flags, FLAG_HAS_PRIVATE)
	}
	if sf.IsFinal {
		flags = SetFlag(flags, FLAG_IS_FINAL)
	}
	if sf.IsAnchored {
		flags = SetFlag(flags, FLAG_IS_ANCHORED)
	}
	if sf.IsMinted {
		flags = SetFlag(flags, FLAG_IS_MINTED)
	}
	if sf.IsTransferred {
		flags = SetFlag(flags, FLAG_IS_TRANSFERRED)
	}

	return flags
}

// FromUint8 creates StateFlags from uint8
func (sf *StateFlags) FromUint8(flags uint8) {
	sf.HasLimit = HasFlag(flags, FLAG_HAS_LIMIT)
	sf.HasPrivate = HasFlag(flags, FLAG_HAS_PRIVATE)
	sf.IsFinal = HasFlag(flags, FLAG_IS_FINAL)
	sf.IsAnchored = HasFlag(flags, FLAG_IS_ANCHORED)
	sf.IsMinted = HasFlag(flags, FLAG_IS_MINTED)
	sf.IsTransferred = HasFlag(flags, FLAG_IS_TRANSFERRED)
}

// CompressionStats tracks compression effectiveness
type CompressionStats struct {
	OriginalSize   int
	CompressedSize int
	Ratio          float64
	Method         string
}

// CalculateCompressionRatio computes the compression ratio
func CalculateCompressionRatio(original, compressed int) float64 {
	if original == 0 {
		return 0.0
	}
	return float64(compressed) / float64(original)
}

// CompactRoleMapping creates compact role assignments
type CompactRoleMapping struct {
	RoleCode  uint8  // Encoded role (Buyer=1, Seller=2, etc.)
	AgentHash uint16 // 2-byte hash of agent public key
}

// Role codes
const (
	ROLE_INITIATOR  = 0x01
	ROLE_VALIDATOR  = 0x02
	ROLE_BUYER      = 0x03
	ROLE_SELLER     = 0x04
	ROLE_VERIFIER   = 0x05
	ROLE_AUDITOR    = 0x06
	ROLE_ORACLE     = 0x07
	ROLE_ADMIN      = 0x08
)

// GetRoleCode converts role name to code
func GetRoleCode(roleName string) uint8 {
	roleMap := map[string]uint8{
		"Initiator": ROLE_INITIATOR,
		"Validator": ROLE_VALIDATOR,
		"Buyer":     ROLE_BUYER,
		"Seller":    ROLE_SELLER,
		"Verifier":  ROLE_VERIFIER,
		"Auditor":   ROLE_AUDITOR,
		"Oracle":    ROLE_ORACLE,
		"Admin":     ROLE_ADMIN,
	}

	if code, exists := roleMap[roleName]; exists {
		return code
	}
	return 0x00 // Unknown role
}

// GetRoleName converts role code to name
func GetRoleName(code uint8) string {
	roleNames := map[uint8]string{
		ROLE_INITIATOR: "Initiator",
		ROLE_VALIDATOR: "Validator",
		ROLE_BUYER:     "Buyer",
		ROLE_SELLER:    "Seller",
		ROLE_VERIFIER:  "Verifier",
		ROLE_AUDITOR:   "Auditor",
		ROLE_ORACLE:    "Oracle",
		ROLE_ADMIN:     "Admin",
	}

	if name, exists := roleNames[code]; exists {
		return name
	}
	return "Unknown"
}

// EncodeRoles creates compact role mapping
func EncodeRoles(roles map[string]string) []CompactRoleMapping {
	var compactRoles []CompactRoleMapping

	for agentName, roleName := range roles {
		roleCode := GetRoleCode(roleName)
		agentHash := hashStringToUint16(agentName)

		compactRoles = append(compactRoles, CompactRoleMapping{
			RoleCode:  roleCode,
			AgentHash: agentHash,
		})
	}

	return compactRoles
}

// hashStringToUint16 creates a 2-byte hash of a string
func hashStringToUint16(s string) uint16 {
	hash := uint16(0)
	for i := 0; i < len(s); i++ {
		hash = hash*31 + uint16(s[i])
	}
	return hash
}

// VerifyAgentRole verifies that an agent has a specific role
func VerifyAgentRole(agentName, expectedRole string, compactRoles []CompactRoleMapping) bool {
	expectedRoleCode := GetRoleCode(expectedRole)
	agentHash := hashStringToUint16(agentName)

	for _, mapping := range compactRoles {
		if mapping.AgentHash == agentHash && mapping.RoleCode == expectedRoleCode {
			return true
		}
	}

	return false
}
