package chess

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
)

// GamesFromPGN returns all PGN decoding games from the
// reader.  It is designed to be used decoding multiple PGNs
// in the same file.  An error is returned if there is an
// issue parsing the PGNs.
func GamesFromPgnNoError(r io.Reader) ([]*Game, error) {
	games := []*Game{}
	current := ""
	count := 0
	totalCount := 0
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if strings.TrimSpace(line) == "" {
			count++
		} else {
			current += line
		}
		if count == 2 {
			game, err := decodePGN(current)
			if err != nil {
				// Ignore error games.
				continue
				//return nil, err
			}
			games = append(games, game)
			count = 0
			current = ""
			totalCount++
			log.Println("Processed game", totalCount)
		}
	}
	return games, nil
}

// GamesFromPGN returns all PGN decoding games from the
// reader.  It is designed to be used decoding multiple PGNs
// in the same file.  An error is returned if there is an
// issue parsing the PGNs.
func GamesFromPGN(r io.Reader) ([]*Game, error) {
	games := []*Game{}
	current := ""
	count := 0
	totalCount := 0
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if strings.TrimSpace(line) == "" {
			count++
		} else {
			current += line
		}
		if count == 2 {
			game, err := decodePGN(current)
			if err != nil {
				return nil, err
			}
			games = append(games, game)
			count = 0
			current = ""
			totalCount++
			log.Println("Processed game", totalCount)
		}
	}
	return games, nil
}

func decodePGN(pgn string) (*Game, error) {
	tagPairs := getTagPairs(pgn)
	moveStrs, outcome := moveList(pgn)
	g := NewGame(TagPairs(tagPairs))
	g.ignoreAutomaticDraws = true
	var notation Notation = AlgebraicNotation{}
	if len(moveStrs) > 0 {
		_, err := LongAlgebraicNotation{}.Decode(g.Position(), moveStrs[0])
		if err == nil {
			notation = LongAlgebraicNotation{}
		}
	}

	var prevMove *Move
	isComment := false
	commentStrs := []string{}
	
	for _, alg := range moveStrs {
		if strings.Contains(alg, "{") {
			isComment = true
			commentStrs = append(commentStrs, alg)
			continue
		}
		
		if isComment {
			if !strings.Contains(alg, "}") {
				commentStrs = append(commentStrs, alg)
				continue
			}
			// end of the comment
			commentStrs = append(commentStrs, alg)
			comment := strings.Join(commentStrs, " ")
			isComment = false
			if prevMove == nil {
				return nil, fmt.Errorf("chess: pgn decode error %s malformed comment")
			}
			prevMove.Comment = comment
			commentStrs = []string{}
			continue
		}
		
		m, err := notation.Decode(g.Position(), alg)
		
		if err != nil {
			return nil, fmt.Errorf("chess: pgn decode error %s on move %d", err.Error(), g.Position().moveCount)
		}
		if err := g.Move(m); err != nil {
			return nil, fmt.Errorf("chess: pgn invalid move error %s on move %d", err.Error(), g.Position().moveCount)
		}
		prevMove = m
	}
	g.outcome = outcome
	return g, nil
}

func encodePGN(g *Game) string {
	s := ""
	for _, tag := range g.tagPairs {
		s += fmt.Sprintf("[%s \"%s\"]\n", tag.Key, tag.Value)
	}
	s += "\n"
	for i, move := range g.moves {
		pos := g.positions[i]
		txt := g.notation.Encode(pos, move)
		if i%2 == 0 {
			s += fmt.Sprintf("%d.%s", (i/2)+1, txt)
		} else {
			s += fmt.Sprintf(" %s ", txt)
		}
	}
	s += " " + string(g.outcome)
	return s
}

var (
	tagPairRegex = regexp.MustCompile(`\[(.*)\s\"(.*)\"\]`)
)

func getTagPairs(pgn string) []*TagPair {
	tagPairs := []*TagPair{}
	matches := tagPairRegex.FindAllString(pgn, -1)
	for _, m := range matches {
		results := tagPairRegex.FindStringSubmatch(m)
		if len(results) == 3 {
			pair := &TagPair{
				Key:   results[1],
				Value: results[2],
			}
			tagPairs = append(tagPairs, pair)
		}
	}
	return tagPairs
}

var (
	moveNumRegex = regexp.MustCompile(`(?:\d+\.+)?(.*)`)
)

func moveList(pgn string) ([]string, Outcome) {
	// keep comments
	//text := removeSection("{", "}", pgn)

	// remove variations
	text := removeSection(`\(`, `\)`, pgn)

	// remove tag pairs
	text = removeTagPairs(text)
	
	// remove line breaks
	text = strings.Replace(text, "\n", " ", -1)

	
	list := strings.Split(text, " ")
	filtered := []string{}
	var outcome Outcome
	for _, move := range list {
		move = strings.TrimSpace(move)
		switch move {
		case string(NoOutcome), string(WhiteWon), string(BlackWon), string(Draw):
			outcome = Outcome(move)
		case "":
		default:
			results := moveNumRegex.FindStringSubmatch(move)
			if len(results) == 2 && results[1] != "" {
				filtered = append(filtered, results[1])
			}
		}
	}

	return filtered, outcome
}

func removeTagPairs(s string) string {
	r := regexp.MustCompile("(?m)^\\[.*?\\]$")
	return r.ReplaceAllString(s, "")
}

func removeSection(leftChar, rightChar, s string) string {
	r := regexp.MustCompile(leftChar + ".*?" + rightChar)
	for {
		i := r.FindStringIndex(s)
		if i == nil {
			return s
		}
		s = s[0:i[0]] + s[i[1]:len(s)]
	}
}
