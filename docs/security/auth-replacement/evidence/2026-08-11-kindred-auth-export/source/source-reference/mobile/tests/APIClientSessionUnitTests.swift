import SwiftGraphQLClient
import SwiftGraphQLClient
import Foundation
import Testing
@testable import kindred_swift

@Suite("API client session handling", .tags(.unit))
struct APIClientSessionUnitTests {
    @Test("401 without GraphQL refresher clears the stored session")
    func unauthorizedRequestWithoutGraphQLRefresherClearsSession() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 401, body: #"{"error":{"code":"TOKEN_EXPIRED","message":"expired"}}"#)
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let client = URLSessionAPIClient(
            baseURL: URL(string: "https://example.test")!,
            session: session,
            tokenStore: tokenStore
        )

        await #expect(throws: APIClientError.self) {
            let _: EmptyResponseDTO = try await client.send(Endpoint(path: "/me/session-check"))
        }

        #expect(tokenStore.authToken == nil)
        #expect(tokenStore.refreshToken == nil)
        #expect(tokenStore.clearCount == 1)
        #expect(session.requests.map(\.url?.path) == ["/me/session-check"])
        #expect(session.authorizationHeaders == ["Bearer expired-access"])
        #expect(session.refreshPayloads.isEmpty)
    }

    @Test("401 with accepted GraphQL refresh retries once and keeps the new session")
    func unauthorizedRequestWithAcceptedGraphQLRefreshRetriesWithNewToken() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 401, body: #"{"error":{"code":"TOKEN_EXPIRED","message":"expired"}}"#),
            .http(status: 200, body: "")
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let refresher = APIClientSessionRefresherFake(response: AuthResponseDTO(
            token: "fresh-access",
            expiresAt: nil,
            refreshToken: "fresh-refresh",
            refreshExpiresAt: nil,
            user: nil
        ))
        let client = URLSessionAPIClient(
            baseURL: URL(string: "https://example.test")!,
            session: session,
            tokenStore: tokenStore,
            sessionRefresher: refresher
        )

        let _: EmptyResponseDTO = try await client.send(Endpoint(path: "/me/session-check"))

        #expect(tokenStore.authToken == "fresh-access")
        #expect(tokenStore.refreshToken == "fresh-refresh")
        #expect(tokenStore.clearCount == 0)
        #expect(refresher.calls.map(\.refreshToken) == ["stale-refresh"])
        #expect(session.requests.map(\.url?.path) == ["/me/session-check", "/me/session-check"])
        #expect(session.authorizationHeaders == ["Bearer expired-access", "Bearer fresh-access"])
    }

    @Test("Accepted GraphQL refresh without refresh token preserves the existing refresh token")
    func acceptedGraphQLRefreshWithoutRefreshTokenPreservesExistingRefreshToken() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 401, body: #"{"error":{"code":"TOKEN_EXPIRED","message":"expired"}}"#),
            .http(status: 200, body: "")
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let refresher = APIClientSessionRefresherFake(response: AuthResponseDTO(
            token: "fresh-access",
            expiresAt: nil,
            refreshToken: nil,
            refreshExpiresAt: nil,
            user: nil
        ))
        let client = URLSessionAPIClient(
            baseURL: URL(string: "https://example.test")!,
            session: session,
            tokenStore: tokenStore,
            sessionRefresher: refresher
        )

        let _: EmptyResponseDTO = try await client.send(Endpoint(path: "/me/session-check"))

        #expect(tokenStore.authToken == "fresh-access")
        #expect(tokenStore.refreshToken == "stale-refresh")
        #expect(tokenStore.clearCount == 0)
        #expect(session.requests.map(\.url?.path) == ["/me/session-check", "/me/session-check"])
    }

    @Test("401 uses injected GraphQL session refresher when available")
    func unauthorizedRequestUsesInjectedSessionRefresherWhenAvailable() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 401, body: #"{"error":{"code":"TOKEN_EXPIRED","message":"expired"}}"#),
            .http(status: 200, body: "")
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let refresher = APIClientSessionRefresherFake(response: AuthResponseDTO(
            token: "fresh-access",
            expiresAt: nil,
            refreshToken: "fresh-refresh",
            refreshExpiresAt: nil,
            user: nil
        ))
        let client = URLSessionAPIClient(
            baseURL: URL(string: "https://example.test")!,
            session: session,
            tokenStore: tokenStore,
            sessionRefresher: refresher
        )

        let _: EmptyResponseDTO = try await client.send(Endpoint(path: "/me/session-check"))

        #expect(refresher.calls.map(\.refreshToken) == ["stale-refresh"])
        #expect(refresher.calls.map(\.deviceID) == ["device-1"])
        #expect(tokenStore.authToken == "fresh-access")
        #expect(tokenStore.refreshToken == "fresh-refresh")
        #expect(tokenStore.clearCount == 0)
        #expect(session.requests.map(\.url?.path) == ["/me/session-check", "/me/session-check"])
        #expect(session.authorizationHeaders == ["Bearer expired-access", "Bearer fresh-access"])
    }

    @Test("SwiftGraphQL session refresher uses generated RefreshSession mutation")
    func graphQLSessionRefresherUsesGeneratedRefreshSessionMutation() async throws {
        let graphQL = APIClientGraphQLSessionRefresherFake(response: refreshSessionSwiftGraphQLData())
        let refresher = GraphQLSessionRefresher(graphQLClient: graphQL)

        let auth = try await refresher.refreshSession(refreshToken: "stale-refresh", deviceID: "device-1")

        #expect(auth?.token == "fresh-access")
        #expect(auth?.refreshToken == "fresh-refresh")
        #expect(auth?.user?.id == "user-1")
        #expect(graphQL.operationNames == ["RefreshSession"])
        let input = try #require(graphQL.refreshInputs.first)
        #expect(input.refreshToken == "stale-refresh")
        #expect(input.deviceID == "device-1")
    }

    @Test("SwiftGraphQL session refresher treats GraphQL errors as rejected refresh")
    func graphQLSessionRefresherTreatsGraphQLErrorsAsRejectedRefresh() async throws {
        let graphQL = APIClientGraphQLSessionRefresherFake(error: KindredGraphQLClientError.graphQLErrors([
            KindredGraphQLError(message: "refresh rejected", code: "UNAUTHENTICATED", status: 401)
        ]))
        let refresher = GraphQLSessionRefresher(graphQLClient: graphQL)

        let auth = try await refresher.refreshSession(refreshToken: "stale-refresh", deviceID: "device-1")

        #expect(auth?.token == nil)
        #expect(graphQL.operationNames == ["RefreshSession"])
    }

    @Test("401 with rejected SwiftGraphQL refresh clears the stored session")
    func unauthorizedRequestWithRejectedSwiftGraphQLRefreshClearsSession() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 401, body: #"{"error":{"code":"TOKEN_EXPIRED","message":"expired"}}"#)
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let graphQL = APIClientGraphQLSessionRefresherFake(error: KindredGraphQLClientError.graphQLErrors([
            KindredGraphQLError(message: "refresh rejected", code: "UNAUTHENTICATED", status: 401)
        ]))
        let client = URLSessionAPIClient(
            baseURL: URL(string: "https://example.test")!,
            session: session,
            tokenStore: tokenStore,
            sessionRefresher: GraphQLSessionRefresher(graphQLClient: graphQL)
        )

        await #expect(throws: APIClientError.self) {
            let _: EmptyResponseDTO = try await client.send(Endpoint(path: "/me/session-check"))
        }

        #expect(tokenStore.authToken == nil)
        #expect(tokenStore.refreshToken == nil)
        #expect(tokenStore.clearCount == 1)
        #expect(session.requests.map(\.url?.path) == ["/me/session-check"])
        #expect(graphQL.operationNames == ["RefreshSession"])
    }

    @Test("Protected SwiftGraphQL query refreshes and retries once after HTTP 401")
    func protectedSwiftGraphQLQueryRefreshesAndRetriesAfterHTTP401() async throws {
        let baseSwiftGraphQL = RefreshingGraphQLClientFake(responses: [
            .failure(KindredGraphQLClientError.unsuccessfulStatusCode(401)),
            .success(homeSwiftGraphQLData())
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let refresher = APIClientSessionRefresherFake(response: AuthResponseDTO(
            token: "fresh-access",
            expiresAt: nil,
            refreshToken: "fresh-refresh",
            refreshExpiresAt: nil,
            user: nil
        ))
        let client = RefreshingKindredGraphQLClient(
            baseClient: baseSwiftGraphQL,
            tokenStore: tokenStore,
            sessionRefresher: refresher
        )

        let data = try await client.fetch(homeQuery())

        #expect(data.home.meStats.pointsAvailable == 42)
        #expect(baseSwiftGraphQL.operationNames == ["Home", "Home"])
        #expect(refresher.calls.map(\.refreshToken) == ["stale-refresh"])
        #expect(refresher.calls.map(\.deviceID) == ["device-1"])
        #expect(tokenStore.authToken == "fresh-access")
        #expect(tokenStore.refreshToken == "fresh-refresh")
        #expect(tokenStore.clearCount == 0)
    }

    @Test("Protected SwiftGraphQL query refreshes and retries once after GraphQL 401")
    func protectedSwiftGraphQLQueryRefreshesAndRetriesAfterGraphQL401() async throws {
        let baseSwiftGraphQL = RefreshingGraphQLClientFake(responses: [
            .failure(KindredGraphQLClientError.graphQLErrors([
                KindredGraphQLError(message: "expired", code: "TOKEN_EXPIRED", status: nil)
            ])),
            .success(homeSwiftGraphQLData())
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let refresher = APIClientSessionRefresherFake(response: AuthResponseDTO(
            token: "fresh-access",
            expiresAt: nil,
            refreshToken: nil,
            refreshExpiresAt: nil,
            user: nil
        ))
        let client = RefreshingKindredGraphQLClient(
            baseClient: baseSwiftGraphQL,
            tokenStore: tokenStore,
            sessionRefresher: refresher
        )

        let data = try await client.fetch(homeQuery())

        #expect(data.home.communityStats.availableItems == 7)
        #expect(baseSwiftGraphQL.operationNames == ["Home", "Home"])
        #expect(tokenStore.authToken == "fresh-access")
        #expect(tokenStore.refreshToken == "stale-refresh")
        #expect(tokenStore.clearCount == 0)
    }

    @Test("Protected SwiftGraphQL query clears session when refresh is rejected")
    func protectedSwiftGraphQLQueryClearsSessionWhenRefreshIsRejected() async throws {
        let baseSwiftGraphQL = RefreshingGraphQLClientFake(responses: [
            .failure(KindredGraphQLClientError.graphQLErrors([
                KindredGraphQLError(message: "Unauthenticated", code: "UNAUTHENTICATED", status: 401)
            ]))
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "expired-access",
            refreshToken: "stale-refresh",
            deviceID: "device-1"
        )
        let client = RefreshingKindredGraphQLClient(
            baseClient: baseSwiftGraphQL,
            tokenStore: tokenStore,
            sessionRefresher: APIClientSessionRefresherFake(response: nil)
        )

        await #expect(throws: KindredGraphQLClientError.self) {
            let _: KindredAPI.HomeQuery.Data = try await client.fetch(homeQuery())
        }

        #expect(baseSwiftGraphQL.operationNames == ["Home"])
        #expect(tokenStore.authToken == nil)
        #expect(tokenStore.refreshToken == nil)
        #expect(tokenStore.clearCount == 1)
    }
}

@Suite("GraphQL client boundary", .tags(.unit))
@MainActor
struct KindredGraphQLClientUnitTests {
    @Test("Bearer mode sends authorization and device fingerprint headers")
    func bearerModeSendsAuthAndDeviceHeaders() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 200, body: #"{"data":{"ok":true}}"#)
        ])
        let tokenStore = APIClientTokenStoreFake(
            authToken: "access-1",
            refreshToken: nil,
            deviceID: "device-1"
        )
        let client = URLSessionKindredGraphQLClient(
            configuration: GraphQLClientConfiguration(
                endpointURL: URL(string: "https://graphql.example.test/graphql")!,
                authMode: .bearer
            ),
            session: session,
            tokenStore: tokenStore
        )

        let response: KindredGraphQLResponse<GraphQLTestPayload> = try await client.send(
            KindredGraphQLRequest(query: "query Health { ok }")
        )

        #expect(response.data?.ok == true)
        let request = try #require(session.requests.first)
        #expect(request.httpMethod == "POST")
        #expect(request.value(forHTTPHeaderField: "Authorization") == "Bearer access-1")
        #expect(request.value(forHTTPHeaderField: "X-Device-Fingerprint") == "device-1")
    }

    @Test("API key mode sends public auth header and encoded variables")
    func apiKeyModeSendsHeaderAndVariables() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 200, body: #"{"data":{"ok":true}}"#)
        ])
        let client = URLSessionKindredGraphQLClient(
            configuration: GraphQLClientConfiguration(
                endpointURL: URL(string: "https://graphql.example.test/graphql")!,
                authMode: .apiKey("public-key")
            ),
            session: session
        )

        let response: KindredGraphQLResponse<GraphQLTestPayload> = try await client.send(
            KindredGraphQLRequest(
                query: "mutation Login($email: String!) { login(input: { email: $email, password: \"pw\" }) { token } }",
                operationName: "Login",
                variables: GraphQLTestVariables(email: "user@example.com")
            )
        )

        #expect(response.data?.ok == true)
        let request = try #require(session.requests.first)
        #expect(request.value(forHTTPHeaderField: "x-api-key") == "public-key")
        #expect(request.value(forHTTPHeaderField: "Authorization") == nil)
        let body = try #require(request.httpBody)
        let captured = try JSONDecoder().decode(CapturedGraphQLRequest.self, from: body)
        #expect(captured.operationName == "Login")
        #expect(captured.variables.email == "user@example.com")
    }

    @Test("GraphQL errors decode extensions code and status")
    func graphQLErrorsDecodeExtensions() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 200, body: #"{"data":{"ok":true},"errors":[{"message":"Forbidden","extensions":{"code":"FORBIDDEN","status":403}},{"message":"Bad input","extensions":{"code":"BAD_USER_INPUT","status":"400"}},{"message":"Expired","extensions":{"errorType":"UnauthorizedException","statusCode":"401"}},{"message":"Unknown"}]}"#)
        ])
        let client = URLSessionKindredGraphQLClient(
            configuration: GraphQLClientConfiguration(
                endpointURL: URL(string: "https://graphql.example.test/graphql")!,
                authMode: .none
            ),
            session: session
        )

        let response: KindredGraphQLResponse<GraphQLTestPayload> = try await client.send(
            KindredGraphQLRequest(query: "query Health { ok }")
        )

        let errors = try #require(response.errors)
        #expect(errors.count == 4)
        #expect(errors[0].message == "Forbidden")
        #expect(errors[0].extensions?.code == "FORBIDDEN")
        #expect(errors[0].extensions?.status == 403)
        #expect(errors[1].extensions?.code == "BAD_USER_INPUT")
        #expect(errors[1].extensions?.status == 400)
        #expect(errors[2].extensions?.code == "UnauthorizedException")
        #expect(errors[2].extensions?.status == 401)
        #expect(errors[3].message == "Unknown")
        #expect(errors[3].extensions == nil)
    }

    @Test("GraphQL errors without data throw client error")
    func graphQLErrorsWithoutDataThrowClientError() async throws {
        let session = APIClientSessionFake(responses: [
            .http(status: 200, body: #"{"errors":[{"message":"Unauthenticated","extensions":{"code":"UNAUTHENTICATED","status":401}}]}"#)
        ])
        let client = URLSessionKindredGraphQLClient(
            configuration: GraphQLClientConfiguration(
                endpointURL: URL(string: "https://graphql.example.test/graphql")!,
                authMode: .none
            ),
            session: session
        )

        do {
            let _: KindredGraphQLResponse<GraphQLTestPayload> = try await client.send(
                KindredGraphQLRequest(query: "query Viewer { viewer { id } }")
            )
            #expect(Bool(false))
        } catch KindredGraphQLClientError.graphQLErrors(let errors) {
            let error = try #require(errors.first)
            #expect(error.message == "Unauthenticated")
            #expect(error.extensions?.code == "UNAUTHENTICATED")
            #expect(error.extensions?.status == 401)
        } catch {
            #expect(Bool(false))
        }
    }
}

