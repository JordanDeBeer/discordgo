package discordgo

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type InteractionsHTTPServer struct {
	*Session
	publicKey ed25519.PublicKey
	handlers  map[string]HandlerFunc
}

type HandlerFunc func(*Session, *InteractionCreate)

func NewInteractionsHTTPServer(sess *Session, pubKeyString string) *InteractionsHTTPServer {
	key, err := hex.DecodeString(pubKeyString)
	if err != nil {
		log.Fatal("couldn't parse public key string")
	}
	return &InteractionsHTTPServer{sess, key, make(map[string]HandlerFunc)}
}

func (s *InteractionsHTTPServer) HandleFunc(name string, fn func(*Session, *InteractionCreate)) {
	s.handlers[name] = fn
}

func (s *InteractionsHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ok := VerifyInteraction(r, s.publicKey); !ok {
		http.Error(w, "invalid request signature", http.StatusUnauthorized)
		return
	}
	var interaction Interaction
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("could not read request body: %v", err)
		http.Error(w, "could not read request body", http.StatusInternalServerError)
		return
	}
	err = interaction.UnmarshalJSON(data)
	if err != nil {
		http.Error(w, "could not parse interaction", http.StatusBadRequest)
		log.Printf("could not parse request interaction: %v", err)
		return
	}
	switch interaction.Type {
	case InteractionPing:
		response := InteractionResponse{Type: InteractionResponsePong}

		b, err := json.Marshal(response)
		if err != nil {
			log.Printf("could not marshal response: %v", err)
			return
		}
		w.Write(b)
	case InteractionApplicationCommand:
		if h, ok := s.handlers[interaction.ApplicationCommandData().Name]; ok {
			h(s.Session, &InteractionCreate{&interaction})
		}
	default:
		log.Printf("unrecognized type: %v", interaction.Type.String())
	}
}