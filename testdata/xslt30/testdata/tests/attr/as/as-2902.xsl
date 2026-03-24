<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable without @select and @as="element(QName, built-in 
  				primitive/derived atomic type)". Sequence constructor contains a typed element from 
  				input file, or explicitly created xsl:element or LRE.-->

   <xslt:variable name="v1" as="element(elem-anyURI, xs:anyURI)">
	     <xslt:copy-of select="/doc-schemaas/elem-anyURI" validation="preserve"/>
   </xslt:variable>

   <xslt:variable name="v2" as="element(elem-NMTOKEN, xs:NMTOKEN)">
	     <elem-NMTOKEN xslt:type="xs:NMTOKEN">business-enforcement::link-chain.common</elem-NMTOKEN>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <var1>
            <xslt:value-of select="$v1 instance of element(elem-anyURI, xs:anyURI)"/>
         </var1>
         <var2>
            <xslt:value-of select="$v2 instance of element(elem-NMTOKEN, xs:NMTOKEN)"/>
         </var2>
      </out>
   </xslt:template>
</xslt:transform>
