import Foundation
import os
import SwiftGraphQLClient

nonisolated protocol KindredOperationGraphQLClient: Sendable {
    func fetch<Query: GraphQLQuery>(
        _ query: Query,
        cachePolicy: CachePolicy.Query.SingleResponse
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat

    func perform<Mutation: GraphQLMutation>(
        _ mutation: Mutation
    ) async throws -> Mutation.Data where Mutation.ResponseFormat == SingleResponseFormat
}

extension KindredOperationGraphQLClient {
    func fetch<Query: GraphQLQuery>(
        _ query: Query
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat {
        try await fetch(query, cachePolicy: .networkOnly)
    }
}

nonisolated final class SwiftKindredGraphQLClient: KindredOperationGraphQLClient, @unchecked Sendable {
    private let client: SwiftGraphQLClient.GraphQLClient

    init(
        configuration: GraphQLClientConfiguration,
        urlSessionConfiguration: URLSessionConfiguration = .default,
        tokenStore: AuthTokenStore? = nil
    ) {
        let authProvider = KindredPackageGraphQLAuthProvider(
            authMode: configuration.authMode,
            tokenStore: tokenStore
        )
        let packageConfiguration = SwiftGraphQLClient.GraphQLClientConfiguration(
            endpointURL: configuration.endpointURL,
            authProvider: authProvider,
            additionalHeaders: configuration.additionalHeaders,
            deviceFingerprint: configuration.deviceFingerprint ?? tokenStore?.deviceID,
            deviceFingerprintHeaderName: configuration.deviceFingerprintHeaderName
        )
        client = SwiftGraphQLClient.GraphQLClient(
            configuration: packageConfiguration,
            session: URLSession(configuration: urlSessionConfiguration),
            encoder: APICoding.encoder(),
            decoder: APICoding.decoder()
        )
    }

    init(client: SwiftGraphQLClient.GraphQLClient) {
        self.client = client
    }

    func fetch<Query: GraphQLQuery>(
        _ query: Query,
        cachePolicy: CachePolicy.Query.SingleResponse = .networkOnly
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat {
        do {
            return try await client.fetch(query, cachePolicy: cachePolicy)
        } catch {
            throw Self.kindredError(from: error)
        }
    }

    func perform<Mutation: GraphQLMutation>(
        _ mutation: Mutation
    ) async throws -> Mutation.Data where Mutation.ResponseFormat == SingleResponseFormat {
        do {
            return try await client.perform(mutation)
        } catch {
            throw Self.kindredError(from: error)
        }
    }

    private static func kindredError(from error: Error) -> Error {
        guard let clientError = error as? GraphQLClientError else { return error }
        switch clientError {
        case .invalidResponse, .missingData, .cacheMiss, .partialCacheHit, .unsupportedCachePolicy, .unsupportedSubscriptions:
            return KindredGraphQLClientError.invalidResponse
        case .httpStatus(let statusCode, _):
            return KindredGraphQLClientError.unsuccessfulStatusCode(statusCode)
        case .graphQLErrors(let errors):
            return KindredGraphQLClientError.graphQLErrors(errors.map(KindredGraphQLError.init(graphQLError:)))
        }
    }
}

nonisolated final class RefreshingKindredGraphQLClient: KindredOperationGraphQLClient, @unchecked Sendable {
    private static let logger = Logger(subsystem: "com.radlof.kindred-swift", category: "Auth")

    private let baseClient: KindredOperationGraphQLClient
    private let tokenStore: AuthTokenStore
    private let sessionRefresher: SessionRefreshing?
    private let refreshCoordinator = RefreshCoordinator()

    init(
        baseClient: KindredOperationGraphQLClient,
        tokenStore: AuthTokenStore,
        sessionRefresher: SessionRefreshing?
    ) {
        self.baseClient = baseClient
        self.tokenStore = tokenStore
        self.sessionRefresher = sessionRefresher
    }

    func fetch<Query: GraphQLQuery>(
        _ query: Query,
        cachePolicy: CachePolicy.Query.SingleResponse
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat {
        try await executeWithRefresh(operationName: Query.operationName) {
            try await baseClient.fetch(query, cachePolicy: cachePolicy)
        }
    }

    func perform<Mutation: GraphQLMutation>(
        _ mutation: Mutation
    ) async throws -> Mutation.Data where Mutation.ResponseFormat == SingleResponseFormat {
        try await executeWithRefresh(operationName: Mutation.operationName) {
            try await baseClient.perform(mutation)
        }
    }

    private func executeWithRefresh<Output>(
        operationName: String,
        didRefresh: Bool = false,
        operation: () async throws -> Output
    ) async throws -> Output {
        do {
            return try await operation()
        } catch {
            guard isUnauthorized(error) else { throw error }
            if !didRefresh, try await refreshSession() {
                return try await executeWithRefresh(
                    operationName: operationName,
                    didRefresh: true,
                    operation: operation
                )
            }
            let reason = authClearReason(didRefresh: didRefresh)
            Self.logger.warning("Clearing auth session reason=\(reason, privacy: .public) graphql_operation=\(operationName, privacy: .public)")
            tokenStore.clearAuthToken()
            throw error
        }
    }

    private func refreshSession() async throws -> Bool {
        guard let refreshToken = tokenStore.refreshToken, !refreshToken.isEmpty else {
            Self.logger.warning("GraphQL refresh skipped reason=missing_refresh_token")
            return false
        }
        let preRefreshAccess = tokenStore.authToken
        let sessionRefresher = self.sessionRefresher
        let deviceID = tokenStore.deviceID
        let logger = Self.logger
        return try await refreshCoordinator.refresh { [tokenStore, logger, sessionRefresher] in
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

    private func authClearReason(didRefresh: Bool) -> String {
        if didRefresh {
            return "retried_graphql_request_unauthorized"
        }
        if tokenStore.refreshToken?.isEmpty ?? true {
            return "missing_refresh_token"
        }
        if sessionRefresher == nil {
            return "missing_session_refresher"
        }
        return "graphql_refresh_rejected"
    }

    private func isUnauthorized(_ error: Error) -> Bool {
        if let graphQLError = error as? KindredGraphQLClientError {
            return graphQLError.isUnauthorized
        }
        if let graphQLError = error as? GraphQLClientError {
            return graphQLError.isUnauthorized
        }
        if case APIClientError.unauthorized = error {
            return true
        }
        if case APIClientError.server(_, _, let status, _) = error, status == 401 {
            return true
        }
        let nsError = error as NSError
        return nsError.domain == NSURLErrorDomain
            && nsError.code == URLError.userAuthenticationRequired.rawValue
    }
}

private final class KindredPackageGraphQLAuthProvider: GraphQLAuthProvider, @unchecked Sendable {
    let authMode: GraphQLAuthMode
    weak var tokenStore: AuthTokenStore?

    init(authMode: GraphQLAuthMode, tokenStore: AuthTokenStore?) {
        self.authMode = authMode
        self.tokenStore = tokenStore
    }

    func graphQLAuthorizationHeaders() async throws -> [String: String] {
        switch authMode.strategy {
        case .none:
            return [:]
        case .bearer:
            guard let token = tokenStore?.authToken, !token.isEmpty else { return [:] }
            return ["Authorization": "Bearer \(token)"]
        case .apiKey(let headerName, let value):
            guard !headerName.isEmpty, !value.isEmpty else { return [:] }
            return [headerName: value]
        }
    }
}

private extension KindredGraphQLError {
    init(graphQLError: GraphQLError) {
        self.init(
            message: graphQLError.message,
            code: graphQLError.code,
            status: graphQLError.statusCode
        )
    }

    var isUnauthorized: Bool {
        if extensions?.status == 401 {
            return true
        }
        guard let code = extensions?.code?.uppercased() else {
            return false
        }
        return [
            "UNAUTHENTICATED",
            "UNAUTHORIZED",
            "UNAUTHORIZEDEXCEPTION",
            "TOKEN_EXPIRED",
            "JWT_EXPIRED"
        ].contains(code)
    }
}

private extension KindredGraphQLClientError {
    var isUnauthorized: Bool {
        switch self {
        case .unsuccessfulStatusCode(let statusCode):
            return statusCode == 401
        case .graphQLErrors(let errors):
            return errors.contains { $0.isUnauthorized }
        case .invalidResponse:
            return false
        }
    }
}

enum KindredGraphQLScalars {
    nonisolated static func date(from value: String?) -> Date? {
        guard let value else { return nil }
        let iso8601WithFractionalSeconds = ISO8601DateFormatter()
        iso8601WithFractionalSeconds.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let iso8601 = ISO8601DateFormatter()
        iso8601.formatOptions = [.withInternetDateTime]
        return iso8601WithFractionalSeconds.date(from: value) ?? iso8601.date(from: value)
    }
}
