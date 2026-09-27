# This formula lives in the project's main repository, so there is no separate
# homebrew-tap repository. Install it with:
#
#   brew tap serhii-chechun/pg-query-validate https://github.com/serhii-chechun/pg-query-validate
#   brew install pgqv
#
# It installs the prebuilt release binary rather than building from source:
# pgqv is built with cgo, and a Homebrew build has no network access for the Go
# module downloads (and would compile the bundled PostgreSQL C sources).
#
# The sha256 values must match the archives attached to the release. After
# publishing a release, refresh them with scripts/update-formula.sh.
class Pgqv < Formula
  desc "Standalone query validator and linter for PostgreSQL"
  homepage "https://github.com/serhii-chechun/pg-query-validate"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.0/pgqv_1.0.0_darwin_arm64.tar.gz"
      sha256 "6d06c7da76b7d49c7e143609d8907658392a57e55aaa3839641ecbcbfaade3da"
    end

    on_intel do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.0/pgqv_1.0.0_darwin_amd64.tar.gz"
      sha256 "4c79b9825068743e816a89c131929167ff3ae58d1189fe37d55d4aaf33b2b6a7"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.0/pgqv_1.0.0_linux_arm64.tar.gz"
      sha256 "85b26f2529a722a2694b5c8694ffbf15936d13e4f94c54d87af48b1bcfe0f767"
    end

    on_intel do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.0/pgqv_1.0.0_linux_amd64.tar.gz"
      sha256 "190fa133488afad0a634b08e19f5287c51d8f68cd4b09516c313a18ce51d7ee7"
    end
  end

  def install
    bin.install "pgqv"
  end

  test do
    (testpath/"valid.sql").write "create table t (id uuid primary key);"
    system bin/"pgqv", testpath/"valid.sql"
  end
end