private struct GraphQLTestPayload: Decodable {
    let ok: Bool
}

private struct GraphQLTestVariables: Encodable {
    let email: String
}

private struct CapturedGraphQLRequest: Decodable {
    struct Variables: Decodable {
        let email: String
    }

    let operationName: String
    let variables: Variables
}

private final class APIClientSessionFake: URLSessioning, @unchecked Sendable {
    enum Response {
        case http(status: Int, body: String)
    }

    private var responses: [Response]
    private let lock = NSLock()
    private(set) var requests: [URLRequest] = []

    init(responses: [Response]) {
        self.responses = responses
    }

    func data(for request: URLRequest) async throws -> (Data, URLResponse) {
        let response = record(request)

        switch response {
        case .http(let status, let body):
            let httpResponse = HTTPURLResponse(
                url: request.url!,
                statusCode: status,
                httpVersion: nil,
                headerFields: nil
            )!
            return (Data(body.utf8), httpResponse)
        }
    }

    private func record(_ request: URLRequest) -> Response {
        lock.lock()
        requests.append(request)
        let response = responses.removeFirst()
        lock.unlock()
        return response
    }

    var authorizationHeaders: [String?] {
        requests.map { $0.value(forHTTPHeaderField: "Authorization") }
    }

    var refreshPayloads: [CapturedRefreshPayload] {
        requests
            .filter { $0.url?.path == "/auth/refresh" }
            .compactMap(\.httpBody)
            .compactMap { try? JSONDecoder().decode(CapturedRefreshPayload.self, from: $0) }
    }
}

