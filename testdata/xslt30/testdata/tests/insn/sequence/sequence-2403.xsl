<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: xsl:sequence with content but no 'select' attribute. 
        (OK in XSLT 3.0) 
   -->

   <t:template match="doc">
      <t:variable name="q" as="item() *">
         <t:sequence>
            <t:element name="foo"/>
            <t:element name="bar"/>
         </t:sequence>
      </t:variable>
      <out>
         <t:copy-of select="$q"/>
      </out>
   </t:template>
</t:transform>
