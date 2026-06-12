package migrate

import "context"

// SQLStatement is a SQL query and its bound arguments.
type SQLStatement struct {
	Query string
	Args  []any
}

// SchemaPlan stores ordered DDL and DML statements for later execution.
type SchemaPlan struct {
	ctx        *MigrationContext
	statements []SQLStatement
	err        error
}

// Raw appends a raw SQL statement.
func (p *SchemaPlan) Raw(sql string, args ...any) *SchemaPlan {
	return p.add(SQLStatement{Query: sql, Args: args})
}

// CreateTable appends a CREATE TABLE statement.
func (p *SchemaPlan) CreateTable(table string, columns *ColumnList, options ...string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.CreateTable(table, columns, optionString(options))})
}

// CreateTableIfNotExists appends CREATE TABLE when the table does not exist.
func (p *SchemaPlan) CreateTableIfNotExists(ctx context.Context, table string, columns *ColumnList, options ...string) *SchemaPlan {
	exists, err := p.ctx.TableExists(ctx, table)
	return p.addIf(!exists, err, SQLStatement{Query: p.ctx.dialect.CreateTable(table, columns, optionString(options))})
}

// DropTable appends a DROP TABLE statement.
func (p *SchemaPlan) DropTable(table string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropTable(table)})
}

// DropTableIfExists appends DROP TABLE when the table exists.
func (p *SchemaPlan) DropTableIfExists(ctx context.Context, table string) *SchemaPlan {
	exists, err := p.ctx.TableExists(ctx, table)
	return p.addIf(exists, err, SQLStatement{Query: p.ctx.dialect.DropTable(table)})
}

// RenameTable appends a RENAME TABLE statement.
func (p *SchemaPlan) RenameTable(oldName, newName string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.RenameTable(oldName, newName)})
}

// TruncateTable appends a TRUNCATE TABLE statement.
func (p *SchemaPlan) TruncateTable(table string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.TruncateTable(table)})
}

// AddColumn appends an ADD COLUMN statement.
func (p *SchemaPlan) AddColumn(table, column string, builder *ColumnBuilder) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.AddColumn(table, column, builder)})
}

// AddColumnIfNotExists appends ADD COLUMN when the column does not exist.
func (p *SchemaPlan) AddColumnIfNotExists(ctx context.Context, table, column string, builder *ColumnBuilder) *SchemaPlan {
	exists, err := p.ctx.ColumnExists(ctx, table, column)
	return p.addIf(!exists, err, SQLStatement{Query: p.ctx.dialect.AddColumn(table, column, builder)})
}

// AlterColumn appends a MODIFY COLUMN statement.
func (p *SchemaPlan) AlterColumn(table, column string, builder *ColumnBuilder) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.AlterColumn(table, column, builder)})
}

// DropColumn appends a DROP COLUMN statement.
func (p *SchemaPlan) DropColumn(table, column string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropColumn(table, column)})
}

// DropColumnIfExists appends DROP COLUMN when the column exists.
func (p *SchemaPlan) DropColumnIfExists(ctx context.Context, table, column string) *SchemaPlan {
	exists, err := p.ctx.ColumnExists(ctx, table, column)
	return p.addIf(exists, err, SQLStatement{Query: p.ctx.dialect.DropColumn(table, column)})
}

// RenameColumn appends a RENAME COLUMN statement.
func (p *SchemaPlan) RenameColumn(table, oldName, newName string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.RenameColumn(table, oldName, newName)})
}

// CreateIndex appends a CREATE INDEX statement.
func (p *SchemaPlan) CreateIndex(name, table string, columns []string, unique bool) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.CreateIndex(name, table, columns, unique)})
}

// CreateIndexIfNotExists appends CREATE INDEX when the index does not exist.
func (p *SchemaPlan) CreateIndexIfNotExists(ctx context.Context, name, table string, columns []string, unique bool) *SchemaPlan {
	exists, err := p.ctx.IndexExists(ctx, table, name)
	return p.addIf(!exists, err, SQLStatement{Query: p.ctx.dialect.CreateIndex(name, table, columns, unique)})
}

// DropIndex appends a DROP INDEX statement.
func (p *SchemaPlan) DropIndex(name, table string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropIndex(name, table)})
}

// DropIndexIfExists appends DROP INDEX when the index exists.
func (p *SchemaPlan) DropIndexIfExists(ctx context.Context, name, table string) *SchemaPlan {
	exists, err := p.ctx.IndexExists(ctx, table, name)
	return p.addIf(exists, err, SQLStatement{Query: p.ctx.dialect.DropIndex(name, table)})
}

// AddPrimaryKey appends an ADD PRIMARY KEY statement.
func (p *SchemaPlan) AddPrimaryKey(name, table string, columns []string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.AddPrimaryKey(name, table, columns)})
}

