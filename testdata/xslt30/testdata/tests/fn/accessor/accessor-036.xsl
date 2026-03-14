<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:document-uri in conjunction with document(), doc()-->

   <xslt:template match="/">
	     <out>
         <t1>
            <xslt:value-of select="ends-with(document-uri(document('')), 'accessor-036.xsl')"/>
         </t1>
         <t2>
            <xslt:value-of select="ends-with(document-uri(doc('')),'accessor-036.xsl')"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
