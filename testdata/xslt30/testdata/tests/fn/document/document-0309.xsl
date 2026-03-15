<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="innerdoc/"
             version="2.0">
<!-- Purpose: Testcase with doc($uri) where the base URI is set in the stylesheet by xml:base.-->

   <t:template match="/">
      <out>
         <t:copy-of select="doc('doc06.xml')"/>
      </out>
   </t:template>
</t:transform>
