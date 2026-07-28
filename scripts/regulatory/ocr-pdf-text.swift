#!/usr/bin/env swift

import AppKit
import Foundation
import PDFKit
import Vision

enum OCRError: Error, CustomStringConvertible {
    case usage
    case unreadablePDF(String)
    case renderFailed(Int)

    var description: String {
        switch self {
        case .usage:
            return "usage: ocr-pdf-text INPUT.pdf OUTPUT.txt CHECKPOINT_DIRECTORY [PAGE_NUMBERS]"
        case let .unreadablePDF(path):
            return "unable to open PDF: \(path)"
        case let .renderFailed(page):
            return "unable to render PDF page \(page)"
        }
    }
}

struct OCRSummary: Codable {
    let engine: String
    let pageCount: Int
    let requestedPageCount: Int
    let requestedPages: [Int]
    let pagesWithOCRText: Int
    let characterCount: Int
    let checkpointDirectory: String
}

func requestedPages(pageCount: Int) throws -> [Int] {
    guard CommandLine.arguments.count == 5 else {
        return Array(1...pageCount)
    }
    let values = CommandLine.arguments[4].split(separator: ",")
    let pages = try values.map { value -> Int in
        guard let page = Int(value), page >= 1, page <= pageCount else {
            throw OCRError.usage
        }
        return page
    }
    return Array(Set(pages)).sorted()
}

func render(page: PDFPage, pageNumber: Int) throws -> CGImage {
    let bounds = page.bounds(for: .mediaBox)
    let targetWidth: CGFloat = 1800
    let targetHeight = targetWidth * max(bounds.height, 1) / max(bounds.width, 1)
    let thumbnail = page.thumbnail(
        of: NSSize(width: targetWidth, height: targetHeight),
        for: .mediaBox
    )
    var proposedRect = NSRect(
        origin: .zero,
        size: NSSize(width: targetWidth, height: targetHeight)
    )
    guard let image = thumbnail.cgImage(
        forProposedRect: &proposedRect,
        context: nil,
        hints: nil
    ) else {
        throw OCRError.renderFailed(pageNumber)
    }
    return image
}

func recognize(image: CGImage) throws -> String {
    let request = VNRecognizeTextRequest()
    request.recognitionLevel = .accurate
    request.usesLanguageCorrection = true
    request.recognitionLanguages = ["en-US"]
    request.minimumTextHeight = 0.006

    let handler = VNImageRequestHandler(cgImage: image, orientation: .up)
    try handler.perform([request])
    let observations = (request.results ?? []).sorted { left, right in
        let verticalDifference = abs(left.boundingBox.midY - right.boundingBox.midY)
        if verticalDifference > 0.012 {
            return left.boundingBox.midY > right.boundingBox.midY
        }
        return left.boundingBox.minX < right.boundingBox.minX
    }
    return observations.compactMap { observation in
        observation.topCandidates(1).first?.string
    }.joined(separator: "\n")
}

func run() throws {
    guard CommandLine.arguments.count == 4 || CommandLine.arguments.count == 5 else {
        throw OCRError.usage
    }

    let inputURL = URL(fileURLWithPath: CommandLine.arguments[1]).standardizedFileURL
    let outputURL = URL(fileURLWithPath: CommandLine.arguments[2]).standardizedFileURL
    let checkpointURL = URL(fileURLWithPath: CommandLine.arguments[3]).standardizedFileURL
    guard let document = PDFDocument(url: inputURL) else {
        throw OCRError.unreadablePDF(inputURL.path)
    }
    let selectedPages = try requestedPages(pageCount: document.pageCount)

    let fileManager = FileManager.default
    try fileManager.createDirectory(
        at: outputURL.deletingLastPathComponent(),
        withIntermediateDirectories: true
    )
    try fileManager.createDirectory(at: checkpointURL, withIntermediateDirectories: true)

    var pagesWithText = 0
    var characterCount = 0
    var combined = ""

    for (selectionIndex, pageNumber) in selectedPages.enumerated() {
        try autoreleasepool {
            let pageIndex = pageNumber - 1
            let checkpoint = checkpointURL.appendingPathComponent(
                String(format: "%05d.txt", pageNumber)
            )
            let text: String
            if fileManager.fileExists(atPath: checkpoint.path) {
                text = try String(contentsOf: checkpoint, encoding: .utf8)
            } else {
                guard let page = document.page(at: pageIndex) else {
                    throw OCRError.renderFailed(pageNumber)
                }
                text = try recognize(image: render(page: page, pageNumber: pageNumber))
                try text.write(to: checkpoint, atomically: true, encoding: .utf8)
            }

            if !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                pagesWithText += 1
            }
            characterCount += text.count
            combined += "\n\n--- PAGE \(pageNumber) [OCR] ---\n\n\(text)"
            let completedCount = selectionIndex + 1
            if completedCount == selectedPages.count || completedCount % 10 == 0 {
                FileHandle.standardError.write(
                    Data(
                        "[OCR] \(inputURL.lastPathComponent): \(completedCount)/\(selectedPages.count) selected pages (document page \(pageNumber)/\(document.pageCount))\n".utf8
                    )
                )
            }
        }
    }

    try combined.write(to: outputURL, atomically: true, encoding: .utf8)
    let summary = OCRSummary(
        engine: "apple-vision-ocr",
        pageCount: document.pageCount,
        requestedPageCount: selectedPages.count,
        requestedPages: selectedPages,
        pagesWithOCRText: pagesWithText,
        characterCount: characterCount,
        checkpointDirectory: checkpointURL.path
    )
    let encoded = try JSONEncoder().encode(summary)
    FileHandle.standardOutput.write(encoded)
    FileHandle.standardOutput.write(Data("\n".utf8))
}

do {
    try run()
} catch {
    let nsError = error as NSError
    FileHandle.standardError.write(
        Data(
            "ocr-pdf-text: \(error) [domain=\(nsError.domain) code=\(nsError.code) userInfo=\(nsError.userInfo)]\n".utf8
        )
    )
    exit(1)
}
