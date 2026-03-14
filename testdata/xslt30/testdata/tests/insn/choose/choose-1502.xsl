<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case that uses the standard attribute [xsl:]xpath-default-namespace on an 
  				xsl:choose instruction and the test is a path expression. -->

   <xslt:variable name="true" as="xs:boolean" select="true()"/>

   <xslt:template match="/">
 	    <out>
         <xslt:choose>
    		      <xslt:when test="$true" xpath-default-namespace="">
      			      <xslt:value-of select="count(//elem)"/>
   			      </xslt:when>
   		    </xslt:choose>
         <xslt:choose xpath-default-namespace="">
    		      <xslt:when test="false()">
      			      <xslt:value-of select="count(//elem)"/>
   			      </xslt:when>
    		      <xslt:otherwise xpath-default-namespace="www.example.com/reset/ns">
     			       <xslt:value-of select="count(//elem)"/>
    		      </xslt:otherwise>
		       </xslt:choose>
      </out>
	  </xslt:template>
</xslt:transform>