private struct CapturedRefreshPayload: Decodable {
    let refreshToken: String
    let deviceId: String
}

private struct CapturedSwiftGraphQLRefreshInput {
    let refreshToken: String
    let deviceID: String?
}

private final class RefreshingGraphQLClientFake: KindredOperationGraphQLClient, @unchecked Sendable {
    enum Response {
        case success(Any)
        case failure(Error)
    }

    private var responses: [Response]
    private let lock = NSLock()
    private(set) var operationNames: [String] = []

    init(responses: [Response]) {
        self.responses = responses
    }

    func fetch<Query: GraphQLQuery>(
        _ query: Query,
        cachePolicy: CachePolicy.Query.SingleResponse
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat {
        try next(operationName: Query.operationName, as: Query.Data.self)
    }

    func perform<Mutation: GraphQLMutation>(
        _ mutation: Mutation
    ) async throws -> Mutation.Data where Mutation.ResponseFormat == SingleResponseFormat {
        try next(operationName: Mutation.operationName, as: Mutation.Data.self)
    }

    private func next<Output>(operationName: String, as type: Output.Type) throws -> Output {
        lock.lock()
        operationNames.append(operationName)
        let response = responses.removeFirst()
        lock.unlock()

        switch response {
        case .success(let value):
            guard let typed = value as? Output else {
                throw KindredGraphQLClientError.invalidResponse
            }
            return typed
        case .failure(let error):
            throw error
        }
    }
}

private final class APIClientGraphQLSessionRefresherFake: KindredOperationGraphQLClient, @unchecked Sendable {
    private let result: Result<Any, Error>
    private(set) var operationNames: [String] = []
    private(set) var refreshInputs: [CapturedSwiftGraphQLRefreshInput] = []

