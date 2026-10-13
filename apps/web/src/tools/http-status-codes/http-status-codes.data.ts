/**
 * IANA HTTP Status Code Registry (https://www.iana.org/assignments/http-status-codes), with the
 * defining RFC section. Descriptions are short summaries.
 */
export interface StatusCode {
  code: number
  phrase: string
  rfc: number
  section: string
  description: string
}

function S(code: number, phrase: string, rfc: number, section: string, description: string): StatusCode {
  return { code, phrase, rfc, section, description }
}

export const STATUS_CODES: StatusCode[] = [
  S(100, 'Continue', 9110, '15.2.1', 'The initial part of the request was received; the client should send the body.'),
  S(101, 'Switching Protocols', 9110, '15.2.2', 'The server switches to the protocol in the Upgrade header (e.g. WebSocket).'),
  S(102, 'Processing', 2518, '10.1', 'WebDAV: request accepted, still processing (deprecated).'),
  S(103, 'Early Hints', 8297, '2', 'Preliminary headers (e.g. Link preload) before the final response.'),
  S(200, 'OK', 9110, '15.3.1', 'The request succeeded.'),
  S(201, 'Created', 9110, '15.3.2', 'A new resource was created; Location points to it.'),
  S(202, 'Accepted', 9110, '15.3.3', 'Accepted for processing, not completed yet.'),
  S(203, 'Non-Authoritative Information', 9110, '15.3.4', 'Success, but the payload was modified by a transforming proxy.'),
  S(204, 'No Content', 9110, '15.3.5', 'Success with no response body.'),
  S(205, 'Reset Content', 9110, '15.3.6', 'Success; the client should reset the document view (e.g. clear a form).'),
  S(206, 'Partial Content', 9110, '15.3.7', 'Range request fulfilled with part of the representation.'),
  S(207, 'Multi-Status', 4918, '11.1', 'WebDAV: multiple status values in an XML body.'),
  S(208, 'Already Reported', 5842, '7.1', 'WebDAV: members already listed earlier in this response.'),
  S(226, 'IM Used', 3229, '10.4.1', 'Delta encoding: result of instance manipulations.'),
  S(300, 'Multiple Choices', 9110, '15.4.1', 'Several representations are available.'),
  S(301, 'Moved Permanently', 9110, '15.4.2', 'Permanent redirect; clients may change POST to GET.'),
  S(302, 'Found', 9110, '15.4.3', 'Temporary redirect; clients may change POST to GET.'),
  S(303, 'See Other', 9110, '15.4.4', 'Redirect to another resource, always retrieved with GET.'),
  S(304, 'Not Modified', 9110, '15.4.5', 'Conditional request: the cached copy is still valid.'),
  S(305, 'Use Proxy', 9110, '15.4.6', 'Deprecated; must not be used.'),
  S(306, '(Unused)', 9110, '15.4.7', 'Reserved; no longer used.'),
  S(307, 'Temporary Redirect', 9110, '15.4.8', 'Temporary redirect; method and body must not change.'),
  S(308, 'Permanent Redirect', 9110, '15.4.9', 'Permanent redirect; method and body must not change.'),
  S(400, 'Bad Request', 9110, '15.5.1', 'The request is malformed or invalid.'),
  S(401, 'Unauthorized', 9110, '15.5.2', 'Authentication is required (see WWW-Authenticate).'),
  S(402, 'Payment Required', 9110, '15.5.3', 'Reserved for future use.'),
  S(403, 'Forbidden', 9110, '15.5.4', 'Understood but refused; authenticating will not help.'),
  S(404, 'Not Found', 9110, '15.5.5', 'No representation found for the target resource.'),
  S(405, 'Method Not Allowed', 9110, '15.5.6', 'The method is not supported for this resource (see Allow).'),
  S(406, 'Not Acceptable', 9110, '15.5.7', 'No representation matches the Accept headers.'),
  S(407, 'Proxy Authentication Required', 9110, '15.5.8', 'The client must authenticate with the proxy.'),
  S(408, 'Request Timeout', 9110, '15.5.9', 'The server timed out waiting for the request.'),
  S(409, 'Conflict', 9110, '15.5.10', 'Conflicts with the current state of the resource.'),
  S(410, 'Gone', 9110, '15.5.11', 'The resource is permanently gone.'),
  S(411, 'Length Required', 9110, '15.5.12', 'Content-Length is required.'),
  S(412, 'Precondition Failed', 9110, '15.5.13', 'A conditional header (If-Match, …) evaluated to false.'),
  S(413, 'Content Too Large', 9110, '15.5.14', 'The request body is larger than the server allows.'),
  S(414, 'URI Too Long', 9110, '15.5.15', 'The request target is longer than the server allows.'),
  S(415, 'Unsupported Media Type', 9110, '15.5.16', 'The content type or encoding is not supported.'),
  S(416, 'Range Not Satisfiable', 9110, '15.5.17', 'The requested range cannot be served.'),
  S(417, 'Expectation Failed', 9110, '15.5.18', 'The Expect header cannot be met.'),
  S(418, '(Unused)', 9110, '15.5.19', 'Reserved (the "I\'m a teapot" joke code); not to be assigned.'),
  S(421, 'Misdirected Request', 9110, '15.5.20', 'Sent to a server that cannot answer for this origin (e.g. reused TLS connection).'),
  S(422, 'Unprocessable Content', 9110, '15.5.21', 'Well-formed but semantically invalid content.'),
  S(423, 'Locked', 4918, '11.3', 'WebDAV: the resource is locked.'),
  S(424, 'Failed Dependency', 4918, '11.4', 'WebDAV: failed because a previous action failed.'),
  S(425, 'Too Early', 8470, '5.2', 'Refused to process data sent in TLS early data (0-RTT).'),
  S(426, 'Upgrade Required', 9110, '15.5.22', 'The client must switch to another protocol (see Upgrade).'),
  S(428, 'Precondition Required', 6585, '3', 'The request must be conditional (to avoid lost updates).'),
  S(429, 'Too Many Requests', 6585, '4', 'Rate limited; see Retry-After.'),
  S(431, 'Request Header Fields Too Large', 6585, '5', 'Headers are too large.'),
  S(451, 'Unavailable For Legal Reasons', 7725, '3', 'Blocked for legal reasons (e.g. censorship, court order).'),
  S(500, 'Internal Server Error', 9110, '15.6.1', 'An unexpected condition prevented fulfilling the request.'),
  S(501, 'Not Implemented', 9110, '15.6.2', 'The server does not support the functionality required.'),
  S(502, 'Bad Gateway', 9110, '15.6.3', 'A gateway or proxy received an invalid response upstream.'),
  S(503, 'Service Unavailable', 9110, '15.6.4', 'Temporarily overloaded or down for maintenance; see Retry-After.'),
  S(504, 'Gateway Timeout', 9110, '15.6.5', 'A gateway or proxy timed out waiting for the upstream.'),
  S(505, 'HTTP Version Not Supported', 9110, '15.6.6', 'The HTTP major version is not supported.'),
  S(506, 'Variant Also Negotiates', 2295, '8.1', 'Content negotiation misconfiguration (circular reference).'),
  S(507, 'Insufficient Storage', 4918, '11.5', 'WebDAV: not enough storage to complete the request.'),
  S(508, 'Loop Detected', 5842, '7.2', 'WebDAV: infinite loop detected while processing.'),
  S(510, 'Not Extended (OBSOLETED)', 2774, '7', 'Further extensions are required (RFC 2774 is historic).'),
  S(511, 'Network Authentication Required', 6585, '6', 'Captive portal: authenticate to gain network access.'),
]
