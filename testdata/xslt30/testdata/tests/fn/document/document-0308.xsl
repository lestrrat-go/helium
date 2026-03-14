<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:strip-space, xsl:preserve-space and doc($uri).-->

   <t:strip-space elements="*"/>

   <t:preserve-space elements="str1"/>

   <t:template match="/doc">
         <t:copy-of select="doc('doc13.xml')"/>     
   </t:template>
   
</t:transform>
