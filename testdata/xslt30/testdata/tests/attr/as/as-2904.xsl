<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test of xsl:template which contains a typed element node from input file, or 
  				explicitly created xsl:element or LRE with xsl:type, 
  				@as="element(QName, built-in primitive/derived atomic type)".-->

   <xslt:import-schema schema-location="builtinTypeSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <temp1>
            <xslt:call-template name="t1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="t2"/>
         </temp2>
      </out>
   </xslt:template>

   <xslt:template name="t1" as="element(elem-anyURI, xs:anyURI)">
	     <xslt:copy-of select="doc-schemaas/elem-anyURI" validation="strict" copy-namespaces="no"/>
   </xslt:template>

   <xslt:template name="t2" as="element(elem-NMTOKEN, xs:NMTOKEN)">
	     <elem-NMTOKEN xslt:type="xs:NMTOKEN">business-enforcement::link-chain.common</elem-NMTOKEN>
   </xslt:template>
   
</xslt:transform>
