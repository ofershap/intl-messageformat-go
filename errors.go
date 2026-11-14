package messageformat

import "errors"

// errCurrencyCodeRequired mirrors FormatJS's
// "TypeError: Currency code is required with currency style".
var errCurrencyCodeRequired = errors.New("messageformat: currency code is required with currency style")
