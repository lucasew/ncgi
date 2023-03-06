{ buildGoModule, lib }:
buildGoModule {
  name = "ncgi";

  src = ./.;

  vendorSha256 = "sha256-xfy8KB8VPz5miZkuyfWSxN0rcJKpJo1xkWFPYhuBux0=";
}
