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

  # Platform-specific hashes for v0.8.3
  hashes = {
    "x86_64-linux" = "0b76d17ff1c22cafde664fde95337db804b245fc36c0c151bad2bf65d8613c61";
    "aarch64-linux" = "f0b9b312913828f5c0b16839e8c0fe3c62a1720dd5f70712d7140a7ff759c67b";
    "x86_64-darwin" = "68799bcaa3b5d04b1c1d4ad39c22b37a7cf28ebeb0edde20495587dec2b4442d";
    "aarch64-darwin" = "6061b9e9cc4288c77190fe26248877ac8079fe9ae03d8b92c4ace25302d34ec8";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.8.3";

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
