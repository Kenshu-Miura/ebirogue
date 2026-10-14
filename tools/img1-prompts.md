# img-1 原画プロンプト

生成: built-in image_gen。各原画ごとに1回、透明背景指定。
被写体の一覧は `img1-manifest.txt`（名前・種類・Subject）の44行。

共通プロンプト（Subject と種類を各行から展開）:

```
Use case: stylized-concept. ONE isolated [kind] sprite original for a
30x30 Japanese ocean-themed roguelike: [Subject]. Saturated 16-bit
pixel art with bold chunky readable clusters, 3-4 shading levels,
upper-left lighting, near-black dark brown outline.
Simplify for readability at 30x30. Center entire object with transparent
empty margin all around. Genuine transparent background. No ground,
no outside drop shadow, no labels, no text, no watermark, no extra
objects. One sprite only, never a sheet.
```

種類別に以下を追加:

- weapon: Handle at lower left and tip at upper right, diagonal 45 degrees.
  Blade, guard and grip different colors.
- enemy: Entire creature full body in three-quarter view, expressive
  visible eyes, distinctive body silhouette.
- shield: Frontal symmetric view, distinct bold rim and large simple
  central emblem.

最初の hibana_umiushi は同等の単体・透明・16bit・左上光・3〜4段階・
太い輪郭・目と全身の指示で生成。原画自体は `img1-originals/` に保持する。

原画の生成は非決定的だが、保存原画からの最終加工と上位種は決定的。

## uni_breaker の描き直し

```
Use case: stylized-concept. ONE isolated weapon sprite for 30x30
Japanese pixel roguelike. A scimitar with an extremely broad CRESCENT
MOON blade: deeply concave inner edge and broad convex outer edge,
the blade spans almost half the sprite width, dramatically hooked tip
to right upper corner. Strong unmistakable crescent silhouette,
not a narrow katana. Handle lower left, blade upper right diagonal45
degrees. SILVER blade with cyan shadow, purple handle and gold guard.
Bold chunky 16-bit pixel art, only 3-4 shading levels, light upper left,
dark brown near-black outline. Center entire subject, genuine transparent
background, wide empty margin, no text, no floor, no drop shadow.
Simple readable shape without ornate patterns.
```

## double_strike_sword の描き直し

```
Use case: stylized-concept. ONE isolated weapon sprite original for
30x30 Japanese roguelike. Double-strike katana: one purple grip at
lower left and gold crossguard, TWO VERY CLEARLY SEPARATED PARALLEL
curved silver blade silhouettes toward upper right, with
3-pixel-equivalent wide empty transparent gap, second blade is bright
lavender afterimage. Make double silhouette unmistakable at tiny
30x30 size. Diagonal45 degrees. Simple chunky saturated 16-bit pixel
art, only3-4 shading levels, upper left light, near-black dark brown
outline. Center whole subject within square with empty transparent
margin. Genuine transparent background. No text, no floor, no outside
shadow, no ornamental patterns. Only one weapon, not a sheet.
```

## dragon_scale_shield の描き直し

```
Use case: stylized-concept. ONE isolated frontal dragon-scale shield
sprite for 30x30 Japanese pixel roguelike. Pointed shield silhouette
with thick gold border. Entire central red face made from ONLY SEVEN
HUGE overlapping red dragon scales arranged in 3 staggered rows,
each scale with a simple bright upper-left facet and dark bottom edge.
Large clearly visible scale shapes occupying the entire face,
NO dragon picture, NO central emblem, NO tiny scale texture,
NO detailed ornament. Chunky saturated16-bit pixel art, 3-4 tones
per material, upper-left light, near-black dark brown outer contour.
Entire shield centered with empty transparent margin all around,
genuine transparent background. No text, no letters, no floor,
no outside shadow. One sprite only.
```