    init(response: KindredAPI.RefreshSessionMutation.Data) {
        result = .success(response)
    }

    init(error: Error) {
        result = .failure(error)
    }

    func fetch<Query: GraphQLQuery>(
        _ query: Query,
        cachePolicy: CachePolicy.Query.SingleResponse
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat {
        throw KindredGraphQLClientError.invalidResponse
    }

    func perform<Mutation: GraphQLMutation>(
        _ mutation: Mutation
    ) async throws -> Mutation.Data where Mutation.ResponseFormat == SingleResponseFormat {
        operationNames.append(Mutation.operationName)
        if let refreshMutation = mutation as? KindredAPI.RefreshSessionMutation {
            let deviceID: String?
            if case let .some(value) = refreshMutation.input.deviceId {
                deviceID = value
            } else {
                deviceID = nil
            }
            refreshInputs.append(CapturedSwiftGraphQLRefreshInput(
                refreshToken: refreshMutation.input.refreshToken,
                deviceID: deviceID
            ))
        }
        switch result {
        case .success(let data):
            guard let typed = data as? Mutation.Data else {
                throw KindredGraphQLClientError.invalidResponse
            }
            return typed
        case .failure(let error):
            throw error
        }
    }
}

private final class APIClientSessionRefresherFake: SessionRefreshing, @unchecked Sendable {
    struct Call {
        let refreshToken: String
        let deviceID: String
    }

