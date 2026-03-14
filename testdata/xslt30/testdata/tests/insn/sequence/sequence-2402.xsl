<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: empty xsl:sequence instruction is missing the REQUIRED 'select' attribute. 
        (OK in XSLT 3.0) 
   -->

   <t:template match="doc">
      <t:variable name="q" as="item() *">
         <t:sequence/>
      </t:variable>
      <out>
         <t:value-of select="data($q)" separator=","/>
      </out>
   </t:template>
</t:transform>
