package sqlite3

/*
// make go-sqlite3 use embedded library without code changes
#cgo CFLAGS: -DUSE_LIBSQLITE3

// enable encryption codec in sqlite
#cgo CFLAGS: -DSQLITE_HAS_CODEC

// SQLCipher 4.x refuses to compile without explicit init/shutdown
// hook registration. The amalgamation has a #error guard checking
// for these macros; passing them registers sqlcipher_extra_init /
// sqlcipher_extra_shutdown as the SQLite extension entry points.
#cgo CFLAGS: -DSQLITE_EXTRA_INIT=sqlcipher_extra_init
#cgo CFLAGS: -DSQLITE_EXTRA_SHUTDOWN=sqlcipher_extra_shutdown

// use memory for temporary storage in sqlite
#cgo CFLAGS: -DSQLITE_TEMP_STORE=2

// use libtomcrypt implementation in sqlcipher
#cgo CFLAGS: -DSQLCIPHER_CRYPTO_LIBTOMCRYPT

// disable anything "not portable" in libtomcrypt
#cgo CFLAGS: -DLTC_NO_ASM

// disable assertions
#cgo CFLAGS: -DNDEBUG

// set operating specific sqlite flags
#cgo linux CFLAGS: -DSQLITE_OS_UNIX=1
#cgo windows CFLAGS: -DSQLITE_OS_WIN=1
*/
import "C"
