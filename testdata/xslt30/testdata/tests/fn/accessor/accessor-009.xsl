<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:base-uri accessor with empty seqeunce as argument -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="base-uri(()) instance of empty-sequence()"/>
         </t1>
         <t2>
            <xslt:value-of select="base-uri(*) instance of empty-sequence()"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
