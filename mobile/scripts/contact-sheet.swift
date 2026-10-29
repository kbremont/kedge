// Lays screenshots out as a contact sheet, grouped under headings. Called by
// shoot.mjs with a JSON spec and the PNG to write:
//
//   swift scripts/contact-sheet.swift <spec.json> <out.png>
import AppKit
import CoreText

struct Item: Decodable { let label: String; let path: String }
struct Section: Decodable { let title: String; let items: [Item] }
struct Spec: Decodable {
  let title: String
  let sections: [Section]
  let columns: Int
  let thumbWidth: Int
  let dark: Bool
  let font: String
}

let args = CommandLine.arguments
let spec = try JSONDecoder().decode(Spec.self, from: Data(contentsOf: URL(fileURLWithPath: args[1])))
CTFontManagerRegisterFontsForURL(URL(fileURLWithPath: spec.font) as CFURL, .process, nil)

// The Career Line inks: chipboard and navy, swapped in dark.
func rgb(_ hex: Int) -> NSColor {
  NSColor(
    srgbRed: CGFloat((hex >> 16) & 0xFF) / 255, green: CGFloat((hex >> 8) & 0xFF) / 255,
    blue: CGFloat(hex & 0xFF) / 255, alpha: 1)
}
let ground = spec.dark ? rgb(0x121729) : rgb(0xC9C3B7)
let ink = spec.dark ? rgb(0xE4DED2) : rgb(0x1B2240)
let ink2 = spec.dark ? rgb(0xA9AEC2) : rgb(0x454B63)

let display = NSFont(name: "SofiaSansExtraCondensed-Black", size: 40) ?? .boldSystemFont(ofSize: 40)
let heading = NSFont(name: "SofiaSansExtraCondensed-Black", size: 30) ?? .boldSystemFont(ofSize: 30)
let label = NSFont.systemFont(ofSize: 17, weight: .semibold)

let pad = 56.0, gap = 28.0, labelH = 36.0, headH = 70.0, titleH = 84.0
let cols = spec.columns
let tw = Double(spec.thumbWidth)
guard let first = spec.sections.first?.items.first, let probe = NSImage(contentsOfFile: first.path),
  let rep = probe.representations.first
else { fatalError("no screenshots") }
let th = (tw * Double(rep.pixelsHigh) / Double(rep.pixelsWide)).rounded()

let width = pad * 2 + Double(cols) * tw + Double(cols - 1) * gap
var height = pad + titleH + pad
for s in spec.sections {
  let rows = (s.items.count + cols - 1) / cols
  height += headH + Double(rows) * (th + labelH + gap)
}

let ctx = CGContext(
  data: nil, width: Int(width), height: Int(height), bitsPerComponent: 8, bytesPerRow: 0,
  space: CGColorSpace(name: CGColorSpace.sRGB)!, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue)!
ctx.translateBy(x: 0, y: height)
ctx.scaleBy(x: 1, y: -1)
NSGraphicsContext.current = NSGraphicsContext(cgContext: ctx, flipped: true)

ground.setFill()
NSRect(x: 0, y: 0, width: width, height: height).fill()

func text(_ s: String, _ font: NSFont, _ color: NSColor, _ x: Double, _ y: Double) {
  NSAttributedString(string: s, attributes: [.font: font, .foregroundColor: color]).draw(at: NSPoint(x: x, y: y))
}

var y = pad
text(spec.title.uppercased(), display, ink, pad, y)
y += titleH
for s in spec.sections {
  text(s.title.uppercased(), heading, ink, pad, y + 14)
  ink.setFill()
  NSRect(x: pad, y: y + 56, width: width - pad * 2, height: 3).fill()
  y += headH
  for (i, item) in s.items.enumerated() {
    let col = i % cols
    if i > 0 && col == 0 { y += th + labelH + gap }
    let x = pad + Double(col) * (tw + gap)
    let frame = NSRect(x: x, y: y, width: tw, height: th)
    if let img = NSImage(contentsOfFile: item.path) {
      NSGraphicsContext.saveGraphicsState()
      NSBezierPath(roundedRect: frame, xRadius: 26, yRadius: 26).addClip()
      img.draw(in: frame, from: .zero, operation: .sourceOver, fraction: 1, respectFlipped: true, hints: nil)
      NSGraphicsContext.restoreGraphicsState()
    }
    text(item.label, label, ink2, x + 2, y + th + 8)
  }
  y += th + labelH + gap
}

let png = NSBitmapImageRep(cgImage: ctx.makeImage()!).representation(using: .png, properties: [:])!
try png.write(to: URL(fileURLWithPath: args[2]))
