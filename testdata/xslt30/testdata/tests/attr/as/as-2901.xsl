<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable with @select selecting a typed element 
  				node from input file, @as="element(QName, built-in primitive/derived atomic type)".-->

   <xslt:variable name="v1"
                  select="/doc-schemaas/elem-anyURI"
                  as="element(elem-anyURI, xs:anyURI)"/>

   <xslt:variable name="v2"
                  select="/doc-schemaas/elem-NMTOKEN"
                  as="element(elem-NMTOKEN, xs:NMTOKEN)"/>

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