// AddPrimaryKeyIfNotExists appends ADD PRIMARY KEY when the named constraint does not exist.
func (p *SchemaPlan) AddPrimaryKeyIfNotExists(ctx context.Context, name, table string, columns []string) *SchemaPlan {
	exists, err := p.ctx.ConstraintExists(ctx, table, name)
	return p.addIf(!exists, err, SQLStatement{Query: p.ctx.dialect.AddPrimaryKey(name, table, columns)})
}

// DropPrimaryKey appends a DROP PRIMARY KEY statement.
func (p *SchemaPlan) DropPrimaryKey(name, table string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropPrimaryKey(name, table)})
}

// DropPrimaryKeyIfExists appends DROP PRIMARY KEY when the named constraint exists.
func (p *SchemaPlan) DropPrimaryKeyIfExists(ctx context.Context, name, table string) *SchemaPlan {
	exists, err := p.ctx.ConstraintExists(ctx, table, name)
	return p.addIf(exists, err, SQLStatement{Query: p.ctx.dialect.DropPrimaryKey(name, table)})
}

// AddForeignKey appends an ADD FOREIGN KEY statement.
func (p *SchemaPlan) AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.AddForeignKey(name, table, columns, refTable, refColumns, onDelete, onUpdate)})
}

// AddForeignKeyIfNotExists appends ADD FOREIGN KEY when the foreign key does not exist.
func (p *SchemaPlan) AddForeignKeyIfNotExists(ctx context.Context, name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) *SchemaPlan {
	exists, err := p.ctx.ForeignKeyExists(ctx, table, name)
	return p.addIf(!exists, err, SQLStatement{Query: p.ctx.dialect.AddForeignKey(name, table, columns, refTable, refColumns, onDelete, onUpdate)})
}

// DropForeignKey appends a DROP FOREIGN KEY statement.
func (p *SchemaPlan) DropForeignKey(name, table string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropForeignKey(name, table)})
}

// DropForeignKeyIfExists appends DROP FOREIGN KEY when the foreign key exists.
func (p *SchemaPlan) DropForeignKeyIfExists(ctx context.Context, name, table string) *SchemaPlan {
	exists, err := p.ctx.ForeignKeyExists(ctx, table, name)
	return p.addIf(exists, err, SQLStatement{Query: p.ctx.dialect.DropForeignKey(name, table)})
}

// AddCommentOnColumn appends a column comment statement.
func (p *SchemaPlan) AddCommentOnColumn(table, column, comment string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.AddCommentOnColumn(table, column, comment)})
}

// DropCommentFromColumn appends a drop column comment statement.
func (p *SchemaPlan) DropCommentFromColumn(table, column string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropCommentFromColumn(table, column)})
}

// AddCommentOnTable appends a table comment statement.
func (p *SchemaPlan) AddCommentOnTable(table, comment string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.AddCommentOnTable(table, comment)})
}

// DropCommentFromTable appends a drop table comment statement.
func (p *SchemaPlan) DropCommentFromTable(table string) *SchemaPlan {
	return p.add(SQLStatement{Query: p.ctx.dialect.DropCommentFromTable(table)})
}

// Insert appends an INSERT statement.
func (p *SchemaPlan) Insert(table string, row Row) *SchemaPlan {
	query, args := p.ctx.dialect.Insert(table, row)
	return p.add(SQLStatement{Query: query, Args: args})
}

// BatchInsert appends a multi-row INSERT statement.
func (p *SchemaPlan) BatchInsert(table string, columns []string, rows [][]any) *SchemaPlan {
	query, args := p.ctx.dialect.BatchInsert(table, columns, rows)
	return p.add(SQLStatement{Query: query, Args: args})
}

// Update appends an UPDATE statement.
func (p *SchemaPlan) Update(table string, row Row, condition string, args ...any) *SchemaPlan {
	query, bound := p.ctx.dialect.Update(table, row, condition, args...)
	return p.add(SQLStatement{Query: query, Args: bound})
}

// Delete appends a DELETE statement.
func (p *SchemaPlan) Delete(table string, condition string, args ...any) *SchemaPlan {
	query, bound := p.ctx.dialect.Delete(table, condition, args...)
	return p.add(SQLStatement{Query: query, Args: bound})
}

// Exec executes all statements in order.
func (p *SchemaPlan) Exec(ctx context.Context) error {
	if p.err != nil {
		return p.err
	}
	for _, statement := range p.statements {
		if err := p.ctx.exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (p *SchemaPlan) add(statement SQLStatement) *SchemaPlan {
	p.statements = append(p.statements, statement)
	return p
}

func (p *SchemaPlan) addIf(ok bool, err error, statement SQLStatement) *SchemaPlan {
	if err != nil && p.err == nil {
		p.err = err
		return p
	}
	if ok {
		p.add(statement)
	}
	return p
}

func optionString(options []string) string {
	if len(options) == 0 {
		return ""
	}
	return options[0]
}
