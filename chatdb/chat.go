package chatdb

import (
	"sort"

	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
)

type (
	// EntityChats represents all of the chats with a given entity (associated
	// with the same vCard, phone number, or email address). In the case of group
	// chats, this struct will only contain a single Chat.
	EntityChats struct {
		Name  string
		Chats []Chat
	}

	// Chat represents a row from the chat table.
	Chat struct {
		ID   int
		GUID string
	}
)

func (d chatDB) GetChats(contactMap map[string]*vcard.Card) ([]EntityChats, error) {
	chatRows, err := d.DB.Query("SELECT ROWID, guid, chat_identifier, COALESCE(display_name, '') FROM chat")
	if err != nil {
		return nil, errors.Wrap(err, "query chats table")
	}
	defer chatRows.Close()
	contactChats := map[*vcard.Card]EntityChats{}
	addressChats := map[string]EntityChats{}
	for chatRows.Next() {
		var id int
		var guid, name, displayName string
		if err := chatRows.Scan(&id, &guid, &name, &displayName); err != nil {
			return nil, errors.Wrap(err, "read chat")
		}
		if displayName == "" {
			displayName = name
		}
		chat := Chat{
			ID:   id,
			GUID: guid,
		}
		if card, ok := contactMap[name]; ok {
			addContactChat(card, displayName, chat, contactChats)
		} else {
			addAddressChat(name, displayName, chat, addressChats)
		}
	}
	chats := []EntityChats{}
	for _, entityChats := range contactChats {
		chats = append(chats, entityChats)
	}
	for _, entityChats := range addressChats {
		chats = append(chats, entityChats)
	}
	sort.SliceStable(chats, func(i, j int) bool { return chats[i].Name < chats[j].Name })
	return chats, nil
}

func addContactChat(card *vcard.Card, displayName string, chat Chat, contactChats map[*vcard.Card]EntityChats) {
	if entityChats, ok := contactChats[card]; ok {
		// We have contact info, and we have seen this contact before.
		entityChats.Chats = append(entityChats.Chats, chat)
		contactChats[card] = entityChats
		return
	}
	// We have contact info, but we haven't seen this contact before.
	contactName := card.PreferredValue(vcard.FieldFormattedName)
	if contactName != "" {
		displayName = contactName
	}
	contactChats[card] = EntityChats{
		Name:  displayName,
		Chats: []Chat{chat},
	}
}

func addAddressChat(address, displayName string, chat Chat, addressChats map[string]EntityChats) {
	entityChats, ok := addressChats[address]
	if ok {
		// We don't have contact info, and we have seen this address before.
		entityChats.Chats = append(entityChats.Chats, chat)
		addressChats[address] = entityChats
		return
	}
	// We don't have contact info, and this is a new address.
	addressChats[address] = EntityChats{
		Name:  displayName,
		Chats: []Chat{chat},
	}
}
