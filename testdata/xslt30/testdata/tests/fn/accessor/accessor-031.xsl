<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:document-uri accessor with document node as argument -->

   <xslt:template match="/doc">
  
      <out>
         <xslt:variable name="temp" select="/"/>
         <t1>
            <xslt:value-of select="ends-with(document-uri(/), 'accessor021.xml')"/>
         </t1>
         <t2>
            <xslt:value-of select="ends-with(document-uri($temp), 'accessor021.xml')"/>
         </t2>
         <t3>
            <xslt:value-of select="ends-with(document-uri( subsequence( /, 1)),'accessor021.xml')"/>
         </t3>
      </out>
  
   </xslt:template>
</xslt:transform>
