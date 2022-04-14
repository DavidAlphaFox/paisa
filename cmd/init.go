package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"time"

	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/scraper/mutualfund"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/google/btree"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"strings"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "generates a sample config and journal file",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		generateConfigFile(cwd)
		generateJournalFile(cwd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func generateConfigFile(cwd string) {
	configFilePath := path.Join(cwd, "paisa.yaml")
	config := `
journal_path: "%s"
db_path: "%s"
commodities:
  - name: NIFTY
    type: mutualfund
    code: 120716
  - name: NIFTY_JR
    type: mutualfund
    code: 120684
`
	log.Info("Generating config file: ", configFilePath)
	journalFilePath := path.Join(cwd, "personal.ledger")
	dbFilePath := path.Join(cwd, "paisa.db")
	err := ioutil.WriteFile(configFilePath, []byte(fmt.Sprintf(config, journalFilePath, dbFilePath)), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func loadPrices(schemeCode string, commodityName string, pricesTree map[string]*btree.BTree) {
	prices, err := mutualfund.GetNav(schemeCode, commodityName)
	if err != nil {
		log.Fatal(err)
	}

	pricesTree[commodityName] = btree.New(2)
	for _, price := range prices {
		pricesTree[commodityName].ReplaceOrInsert(*price)

	}
}

func formatFloat(num float64) string {
	s := fmt.Sprintf("%.4f", num)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

func generateJournalFile(cwd string) {
	journalFilePath := path.Join(cwd, "personal.ledger")
	log.Info("Generating journal file: ", journalFilePath)
	ledgerFile, err := os.OpenFile(journalFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	end := time.Now()
	start, err := time.Parse("02-01-2006", "01-01-2019")
	if err != nil {
		log.Fatal(err)
	}

	pricesTree := make(map[string]*btree.BTree)
	loadPrices("120716", "NIFTY", pricesTree)
	loadPrices("120684", "NIFTY_JR", pricesTree)

	for ; start.Before(end); start = start.AddDate(0, 1, 0) {
		// salary
		var salary float64 = 100000 + (float64(start.Year())-2019)*20000
		_, err := ledgerFile.WriteString(fmt.Sprintf(`
%s Salary
    Income:Salary
    Asset:Debt:EPF                  %s INR
    Checking                        %s INR
`, start.Format("2006/01/02"), formatFloat(salary*0.12), formatFloat(salary*0.88)))
		if err != nil {
			log.Fatal(err)
		}

		// mutual fund
		pc := utils.BTreeDescendFirstLessOrEqual(pricesTree["NIFTY"], price.Price{Date: start})
		_, err = ledgerFile.WriteString(fmt.Sprintf(`
%s Mutual Fund Nifty
    Asset:Equity:NIFTY   %s NIFTY @ %s INR
    Checking
`, start.Format("2006/01/02"), formatFloat(10000/pc.Value), formatFloat(pc.Value)))
		if err != nil {
			log.Fatal(err)
		}

		pc = utils.BTreeDescendFirstLessOrEqual(pricesTree["NIFTY_JR"], price.Price{Date: start})
		_, err = ledgerFile.WriteString(fmt.Sprintf(`
%s Mutual Fund Nifty Next 50
    Asset:Equity:NIFTY_JR   %s NIFTY_JR @ %s INR
    Checking
`, start.Format("2006/01/02"), formatFloat(10000/pc.Value), formatFloat(pc.Value)))
		if err != nil {
			log.Fatal(err)
		}
	}
}