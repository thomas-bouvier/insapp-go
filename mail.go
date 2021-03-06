package insapp

import (
	"bytes"
	"html/template"
	"net/smtp"
)

// SendEmail sends an email to the given recipient.
func SendEmail(to string, subject string, body string) {
	from := config.GoogleEmail
	pass := config.GooglePassword
	cc := config.GoogleEmail

	if config.Environment != "prod" {
		to = from
		subject = "[DEV] " + subject
	}

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Cc: " + cc + "\n" +
		"Subject: " + subject + "\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		body

	smtp.SendMail("smtp.gmail.com:587", smtp.PlainAuth("", from, pass, "smtp.gmail.com"), from, []string{to}, []byte(msg))
}

// SendAssociationEmailSubscription sends a subscription email containing the credentials.
func SendAssociationEmailSubscription(email string, password string) error {
	data := struct {
		Email    string
		Password string
	}{
		Email:    email,
		Password: password,
	}

	body, err := parseTemplate("templates/association_subscription_template.html", data)
	if err == nil {
		SendEmail(email, "Tes identifiants Insapp", body)
	}

	return err
}

// SendAssociationEmailForCommentOnEvent sends an email indicating
// a new comment has been added on an event
func SendAssociationEmailForCommentOnEvent(email string, event Event, comment Comment, user User) error {
	data := struct {
		EventName        string
		EventImage       string
		EventDescription string
		CommentContent   string
		Username         string
	}{
		EventName:        event.Name,
		EventImage:       config.GetCDN() + event.Image,
		EventDescription: event.Description,
		CommentContent:   comment.Content,
		Username:         user.Username,
	}

	body, err := parseTemplate("templates/association_comment_event_template.html", data)
	if err == nil {
		SendEmail(email, "Nouveau commentaire sur \""+event.Name+"\"", body)
	}

	return err
}

// SendAssociationEmailForCommentOnPost sends an email indicating
// a new comment has been added on a post
func SendAssociationEmailForCommentOnPost(email string, post Post, comment Comment, user User) error {
	data := struct {
		PostName        string
		PostImage       string
		PostDescription string
		CommentContent  string
		Username        string
	}{
		PostName:        post.Title,
		PostImage:       config.GetCDN() + post.Image,
		PostDescription: post.Description,
		CommentContent:  comment.Content,
		Username:        user.Username,
	}

	body, err := parseTemplate("templates/association_comment_post_template.html", data)
	if err == nil {
		SendEmail(email, "Nouveau commentaire sur \""+post.Title+"\"", body)
	}

	return err
}

func parseTemplate(templateFileName string, data interface{}) (string, error) {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
