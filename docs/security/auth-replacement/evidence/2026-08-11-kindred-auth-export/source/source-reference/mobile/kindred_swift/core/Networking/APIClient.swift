import SwiftGraphQLClient
import Foundation
import os

nonisolated enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
    case patch = "PATCH"
}

nonisolated struct Endpoint<Response: Decodable> {
    let path: String
    let method: HTTPMethod
    let queryItems: [URLQueryItem]
    let headers: [String: String]
    let body: Data?

    init(
        path: String,
        method: HTTPMethod = .get,
        queryItems: [URLQueryItem] = [],
        headers: [String: String] = [:],
        body: Data? = nil
    ) {
        self.path = path
        self.method = method
        self.queryItems = queryItems
        self.headers = headers
        self.body = body
    }
}

nonisolated protocol APIClient {
    func send<Response: Decodable>(_ endpoint: Endpoint<Response>) async throws -> Response
}

nonisolated protocol URLSessioning {
    func data(for request: URLRequest) async throws -> (Data, URLResponse)
}

nonisolated extension URLSession: URLSessioning {}

nonisolated protocol SessionRefreshing: Sendable {
    func refreshSession(refreshToken: String, deviceID: String) async throws -> AuthResponseDTO?
}

nonisolated struct GraphQLSessionRefresher: SessionRefreshing {
    let graphQLClient: KindredOperationGraphQLClient

    func refreshSession(refreshToken: String, deviceID: String) async throws -> AuthResponseDTO? {
        do {
            let data = try await graphQLClient.perform(KindredAPI.RefreshSessionMutation(
                input: KindredAPI.RefreshSessionInput(
                    refreshToken: refreshToken,
                    deviceId: deviceID.isEmpty ? .none : .some(deviceID)
                )
            ))
            return data.refreshSession.fragments.authPayloadFields.toDTO()
        } catch KindredGraphQLClientError.graphQLErrors {
            return nil
        }
    }
}

/// Provides + persists the bearer token. Implemented by AppSettingsStore.
nonisolated protocol AuthTokenStore: AnyObject, Sendable {
    var authToken: String? { get }
    var refreshToken: String? { get }
    /// Stable per-install identifier sent on auth requests so the server can
    /// bind refresh tokens to a single device. Empty string disables binding.
    var deviceID: String { get }
    func setTokens(accessToken: String, refreshToken: String?)
    func clearAuthToken()
}

nonisolated struct APIErrorPayload: Decodable {
    struct Body: Decodable {
        let code: String
        let message: String
    }
    let error: Body
}

nonisolated enum APIClientError: LocalizedError {
    case invalidURL
    case invalidResponse
    case unsuccessfulStatusCode(Int)
    case unauthorized
    case server(code: String, message: String, status: Int, retryAfter: TimeInterval?)

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "The API URL is invalid."
        case .invalidResponse:
            return "The API response is invalid."
        case .unsuccessfulStatusCode(let code):
            return "The API returned an unsuccessful status code: \(code)."
        case .unauthorized:
            return "Unauthorized."
        case .server(_, let message, _, _):
            return message
        }
    }
}

nonisolated enum APICoding {
    static func encoder() -> JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .custom { date, encoder in
            var container = encoder.singleValueContainer()
            try container.encode(iso8601WithFractionalSeconds.string(from: date))
        }
        return encoder
    }

    static func decoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let value = try container.decode(String.self)
            if let date = iso8601WithFractionalSeconds.date(from: value)
                ?? iso8601.date(from: value) {
                return date
            }
            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Invalid ISO-8601 date: \(value)"
            )
        }
        return decoder
    }

    private static let iso8601: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()

    private static let iso8601WithFractionalSeconds: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()
}

