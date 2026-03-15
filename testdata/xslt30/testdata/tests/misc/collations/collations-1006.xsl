<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0"
   expand-text="true">
<!-- Purpose: To verify that the default-collation is actually used by various constructs.-->

   <t:template match="doc"
      default-collation="http://www.w3.org/2013/collation/UCA?strength=secondary;fallback=no">
      <out>
         <a>{'abc' = 'ABC'}</a>
         <b>{'abc' eq 'ABC'}</b>
         <c>{starts-with('abc', 'AB')}</c>
         <d>{compare('abc', 'ABc') eq 0}</d>
         <e><t:perform-sort select="('Ab', 'aBcD', 'aBC')">
            <t:sort/>
         </t:perform-sort></e>
      </out> 
   </t:template>
</t:transform>
