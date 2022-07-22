<script>
  import { Button, Input, Space } from "@arya-analytics/pluto";
  import Logo from "../lib/Logo/Logo.svelte";
</script>
<Space
  class="login-dialog"
  direction="vertical"
  gap={6}
  align="center"
  justify="center"
>
  <Logo variant="icon" scale={22} />
  <Space width="100%" align="center" size="large">
    <Input label="Username" size="default" width="80%" />
    <Input label="Password" type="password" width="80%" />
  </Space>
  <Button width="35%">Log In</Button>
</Space>
<style lang="scss">
  :global(.login-dialog) {
    display: flex;
    justify-content: center;
    align-items: center;
    width: calc(var(--grid) * 60);
    height: calc(var(--grid) * 70);
    border: 1px solid var(--gray-p2);
    border-radius: var(--border-radius);
    position: absolute;
    top: 0;
    bottom: calc(var(--grid) * 10);
    left: 0;
    right: 0;
    margin: auto;
  }
</style>