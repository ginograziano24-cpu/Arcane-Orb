# Diversified hero source

The seven `characters/*.blend` files are the canonical diversified hero models integrated in this development copy. Original supplied Downloads files are preserved.

Each model retains its class equipment and mechanics. Body and skeleton proportions were transformed together. Receiving hands, elbows, wrists and three phalanges per visible finger have weighted controls. Held equipment follows its carrying hand. The small catch action has no scale animation. The Guardian keeps her closed visor; the six visible faces have `determined`, `hurt` and `smile` component morphs.

These remain stylized development models. Thumbs are hand-bound; final anatomy, detailed faces, lip sync, hair and cloth simulation need further refinement. Mirabel is not part of this hero revision.

Run `blender -b --python-exit-code 1 --python export.py` from this directory to export the available hero GLBs into `models/`. The exporter excludes hidden Boolean cutters, studio objects and volume-only helper meshes, consolidates render geometry and retains skin and facial morphs. It leaves the source Blender files untouched.

The integrated game assets are `assets/portraits/<id>.jpg`, matching grayscale alpha PNGs and normalized effect-anchor JSONs, plus four `assets/portraits/hud/<id>_<pose>.png` files per hero. Full portraits are 1200 × 1600; HUD files are 536 × 680. Preserve these dimensions and remap anchors with any crop change. `portrait_assets_test.go` checks this contract.

The workspace's `work/integrate-diverse-heroes.cjs` packs full portraits and anchors using the same safe-area transform. `work/refine-hero-expressions.py` produces HUD expressions. `work/refine-hero-articulation.py` uses original model landmarks to weight the joints. Rerunning the original diversity generator rebuilds from pristine pre-diversity files; apply articulation and expression passes again afterward.

For offline review, build the game with `-tags heroreview`. `-review-hud` draws all 28 HUD poses through the real renderer; `-review-chambers` captures all three act maps; `-review-navigation` also verifies reachable focus and chamber entry. These developer builds disable persistence. The default release build does not run capture fixtures.
