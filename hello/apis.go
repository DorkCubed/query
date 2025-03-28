// Service hello implements a simple hello world example with a sql database.
package apis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"net/smtp"

	"encore.dev/beta/errs"
	"encore.dev/cron"
	"encore.dev/storage/sqldb"
)

var SendNew = cron.NewJob("send-email", cron.JobConfig{
	Title:    "Send emails",
	Every:    6 * cron.Hour,
	Endpoint: SendEmail,
})

type AskParams struct {
	ID   string
	Ask  string
	Anon bool
	Name string
}

type AskResponse struct {
	// Message is the greeting response.
	Message string
}

type DbQuery struct {
	ID      string
	Ask     string
	Anon    bool
	Name    string
	Created string
}

// There responds with a personalized greeting.
//
//encore:api public tag:foo
func Ask(ctx context.Context, params *AskParams) (*AskResponse, error) {
	err := storeMessage(ctx, *params)
	if err != nil {
		return nil, err
	}
	return &AskResponse{Message: "Successfully Submitted"}, nil
}

func storeMessage(ctx context.Context, params AskParams) (err error) {
	if params.Ask == "" {
		eb := errs.B()
		return eb.Code(errs.InvalidArgument).Msg("Empty Ask").Err()
	}
	if params.Anon {
		_, err = db.Exec(ctx, `
		INSERT INTO entries (id, ask, anon, name)
		VALUES ($1, $2, $3, $4)
	`, params.ID, params.Ask, params.Anon, "")
	} else {
		if params.Name == "" {
			eb := errs.B()
			return eb.Code(errs.InvalidArgument).Msg("Empty Name").Err()
		}
		_, err = db.Exec(ctx, `
		INSERT INTO entries (id, ask, anon, name)
		VALUES ($1, $2, $3, $4)
	`, params.ID, params.Ask, params.Anon, params.Name)
	}

	if err != nil {
		return fmt.Errorf("could not update table: %v", err)
	}

	return nil
}

//encore:api private
func SendEmail(ctx context.Context) error {
	var mail = ""
	rows, err := db.Query(ctx, `
    SELECT id, ask, name, created
    FROM entries`)
	if err != nil {
		return fmt.Errorf("could not query table: %v", err)
	}

	for rows.Next() {
		var row []any = make([]any, 4)
		err = rows.Scan(&row[0], &row[1], &row[2], &row[3])
		if err != nil {
			return fmt.Errorf("could not scan row: %v", err)
		}
		if row[2] == "" {
			mail = mail + fmt.Sprintf("Anonymous asked %v\nat %v\n\n", row[1], row[3])
		} else {
			mail = mail + fmt.Sprintf("%v asked %v\nat %v\n\n", row[2], row[1], row[3])
		}
	}
	rows.Close()

	if mail == "" {
		log.Printf("No messages")
		return nil
	}
	err = smtpSender(mail)
	if err != nil {
		return fmt.Errorf("could not send email: %v", err)
	}

	_, err = db.Exec(ctx, `
	DELETE FROM entries`)
	if err != nil {
		return fmt.Errorf("could not delete table: %v", err)
	}

	return nil
}

func smtpSender(msg string) error {
	env, err := os.ReadFile("./hello/.env")
	if err != nil {
		log.Printf("Error reading .env file")
	}
	smtpParams := strings.Split(string(env), ", ")
	log.Printf("SMTP Params: %v", smtpParams)

	from := string(smtpParams[0])
	password := string(smtpParams[1])
	toList := []string{smtpParams[2]}
	host := string(smtpParams[3])
	port := "587"

	msg = "Subject: Today's Messages\n\n" + msg
	body := []byte(msg)
	auth := smtp.PlainAuth("", from, password, host)
	err = smtp.SendMail(host+":"+port, auth, from, toList, body)
	return err
}

/*
//encore:api public raw
func Submit(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	forwarded := req.Header.Get("X-FORWARDED-FOR")
	log.Printf("Req: %v", req.Body)
	var ip string
	if forwarded != "" {
		ip = forwarded
	} else {
		ip = req.RemoteAddr
	}
	log.Printf("IP: %v", ip)
		err := InsertIP(context.Background(), ip)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	resp, _ := json.Marshal(map[string]string{
		"Message": "Submitted Successfully",
	})
	w.Write(resp)
}

func InsertIP(ctx context.Context, IP string) (err error) {
	err = db.QueryRow(ctx, `
	INSERT INTO entries (id, ip)
	VALUES ($1, $2)
	ON CONFLICT (id) DO UPDATE
	SET ip = $2
	RETURNING ip
`, id, IP).Scan(&ip)
}
*/

// Define a database named 'hello', using the database migrations
// in the "./migrations" folder. Encore automatically provisions,
// migrates, and connects to the database.
// Learn more: https://encore.dev/docs/primitives/databases
var db = sqldb.NewDatabase("hello", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})
