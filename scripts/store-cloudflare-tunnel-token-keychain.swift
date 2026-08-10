#!/usr/bin/env swift
import Foundation
import Security

private struct SafeFailure: Error {
  let message: String
}

private func requireFullMatch(_ value: String, pattern: String, label: String) throws {
  let expression = try NSRegularExpression(pattern: pattern)
  let range = NSRange(value.startIndex..<value.endIndex, in: value)
  guard expression.firstMatch(in: value, range: range)?.range == range else {
    throw SafeFailure(message: "\(label) contains unsupported characters")
  }
}

private func validateConnectorToken(_ tokenData: Data) throws {
  guard !tokenData.isEmpty, tokenData.count <= 8192,
        let token = String(data: tokenData, encoding: .ascii),
        token.hasPrefix("eyJ") else {
    throw SafeFailure(message: "connector token is malformed or truncated")
  }
  try requireFullMatch(
    token,
    pattern: #"[A-Za-z0-9+/_-]+={0,2}"#,
    label: "connector token",
  )

  let standardBase64 = token
    .replacingOccurrences(of: "-", with: "+")
    .replacingOccurrences(of: "_", with: "/")
  let unpadded = standardBase64.replacingOccurrences(
    of: #"=+$"#,
    with: "",
    options: .regularExpression,
  )
  guard unpadded.count % 4 != 1 else {
    throw SafeFailure(message: "connector token is malformed or truncated")
  }
  let suppliedPadding = standardBase64.count - unpadded.count
  let expectedPadding = (4 - (unpadded.count % 4)) % 4
  guard suppliedPadding == 0 || suppliedPadding == expectedPadding,
        let decoded = Data(
          base64Encoded: unpadded + String(repeating: "=", count: expectedPadding)
        ),
        decoded.base64EncodedString().replacingOccurrences(
          of: #"=+$"#,
          with: "",
          options: .regularExpression,
        ) == unpadded,
        let object = try? JSONSerialization.jsonObject(with: decoded),
        let connector = object as? [String: Any] else {
    throw SafeFailure(message: "connector token is malformed or truncated")
  }
  for field in ["a", "t", "s"] {
    guard let value = connector[field] as? String, !value.isEmpty else {
      throw SafeFailure(message: "connector token is malformed or truncated")
    }
  }
}

private func keychainMessage(_ status: OSStatus) -> String {
  if let message = SecCopyErrorMessageString(status, nil) as String? {
    return message
  }
  return "Security framework status \(status)"
}

private func run() throws {
  guard CommandLine.arguments.count == 4 else {
    throw SafeFailure(
      message: "usage: store-cloudflare-tunnel-token-keychain.swift <service> <account> <hostname>"
    )
  }
  let service = CommandLine.arguments[1]
  let account = CommandLine.arguments[2]
  let hostname = CommandLine.arguments[3]
  try requireFullMatch(service, pattern: #"[A-Za-z0-9._:-]+"#, label: "Keychain service")
  try requireFullMatch(account, pattern: #"[A-Za-z0-9._:@-]+"#, label: "Keychain account")
  try requireFullMatch(
    hostname,
    pattern: #"[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+"#,
    label: "hostname",
  )

  var tokenData = FileHandle.standardInput.readDataToEndOfFile()
  defer {
    if !tokenData.isEmpty {
      tokenData.resetBytes(in: 0..<tokenData.count)
    }
  }
  try validateConnectorToken(tokenData)

  let itemLabel = "AviaSurveil360 Tunnel: \(hostname)"
  var trustedApplication: SecTrustedApplication?
  var status = SecTrustedApplicationCreateFromPath(
    "/usr/bin/security",
    &trustedApplication,
  )
  guard status == errSecSuccess, let trustedApplication else {
    throw SafeFailure(message: "could not authorize the macOS Keychain reader")
  }

  var access: SecAccess?
  status = SecAccessCreate(
    itemLabel as CFString,
    [trustedApplication] as CFArray,
    &access,
  )
  guard status == errSecSuccess, let access else {
    throw SafeFailure(message: "could not create restricted macOS Keychain access")
  }

  let attributes: [String: Any] = [
    kSecClass as String: kSecClassGenericPassword,
    kSecAttrAccount as String: account,
    kSecAttrService as String: service,
    kSecAttrDescription as String: "Cloudflare Tunnel connector token",
    kSecAttrLabel as String: itemLabel,
    kSecAttrComment as String: "Remotely-managed connector token for \(hostname)",
    kSecAttrAccess as String: access,
    kSecValueData as String: tokenData,
  ]
  status = SecItemAdd(attributes as CFDictionary, nil)
  guard status == errSecSuccess else {
    throw SafeFailure(message: "macOS Keychain write failed: \(keychainMessage(status))")
  }
}

do {
  try run()
} catch let failure as SafeFailure {
  FileHandle.standardError.write(
    Data("cloudflare-tunnel-keychain: \(failure.message)\n".utf8)
  )
  exit(1)
} catch {
  FileHandle.standardError.write(
    Data("cloudflare-tunnel-keychain: unexpected Keychain helper failure\n".utf8)
  )
  exit(1)
}
