class Adx < Formula
  desc "Making Android development effortless — zero-config CLI for Gradle and ADB"
  homepage "https://github.com/Shashwat-CODING/adx"
  version "1.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Shashwat-CODING/adx/releases/download/v1.0.0/adx_v1.0.0_darwin_arm64.tar.gz"
    else
      url "https://github.com/Shashwat-CODING/adx/releases/download/v1.0.0/adx_v1.0.0_darwin_amd64.tar.gz"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/Shashwat-CODING/adx/releases/download/v1.0.0/adx_v1.0.0_linux_arm64.tar.gz"
    else
      url "https://github.com/Shashwat-CODING/adx/releases/download/v1.0.0/adx_v1.0.0_linux_amd64.tar.gz"
    end
  end

  def install
    bin.install "adx"
  end

  test do
    system "#{bin}/adx", "--help"
  end
end
