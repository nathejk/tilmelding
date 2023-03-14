package command

type AllowedCommands interface {
	NextTeamSlug() string
}

type Commands interface {
	CreatePatrulje()
	UpdatePatrulje() error
}
