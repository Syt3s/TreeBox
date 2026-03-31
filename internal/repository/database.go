// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"github.com/pkg/errors"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/model"
)

var AllTables = []interface{}{
	&model.User{},
	&model.Question{},
	&model.Tenant{},
	&model.TenantMember{},
	&model.Workspace{},
	&model.AuditLog{},
}

func Init(typ, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch typ {
	case "mysql", "":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, errors.Errorf("unknown database type: %q", typ)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "connect to database")
	}

	if err := db.AutoMigrate(AllTables...); err != nil {
		return nil, errors.Wrap(err, "auto migrate")
	}

	Users = NewUserRepository(db)
	Questions = NewQuestionRepository(db)
	Tenants = NewTenantRepository(db)
	Workspaces = NewWorkspaceRepository(db)
	AuditLogs = NewAuditLogRepository(db)

	if err := db.Use(otelgorm.NewPlugin(
		otelgorm.WithDBName(config.Database.Name),
	)); err != nil {
		return nil, errors.Wrap(err, "register otelgorm plugin")
	}

	return db, nil
}
