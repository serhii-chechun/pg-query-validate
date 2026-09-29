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
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.1/pgqv_1.0.1_darwin_arm64.tar.gz"
      sha256 "c9410a8f2c40dce9290933742a48df29f181601991bf002b94452c70af3caea2"
    end

    on_intel do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.1/pgqv_1.0.1_darwin_amd64.tar.gz"
      sha256 "df8ce501a8c3de1726be3487085e6daf51816aa26cb94aba5bed20e3390ebd46"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.1/pgqv_1.0.1_linux_arm64.tar.gz"
      sha256 "151f545eba7a9375dff633fa1874b5b5a6eb7d18f0b0ce7c0657765756c45aef"
    end

    on_intel do
      url "https://github.com/serhii-chechun/pg-query-validate/releases/download/v1.0.1/pgqv_1.0.1_linux_amd64.tar.gz"
      sha256 "fdf217f7d098297ce797013c3f250ad9da42c3e2cb4bcdf7f8630e959fc12169"
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
