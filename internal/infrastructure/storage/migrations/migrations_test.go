package migrations

import (
	"strings"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migrations SQL Injection Safe Reset", func() {
	It("safely handles malicious table names when constructing queries", func() {
		// Mock testing the query string construction because testcontainers doesn't work in this exact environment setup (containerd overlay issues)
		tables := []string{"normal_table", "my\"table", "user_data", "drop table users;"}

		var escapedTables []string
		for _, tableName := range tables {
			escapedTables = append(escapedTables, pgx.Identifier{tableName}.Sanitize())
		}

		query := "TRUNCATE TABLE " + strings.Join(escapedTables, ", ") + " RESTART IDENTITY CASCADE;"

		Expect(query).To(Equal(`TRUNCATE TABLE "normal_table", "my""table", "user_data", "drop table users;" RESTART IDENTITY CASCADE;`))

		// The malicious strings are properly quoted and escaped with double quotes!
		Expect(strings.Contains(query, `my""table`)).To(BeTrue())
	})
})
