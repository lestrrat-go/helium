<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0" use-when="false()">
   <!-- Purpose: Test case that uses use-when on xsl:stylesheet to verify it has no effect 
  				on this element but it takes effect on all its children.  -->

   <t:template match="elem">
      <out>
         <t:copy>
            <t:apply-templates/>
         </t:copy>
      </out>
   </t:template>

   <t:template match="a | b" use-when="true()">
      <print>
         <t:next-match/>
      </print>
   </t:template>
</t:transform>
