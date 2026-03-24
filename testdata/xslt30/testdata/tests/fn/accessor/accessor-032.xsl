<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:document-uri accessor with empty sequence as argument -->

   <xslt:template match="/doc/inner">
      <out>
         <t1>
            <xslt:value-of select="document-uri(()) instance of empty-sequence()"/>
         </t1>
         <t2>
            <xslt:value-of select="document-uri(*) instance of empty-sequence()"/>
         </t2>
      </out>  
   </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
