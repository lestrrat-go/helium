<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.uri.func"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test type of result of xsl:function with @as="element(QName, built-in primitive/derived atomic type)". 
  				Sequence constructor copies a typed element from input file.-->

   <xslt:import-schema schema-location="builtinTypeSchemaAs.xsd"/>

   <xslt:function name="my:func1" as="element(elem-anyURI, xs:anyURI)">
	     <xslt:copy-of select="document('as-29.xml')/doc-schemaas/elem-anyURI"
                    validation="strict"/>
   </xslt:function>

   <xslt:function name="my:func2" as="element(elem-NMTOKEN, xs:NMTOKEN)">
      <xslt:copy-of select="document('as-29.xml')/doc-schemaas/elem-NMTOKEN"
                    validation="strict"/>
   </xslt:function>

   <xslt:template match="/">
      <out>
         <func1>
            <xslt:value-of select="my:func1() instance of element(elem-anyURI, xs:anyURI)"/>
         </func1>
         <func2>
            <xslt:value-of select="my:func2() instance of element(elem-NMTOKEN, xs:NMTOKEN)"/>
         </func2>
      </out>
   </xslt:template>
   
</xslt:transform>