    private let response: AuthResponseDTO?
    private let lock = NSLock()
    private(set) var calls: [Call] = []

    init(response: AuthResponseDTO?) {
        self.response = response
    }

    func refreshSession(refreshToken: String, deviceID: String) async throws -> AuthResponseDTO? {
        record(Call(refreshToken: refreshToken, deviceID: deviceID))
        return response
    }

    private func record(_ call: Call) {
        lock.lock()
        calls.append(call)
        lock.unlock()
    }
}

private func refreshSessionSwiftGraphQLData(
    refreshToken: String? = "fresh-refresh"
) -> KindredAPI.RefreshSessionMutation.Data {
    KindredAPI.RefreshSessionMutation.Data(
        refreshSession: KindredAPI.RefreshSessionMutation.Data.RefreshSession(
            token: "fresh-access",
            expiresAt: nil,
            refreshToken: refreshToken,
            refreshExpiresAt: nil,
            user: apiClientSessionSwiftGraphQLUser()
        )
    )
}

private func homeQuery() -> KindredAPI.HomeQuery {
    KindredAPI.HomeQuery(
        lat: 38.4619922,
        lng: 27.2111511,
        radiusKm: 10,
        limit: 5,
        cursor: .none
    )
}

private func homeSwiftGraphQLData() -> KindredAPI.HomeQuery.Data {
    KindredAPI.HomeQuery.Data(
        home: KindredAPI.HomeQuery.Data.Home(
            meStats: KindredAPI.HomeQuery.Data.Home.MeStats(
                pointsAvailable: 42,
                itemsListed: 2,
                itemsActive: 1,
                itemsCompleted: 1,
                requestsOpen: 3,
                requestsCompleted: 4,
                activeRequests: 5,
                completedDeliveries: 6,
                city: "Izmir"
            ),
            communityStats: KindredAPI.HomeQuery.Data.Home.CommunityStats(
                availableItems: 7,
                activeDonors: 8
            ),
            items: KindredAPI.HomeQuery.Data.Home.Items(items: [], nextCursor: nil)
        )
    )
}

private func apiClientSessionSwiftGraphQLUser() -> KindredAPI.RefreshSessionMutation.Data.RefreshSession.User {
    KindredAPI.RefreshSessionMutation.Data.RefreshSession.User(
        id: "user-1",
        email: "user@example.com",
        displayName: "User One",
        phone: "+15555550100",
        phoneVerified: true,
        emailVerified: false,
        city: "Istanbul",
        birthYear: 1994,
        gender: "unspecified"
    )
}

private final class APIClientTokenStoreFake: AuthTokenStore, @unchecked Sendable {
    private let lock = NSLock()
    private var token: String?
    private var refresh: String?
    let deviceID: String
    private(set) var clearCount = 0

    init(authToken: String?, refreshToken: String?, deviceID: String) {
        token = authToken
        refresh = refreshToken
        self.deviceID = deviceID
    }

    var authToken: String? {
        lock.lock(); defer { lock.unlock() }
        return token
    }

    var refreshToken: String? {
        lock.lock(); defer { lock.unlock() }
        return refresh
    }

    func setTokens(accessToken: String, refreshToken: String?) {
        lock.lock()
        token = accessToken
        refresh = refreshToken
        lock.unlock()
    }

    func clearAuthToken() {
        lock.lock()
        token = nil
        refresh = nil
        clearCount += 1
        lock.unlock()
    }
}
