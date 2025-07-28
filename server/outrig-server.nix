{
  stdenv,
  lib,
  fetchurl,
}:
let
  system = lib.splitString "-" stdenv.hostPlatform.system;
  arch = lib.head system;
  platform =
    let
      platform = lib.head (lib.tail system);
    in
    if platform == "linux" then
      "Linux"
    else if platform == "darwin" then
      "Darwin"
    else
      platform;

  # Platform-specific hashes for v0.9.0
  hashes = {
    "x86_64-linux" = "f7194003b5b627a0b23a76545dc9301491ef15c4357baec80332a2f5c77ec7e1";
    "aarch64-linux" = "1fa03e4d66c0eba6481bc6e81ec8c35c872a5164fa23338d7c95c954d51df48d";
    "x86_64-darwin" = "08d1bc187053dc982c7df19f0997ff203bc4e2b297d7aa9fb3339e3493f76fdd";
    "aarch64-darwin" = "474aa387788c3194b2a261ade0a43177f60600ce17e449a5c3ea933d2761d7fd";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.0";

  src = fetchurl {
    url = "https://github.com/outrigdev/outrig/releases/download/v${version}/${pname}_${version}_${platform}_${arch}.tar.gz";
    sha256 = hashes.${stdenv.hostPlatform.system};
  };

  sourceRoot = "${pname}_${version}_${platform}_${arch}";

  unpackPhase = ''
    tar -xzf $src
  '';

  dontBuild = true;

  installPhase = ''
    mkdir -p $out/bin
    cp ${pname} $out/bin/${pname}
    chmod +x $out/bin/${pname}
  '';

  meta = {
    description = "Dev-time observability tool for Go programs. Search logs, monitor goroutines, and track variables";
    homepage = "https://outrig.run/";
    license = lib.licenses.asl20;
    maintainers = with lib.maintainers; [ sawka ];
    mainProgram = "outrig";
  };
})
