package tibia

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCharacterDecodesV5Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/character/Trollefar" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		fmt.Fprint(w, `{"character":{"character":{"name":"Trollefar","sex":"male","vocation":"Knight","level":202,"residence":"Thais","guild":{"name":"Elysium","rank":"Follower"},"last_login":"2026-07-05T00:33:18Z","account_status":"Premium Account"}}}`)
	}))
	defer server.Close()

	character, err := NewClient(server.URL+"/v5/", "").GetCharacter("Trollefar")
	if err != nil {
		t.Fatalf("GetCharacter() error = %v", err)
	}

	if character.Name != "Trollefar" || character.Vocation != "Knight" || character.Level != 202 {
		t.Fatalf("unexpected character identity: %+v", character)
	}
	if character.Residence != "Thais" || character.Guild != "Elysium" || character.GuildRank != "Follower" {
		t.Fatalf("unexpected character details: %+v", character)
	}
	if !character.IsPremium || character.LastLogin != "2026-07-05T00:33:18Z" {
		t.Fatalf("unexpected account details: %+v", character)
	}
}

func TestGetGuildMembersUsesProxyURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/guilds/42" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		fmt.Fprint(w, `{"guild_id":42,"members":[{"name":"Trollefar"}],"total":1}`)
	}))
	defer server.Close()

	guild, err := NewClient("https://api.tibiadata.com/v5", server.URL).GetGuildMembers(42)
	if err != nil {
		t.Fatalf("GetGuildMembers() error = %v", err)
	}
	if guild.GuildID != 42 || guild.Total != 1 || len(guild.Members) != 1 || guild.Members[0].Name != "Trollefar" {
		t.Fatalf("unexpected guild response: %+v", guild)
	}
}
