package emitter

import (
	"fmt"
	"strings"

	"github.com/huderlem/poryscript/ast"
)

// Represents a single chunk of script output. Each chunk has an associated label in
// the emitted bytecode output.
type chunk struct {
	id             int
	returnID       int
	statements     []ast.Statement
	branchBehavior brancher
}

func (c *chunk) renderLabel(scriptName string, sb *strings.Builder) {
	if c.id == 0 {
		// Main script entrypoint, so it gets a global label.
		sb.WriteString(fmt.Sprintf("%s::\n", scriptName))
	} else {
		sb.WriteString(fmt.Sprintf("%s_%d:\n", scriptName, c.id))
	}
}

func (c *chunk) renderStatements(sb *strings.Builder) {
	// Render basic non-branching commands.
	for _, stmt := range c.statements {
		commandStmt, ok := stmt.(*ast.CommandStatement)
		if !ok {
			panic(fmt.Sprintf("Could not render chunk statement because it is not a command statement %q", stmt.TokenLiteral()))
		}

		sb.WriteString(renderCommandStatement(commandStmt))
	}
}

func (c *chunk) renderBranching(scriptName string, sb *strings.Builder) {
	requiresTailJump := true
	if c.branchBehavior != nil {
		c.branchBehavior.renderBranchConditions(sb, scriptName)
		requiresTailJump = c.branchBehavior.requiresTailJump()
	}
	// Sometimes, a tail jump/return isn't needed.  For example, a chunk that ends in an "else"
	// branch will always naturally end with a "goto" bytecode command.
	if requiresTailJump {
		if c.returnID == -1 {
			sb.WriteString("\treturn\n")
		} else {
			sb.WriteString(fmt.Sprintf("\tgoto %s_%d\n", scriptName, c.returnID))
		}
	}
}

func (c *chunk) splitChunkForBranch(statementIndex int, chunkCounter *int, remainingChunks []*chunk) ([]*chunk, int) {
	var returnID int
	if c.isLastStatement(statementIndex) {
		// The statement is the last of the current chunk, so it
		// has the same return point as the current chunk.
		returnID = c.returnID
	} else {
		// The statement needs to return to a chunk of logic
		// that occurs directly after it. So, create a new Chunk for
		// that logic.
		*chunkCounter++
		newChunk := c.createPostLogicChunk(*chunkCounter, statementIndex)
		remainingChunks = append(remainingChunks, newChunk)
		returnID = newChunk.id
		c.returnID = newChunk.id
	}
	return remainingChunks, returnID
}

func (c *chunk) isLastStatement(statementIndex int) bool {
	return statementIndex == len(c.statements)-1
}

func (c *chunk) createPostLogicChunk(id int, lastStatementIndex int) *chunk {
	newChunk := &chunk{
		id:         id,
		returnID:   c.returnID,
		statements: c.statements[lastStatementIndex+1:],
	}
	return newChunk
}

func renderCommandStatement(commandStmt *ast.CommandStatement) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\t%s", commandStmt.Name.Value))
	if len(commandStmt.Args) > 0 {
		sb.WriteString(fmt.Sprintf(" %s", strings.Join(commandStmt.Args, ", ")))
	}
	sb.WriteString("\n")
	return sb.String()
}
