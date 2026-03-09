# TokMan Homebrew Formula
# Install with: brew install GrayCodeAI/tokman/tokman
# Or: brew tap GrayCodeAI/tokman && brew install tokman

class Tokman < Formula
  desc "Token-aware CLI proxy that reduces LLM token consumption by 60-90%"
  homepage "https://github.com/GrayCodeAI/tokman"
  version "0.1.0"
  license "MIT"
  head "https://github.com/GrayCodeAI/tokman.git", branch: "main"

  livecheck do
    url :stable
    strategy :github_latest
  end

  on_macos do
    on_intel do
      url "https://github.com/GrayCodeAI/tokman/releases/download/v#{version}/tokman-x86_64-apple-darwin.tar.gz"
      sha256 "PLACEHOLDER_X86_64_DARWIN_SHA256"
    end
    on_arm do
      url "https://github.com/GrayCodeAI/tokman/releases/download/v#{version}/tokman-aarch64-apple-darwin.tar.gz"
      sha256 "PLACEHOLDER_AARCH64_DARWIN_SHA256"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/GrayCodeAI/tokman/releases/download/v#{version}/tokman-x86_64-unknown-linux-musl.tar.gz"
      sha256 "PLACEHOLDER_X86_64_LINUX_SHA256"
    end
    on_arm do
      url "https://github.com/GrayCodeAI/tokman/releases/download/v#{version}/tokman-aarch64-unknown-linux-gnu.tar.gz"
      sha256 "PLACEHOLDER_AARCH64_LINUX_SHA256"
    end
  end

  def install
    bin.install "tokman"
    
    # Install shell completions
    generate_completions_from_executable(bin/"tokman", "completion", shells: [:bash, :zsh, :fish])
    
    # Install man page if exists
    man1.install "tokman.1" if File.exist?("tokman.1")
  end

  def caveats
    <<~EOS
      TokMan installed! To set up shell integration:
      
        tokman init
      
      Then restart your shell or run:
        source ~/.bashrc   # or ~/.zshrc
      
      For Claude Code integration, add to ~/.claude/settings.json:
        "hooks": {
          "PreToolUse": [
            {
              "matcher": "Bash",
              "hooks": ["~/.claude/hooks/tokman-rewrite.sh"]
            }
          ]
        }
      
      View your token savings:
        tokman gain
      
      Documentation: https://github.com/GrayCodeAI/tokman#readme
    EOS
  end

  test do
    # Test version command
    assert_match "TokMan", shell_output("#{bin}/tokman --version")
    
    # Test help command
    assert_match "Token-aware CLI proxy", shell_output("#{bin}/tokman --help")
    
    # Test init command (dry run)
    output = shell_output("#{bin}/tokman init --help")
    assert_match "Initialize", output
  end
end
