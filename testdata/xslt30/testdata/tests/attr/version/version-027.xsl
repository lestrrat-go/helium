<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="8.5">
<!-- Purpose: Later-version (8.5) stylesheet has (fictional) 8.5 declaration guarded by use-when testing system-property. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:fabulous-8.5-environment charm="+50" use-when="system-property('t:version')='8.5'"/>

   <t:template match="/">
      <out>success</out>
   </t:template>
</t:transform>