/// Dedupes concurrent /auth/refresh calls. When several requests fail with 401
/// at the same time, only one network refresh runs; the rest await the same
/// `Task` and share its outcome.
actor RefreshCoordinator {
    private var inFlight: Task<Bool, Error>?

    func refresh(_ perform: @Sendable @escaping () async throws -> Bool) async throws -> Bool {
        if let task = inFlight {
            return try await task.value
        }
        let task = Task<Bool, Error> { try await perform() }
        inFlight = task
        defer { inFlight = nil }
        return try await task.value
    }
}

nonisolated struct URLSessionAPIClient: APIClient {
    private static let logger = Logger(subsystem: "com.radlof.kindred-swift", category: "Auth")

    let baseURL: URL
    let session: URLSessioning
    let decoder: JSONDecoder
    let tokenStore: AuthTokenStore?
    let maxRetries: Int
    let sessionRefresher: SessionRefreshing?
    private let refreshCoordinator = RefreshCoordinator()

    init(
        baseURL: URL,
        session: URLSessioning = URLSession.shared,
        decoder: JSONDecoder = APICoding.decoder(),
        tokenStore: AuthTokenStore? = nil,
        maxRetries: Int = 2,
        sessionRefresher: SessionRefreshing? = nil
    ) {
        self.baseURL = baseURL
        self.session = session
        self.decoder = decoder
        self.tokenStore = tokenStore
        self.maxRetries = maxRetries
        self.sessionRefresher = sessionRefresher
    }

    func send<Response: Decodable>(_ endpoint: Endpoint<Response>) async throws -> Response {
        try await send(endpoint, attempt: 0, didRefresh: false)
    }

    private func send<Response: Decodable>(
        _ endpoint: Endpoint<Response>,
        attempt: Int,
        didRefresh: Bool
    ) async throws -> Response {
        var components = URLComponents(
            url: baseURL.appending(path: endpoint.path),
            resolvingAgainstBaseURL: false
        )
        components?.queryItems = endpoint.queryItems.isEmpty ? nil : endpoint.queryItems

        guard let url = components?.url else {
            throw APIClientError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        if let body = endpoint.body {
            request.httpBody = body
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }

        for (key, value) in endpoint.headers {
            request.setValue(value, forHTTPHeaderField: key)
        }

        if let token = tokenStore?.authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if let deviceID = tokenStore?.deviceID, !deviceID.isEmpty {
            request.setValue(deviceID, forHTTPHeaderField: "X-Device-Fingerprint")
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            if shouldRetry(endpoint: endpoint, attempt: attempt, error: error) {
                try await sleepBeforeRetry(attempt: attempt, retryAfter: nil)
                return try await send(endpoint, attempt: attempt + 1, didRefresh: didRefresh)
            }
            throw error
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIClientError.invalidResponse
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            let retryAfter: TimeInterval? = (httpResponse.value(forHTTPHeaderField: "Retry-After"))
                .flatMap { Int($0.trimmingCharacters(in: .whitespaces)) }
                .map { TimeInterval($0) }
            if shouldRetry(endpoint: endpoint, attempt: attempt, statusCode: httpResponse.statusCode) {
                try await sleepBeforeRetry(attempt: attempt, retryAfter: retryAfter)
                return try await send(endpoint, attempt: attempt + 1, didRefresh: didRefresh)
            }
            if httpResponse.statusCode == 401 {
                if !didRefresh, !isAuthEndpoint(endpoint.path), try await refreshSession() {
                    return try await send(endpoint, attempt: attempt, didRefresh: true)
                }
                let reason = authClearReason(endpoint: endpoint, didRefresh: didRefresh)
                Self.logger.warning("Clearing auth session reason=\(reason, privacy: .public) endpoint=\(endpoint.path, privacy: .public) status=\(httpResponse.statusCode)")
                tokenStore?.clearAuthToken()
            }
            if let payload = try? decoder.decode(APIErrorPayload.self, from: data) {
                throw APIClientError.server(
                    code: payload.error.code,
                    message: payload.error.message,
                    status: httpResponse.statusCode,
                    retryAfter: retryAfter
                )
            }
            if httpResponse.statusCode == 401 {
                throw APIClientError.unauthorized
            }
            throw APIClientError.unsuccessfulStatusCode(httpResponse.statusCode)
        }

        if data.isEmpty, Response.self == EmptyResponseDTO.self {
            return try decoder.decode(Response.self, from: Data("{}".utf8))
        }

        return try decoder.decode(Response.self, from: data)
    }

    private func refreshSession() async throws -> Bool {
        // Capture the refresh token at call time so concurrent waiters that
        // share the in-flight refresh task observe the same input.
        guard let tokenStore, let refreshToken = tokenStore.refreshToken, !refreshToken.isEmpty else {
            Self.logger.warning("Refresh skipped reason=missing_refresh_token")
            return false
        }
        let preRefreshAccess = tokenStore.authToken
        let sessionRefresher = self.sessionRefresher
        let deviceID = tokenStore.deviceID
        let logger = Self.logger
        return try await refreshCoordinator.refresh { [tokenStore, logger, sessionRefresher] in
            // If another refresh already swapped the access token while we
            // were waiting for the lock, treat as success without re-hitting
            // the network.
            if tokenStore.authToken != preRefreshAccess, tokenStore.authToken != nil {
                return true
            }
            guard let sessionRefresher else {
                logger.warning("GraphQL refresh skipped reason=missing_session_refresher")
                return false
            }
            guard let auth = try await sessionRefresher.refreshSession(
                refreshToken: refreshToken,
                deviceID: deviceID
            ) else {
                logger.warning("GraphQL refresh rejected")
                return false
            }
            if auth.refreshToken == nil {
                logger.info("GraphQL refresh response omitted refresh token; preserving existing refresh token")
            }
            tokenStore.setTokens(accessToken: auth.token, refreshToken: auth.refreshToken ?? refreshToken)
            return true
        }
    }

    private func isAuthEndpoint(_ path: String) -> Bool {
        path.hasPrefix("/auth/")
    }

    private func authClearReason<Response: Decodable>(endpoint: Endpoint<Response>, didRefresh: Bool) -> String {
        if isAuthEndpoint(endpoint.path) {
            return "auth_endpoint_unauthorized"
        }
        if didRefresh {
            return "retried_request_unauthorized"
        }
        if tokenStore?.refreshToken?.isEmpty ?? true {
            return "missing_refresh_token"
        }
        return "refresh_rejected"
    }

    private func shouldRetry(endpoint: Endpoint<some Decodable>, attempt: Int, statusCode: Int) -> Bool {
        endpoint.method == .get
            && attempt < maxRetries
            && [408, 429, 500, 502, 503, 504].contains(statusCode)
    }

    private func shouldRetry(endpoint: Endpoint<some Decodable>, attempt: Int, error: Error) -> Bool {
        guard endpoint.method == .get, attempt < maxRetries else { return false }
        guard let urlError = error as? URLError else { return false }
        switch urlError.code {
        case .timedOut, .networkConnectionLost, .cannotFindHost, .cannotConnectToHost, .notConnectedToInternet:
            return true
        default:
            return false
        }
    }

    private func sleepBeforeRetry(attempt: Int, retryAfter: TimeInterval?) async throws {
        let delay = retryAfter ?? min(0.4 * pow(2, Double(attempt)), 2.0)
        try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
    }
}

nonisolated struct StubAPIClient: APIClient {
    let responder: @Sendable (String) throws -> Data
    let decoder: JSONDecoder

    init(
        responder: @escaping @Sendable (String) throws -> Data,
        decoder: JSONDecoder = APICoding.decoder()
    ) {
        self.responder = responder
        self.decoder = decoder
    }

    func send<Response: Decodable>(_ endpoint: Endpoint<Response>) async throws -> Response {
        let data = try responder(endpoint.path)
        return try decoder.decode(Response.self, from: data)
    }
}
