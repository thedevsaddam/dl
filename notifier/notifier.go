package notifier

import "github.com/martinlindhe/notify"

// Notifier represents a notifier type
type Notifier struct {
	appName  string
	iconPath string
}

// New return a new notifier
func New(name string) *Notifier {
	writeIcon() // call every time
	return &Notifier{
		appName: name,
	}
}

// Notify send a notification to device
func (n *Notifier) Notify(title, text string) {
	iconPath, _ := getIconPath()
	notify.Notify(n.appName, title, text, iconPath)
}
