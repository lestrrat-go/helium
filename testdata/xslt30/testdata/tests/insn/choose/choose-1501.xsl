<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case that uses the standard attribute [xsl:]xpath-default-namespace on an 
  				xsl:if instruction and the test is a path expression. -->

   <xslt:variable name="true" as="xs:boolean" select="true()"/>

   <xslt:template match="/">
 	    <out>
         <xslt:if test="$true" xpath-default-namespace="http://example.com/ns1">
 			        <xslt:value-of select="count(//elem)"/>
 		      </xslt:if>
         <xslt:if test="$true">
 			        <xslt:value-of select="count(//elem)" xpath-default-namespace="http://example.com/ns2"/>
 		      </xslt:if>
      </out>
	  </xslt:template>
</xslt:transform>
