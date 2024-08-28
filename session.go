package session

import (
	stdmaps "maps"
	"slices"

	"github.com/go-rat/securecookie"
	"github.com/go-rat/utils/maps"
	"github.com/jaevor/go-nanoid"
	"github.com/spf13/cast"

	"github.com/go-rat/session/driver"
)

type Session struct {
	id         string
	name       string
	attributes map[string]any
	codec      securecookie.Codec
	driver     driver.Driver
	started    bool
}

func (s *Session) All() map[string]any {
	return s.attributes
}

func (s *Session) Exists(key string) bool {
	return maps.Exists(s.attributes, key)
}

func (s *Session) Flash(key string, value any) *Session {
	s.Put(key, value)

	old := s.Get("_flash.new", []any{}).([]any)
	s.Put("_flash.new", append(old, key))

	s.removeFromOldFlashData(key)
	return s
}

func (s *Session) Flush() *Session {
	s.attributes = make(map[string]any)
	return s
}

func (s *Session) Forget(keys ...string) *Session {
	maps.Forget(s.attributes, keys...)
	return s
}

func (s *Session) Get(key string, defaultValue ...any) any {
	return maps.Get(s.attributes, key, defaultValue...)
}

func (s *Session) GetID() string {
	return s.id
}

func (s *Session) GetName() string {
	return s.name
}

func (s *Session) Has(key string) bool {
	val, ok := s.attributes[key]
	if !ok {
		return false
	}

	return val != nil
}

func (s *Session) Invalidate() error {
	s.Flush()
	return s.migrate(true)
}

func (s *Session) Keep(keys ...string) *Session {
	s.mergeNewFlashes(keys...)
	s.removeFromOldFlashData(keys...)
	return s
}

func (s *Session) Missing(key string) bool {
	return !s.Exists(key)
}

func (s *Session) Now(key string, value any) *Session {
	s.Put(key, value)

	old := s.Get("_flash.old", []any{}).([]any)
	s.Put("_flash.old", append(old, key))

	return s
}

func (s *Session) Only(keys []string) map[string]any {
	return maps.Only(s.attributes, keys...)
}

func (s *Session) Pull(key string, def ...any) any {
	return maps.Pull(s.attributes, key, def...)
}

func (s *Session) Put(key string, value any) *Session {
	s.attributes[key] = value
	return s
}

func (s *Session) Reflash() *Session {
	old := toStringSlice(s.Get("_flash.old", []any{}).([]any))
	s.mergeNewFlashes(old...)
	s.Put("_flash.old", []any{})
	return s
}

func (s *Session) Regenerate(destroy ...bool) error {
	err := s.migrate(destroy...)
	if err != nil {
		return err
	}

	return nil
}

func (s *Session) Remove(key string) any {
	return s.Pull(key)
}

func (s *Session) Save() error {
	s.ageFlashData()

	data, err := s.codec.Encode(s.GetName(), s.attributes)
	if err != nil {
		return err
	}

	if err = s.driver.Write(s.GetID(), data); err != nil {
		return err
	}

	s.started = false
	return nil
}

func (s *Session) SetID(id string) *Session {
	if s.isValidID(id) {
		s.id = id
	} else {
		s.id = s.generateSessionID()
	}

	return s
}

func (s *Session) SetName(name string) *Session {
	s.name = name
	return s
}

func (s *Session) Start() bool {
	s.loadSession()
	s.started = true
	return s.started
}

func (s *Session) generateSessionID() string {
	alphabet := `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`
	generator := nanoid.MustCustomASCII(alphabet, 32)
	return generator()
}

func (s *Session) isValidID(id string) bool {
	return len(id) == 32
}

func (s *Session) loadSession() {
	data := s.readFromHandler()
	if data != nil {
		stdmaps.Copy(s.attributes, data)
	}
}

func (s *Session) migrate(destroy ...bool) error {
	shouldDestroy := false
	if len(destroy) > 0 {
		shouldDestroy = destroy[0]
	}

	if shouldDestroy {
		err := s.driver.Destroy(s.GetID())
		if err != nil {
			return err
		}
	}

	s.id = s.generateSessionID()
	return nil
}

func (s *Session) readFromHandler() map[string]any {
	value, err := s.driver.Read(s.GetID())
	if err != nil {
		return nil
	}

	var data map[string]any
	if err = s.codec.Decode(s.GetName(), value, &data); err != nil {
		return nil
	}
	return data
}

func (s *Session) ageFlashData() {
	old := toStringSlice(s.Get("_flash.old", []any{}).([]any))
	s.Forget(old...)
	s.Put("_flash.old", s.Get("_flash.new", []any{}))
	s.Put("_flash.new", []any{})
}

func (s *Session) mergeNewFlashes(keys ...string) {
	values := s.Get("_flash.new", []any{}).([]any)
	for _, key := range keys {
		if !slices.Contains(values, any(key)) {
			values = append(values, key)
		}
	}

	s.Put("_flash.new", values)
}

func (s *Session) removeFromOldFlashData(keys ...string) {
	old := s.Get("_flash.old", []any{}).([]any)
	for _, key := range keys {
		old = slices.DeleteFunc(old, func(i any) bool {
			return cast.ToString(i) == key
		})
	}
	s.Put("_flash.old", old)
}

func (s *Session) reset() {
	s.id = ""
	s.name = ""
	s.attributes = make(map[string]any)
	s.codec = nil
	s.driver = nil
	s.started = false
}

// toStringSlice converts an interface slice to a string slice.
func toStringSlice(anySlice []any) []string {
	strSlice := make([]string, len(anySlice))
	for i, v := range anySlice {
		strSlice[i] = cast.ToString(v)
	}
	return strSlice
}
