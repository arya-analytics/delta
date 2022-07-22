<script lang="ts">
  import Full from "./LogoFull.svelte";
  import Partial from "./LogoPartial.svelte";
  import Icon from "./LogoIconOnly.svelte";
  import "./dark.scss";
  import { getContext } from "svelte";

  export let scale = 40;
  export let variant: "full" | "partial" | "icon" = "full";
  let variantMap = {
    full: Full,
    partial: Partial,
    icon: Icon
  };

  let themeValue;
  getContext('theme').theme.subscribe(theme => {themeValue = theme;});
</script>

<svelte:component
  this={variantMap[variant]}
  width={themeValue.Dimensions.Grid * scale}
  fill="white"
/>