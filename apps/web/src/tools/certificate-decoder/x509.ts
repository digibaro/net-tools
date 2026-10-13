// @peculiar/x509 uses tsyringe, which needs the Reflect metadata polyfill evaluated first.
// Keep this module tiny so import sorting can never move the polyfill after the library.
import 'reflect-metadata'

export * from '@peculiar/x509'
