<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case that uses xsl:document inside a literal result element. -->

   <t:template match="doc">
      <out>
         <t:document>
            <child number="1">child element</child>
         </t:document>
      </out>
   </t:template>
</t:transform>
