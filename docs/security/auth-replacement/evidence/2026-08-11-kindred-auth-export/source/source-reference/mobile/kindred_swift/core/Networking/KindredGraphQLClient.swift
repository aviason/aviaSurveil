import Foundation

struct GraphQLAuthMode: Equatable, Sendable {
    enum Strategy: Equatable, Sendable {
        case none
        case bearer
        case apiKey(headerName: String, value: String)
    }

    let strategy: Strategy

    static let none = GraphQLAuthMode(strategy: .none)
    static let bearer = GraphQLAuthMode(strategy: .bearer)

    static func apiKey(_ value: String, headerName: String = "x-api-key") -> GraphQLAuthMode {
        GraphQLAuthMode(strategy: .apiKey(headerName: headerName, value: value))
    }
}

struct GraphQLClientConfiguration: Equatable, Sendable {
    let endpointURL: URL
    let authMode: GraphQLAuthMode
    let deviceFingerprint: String?
    let deviceFingerprintHeaderName: String
    let additionalHeaders: [String: String]

    init(
        endpointURL: URL,
        authMode: GraphQLAuthMode = .bearer,
        deviceFingerprint: String? = nil,
        deviceFingerprintHeaderName: String = "X-Device-Fingerprint",
        additionalHeaders: [String: String] = [:]
    ) {
        self.endpointURL = endpointURL
        self.authMode = authMode
        self.deviceFingerprint = deviceFingerprint
        self.deviceFingerprintHeaderName = deviceFingerprintHeaderName
        self.additionalHeaders = additionalHeaders
    }
}

struct EmptyGraphQLVariables: Encodable, Sendable {
    init() {}
}

struct KindredGraphQLRequest<Variables: Encodable>: Encodable {
    let query: String
    let operationName: String?
    let variables: Variables?

    init(
        query: String,
        operationName: String? = nil,
        variables: Variables? = nil
    ) {
        self.query = query
        self.operationName = operationName
        self.variables = variables
    }
}

extension KindredGraphQLRequest: Sendable where Variables: Sendable {}

extension KindredGraphQLRequest where Variables == EmptyGraphQLVariables {
    init(query: String, operationName: String? = nil) {
        self.init(query: query, operationName: operationName, variables: nil)
    }
}

struct KindredGraphQLResponse<Payload: Decodable>: Decodable {
    let data: Payload?
    let errors: [KindredGraphQLError]?

    var hasErrors: Bool {
        !(errors?.isEmpty ?? true)
    }
}

extension KindredGraphQLResponse: Sendable where Payload: Sendable {}

struct KindredGraphQLError: Decodable, Equatable, LocalizedError, Sendable {
    struct Extensions: Decodable, Equatable, Sendable {
        let code: String?
        let status: Int?

        private enum CodingKeys: String, CodingKey {
            case code
            case errorType
            case status
            case statusCode
        }

        init(from decoder: Decoder) throws {
            let container = try decoder.container(keyedBy: CodingKeys.self)
            code = (try? container.decode(String.self, forKey: .code))
                ?? (try? container.decode(String.self, forKey: .errorType))
            if let intStatus = try? container.decode(Int.self, forKey: .status) {
                status = intStatus
            } else if let stringStatus = try? container.decode(String.self, forKey: .status) {
                status = Int(stringStatus)
            } else if let intStatusCode = try? container.decode(Int.self, forKey: .statusCode) {
                status = intStatusCode
            } else if let stringStatusCode = try? container.decode(String.self, forKey: .statusCode) {
                status = Int(stringStatusCode)
            } else {
                status = nil
            }
        }

        init(code: String?, status: Int?) {
            self.code = code
            self.status = status
        }
    }

    let message: String
    let extensions: Extensions?

    init(message: String, code: String? = nil, status: Int? = nil) {
        self.message = message
        if code != nil || status != nil {
            extensions = Extensions(code: code, status: status)
        } else {
            extensions = nil
        }
    }

    var errorDescription: String? {
        message
    }
}

enum KindredGraphQLClientError: Equatable, LocalizedError, Sendable {
    case invalidResponse
    case unsuccessfulStatusCode(Int)
    case graphQLErrors([KindredGraphQLError])

    var errorDescription: String? {
        switch self {
        case .invalidResponse:
            return "The GraphQL response is invalid."
        case .unsuccessfulStatusCode(let statusCode):
            return "The GraphQL endpoint returned an unsuccessful status code: \(statusCode)."
        case .graphQLErrors(let errors):
            return errors.first?.message ?? "The GraphQL endpoint returned errors."
        }
    }
}

protocol KindredGraphQLClient {
    func send<Response: Decodable, Variables: Encodable>(
        _ request: KindredGraphQLRequest<Variables>
    ) async throws -> KindredGraphQLResponse<Response>
}

struct URLSessionKindredGraphQLClient: KindredGraphQLClient {
    let configuration: GraphQLClientConfiguration
    let session: URLSessioning
    let encoder: JSONEncoder
    let decoder: JSONDecoder
    let tokenStore: AuthTokenStore?

    init(
        configuration: GraphQLClientConfiguration,
        session: URLSessioning = URLSession.shared,
        encoder: JSONEncoder = APICoding.encoder(),
        decoder: JSONDecoder = APICoding.decoder(),
        tokenStore: AuthTokenStore? = nil
    ) {
        self.configuration = configuration
        self.session = session
        self.encoder = encoder
        self.decoder = decoder
        self.tokenStore = tokenStore
    }

    func send<Response: Decodable, Variables: Encodable>(
        _ graphqlRequest: KindredGraphQLRequest<Variables>
    ) async throws -> KindredGraphQLResponse<Response> {
        var request = URLRequest(url: configuration.endpointURL)
        request.httpMethod = HTTPMethod.post.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        for (key, value) in configuration.additionalHeaders {
            request.setValue(value, forHTTPHeaderField: key)
        }
        applyAuthHeader(to: &request)
        applyDeviceFingerprintHeader(to: &request)

        request.httpBody = try encoder.encode(graphqlRequest)

        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw KindredGraphQLClientError.invalidResponse
        }
        guard (200...299).contains(httpResponse.statusCode) else {
            throw KindredGraphQLClientError.unsuccessfulStatusCode(httpResponse.statusCode)
        }
        guard !data.isEmpty else {
            throw KindredGraphQLClientError.invalidResponse
        }

        let graphQLResponse = try decoder.decode(KindredGraphQLResponse<Response>.self, from: data)
        if graphQLResponse.data == nil, let errors = graphQLResponse.errors, !errors.isEmpty {
            throw KindredGraphQLClientError.graphQLErrors(errors)
        }
        return graphQLResponse
    }

    private func applyAuthHeader(to request: inout URLRequest) {
        switch configuration.authMode.strategy {
        case .none:
            break
        case .bearer:
            guard let token = tokenStore?.authToken, !token.isEmpty else { return }
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        case .apiKey(let headerName, let value):
            guard !headerName.isEmpty, !value.isEmpty else { return }
            request.setValue(value, forHTTPHeaderField: headerName)
        }
    }

    private func applyDeviceFingerprintHeader(to request: inout URLRequest) {
        let fingerprint = configuration.deviceFingerprint ?? tokenStore?.deviceID
        guard let fingerprint, !fingerprint.isEmpty else { return }
        request.setValue(fingerprint, forHTTPHeaderField: configuration.deviceFingerprintHeaderName)
    }
}
