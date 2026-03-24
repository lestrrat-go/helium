<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with xpath-default-namespace attribute declared on the xsl:stylesheet element. 
  				Tested XPath expressions include unprefixed element and type names in:
				-@select of xsl:value-of and xsl:apply-templates-->

   <xslt:import-schema namespace="http://www.defaultnamespace.example.com/ns/schema"
                       schema-location="schemaForDefaultnamespace.xsd"/>

   <xslt:template match="/doc-default">
      <out>
         <one>
            <xslt:value-of select="element(elem, myPartNumberType)"/>
         </one>
         <xslt:apply-templates select="element(elem, myPartNumberType)"/>
      </out>
   </xslt:template>

   <xslt:template match="elem">
      <two>hello</two>
   </xslt:template>
</xslt:transform>
