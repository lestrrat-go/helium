<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
	<!-- Purpose: Test case using a variable with static="no" attribute, that uses use-when with an EBV of true on an xsl:choose instruction. -->

   <xslt:variable name="x" as="xs:string" select="'error'" static=" no"/>

   <xslt:template match="doc">   
		    <out>
         <xslt:choose use-when="true()">
		          <xslt:when test="($x castable as xs:integer)" use-when="false()">
		             <xslt:sequence select="-1"/>
		          </xslt:when>
		          <xslt:when test="not($x castable as xs:integer)" use-when="true()">
		             <xslt:sequence select="-1"/>
		          </xslt:when>
		          <xslt:when test="xs:integer($x) lt 0" use-when="'a'='a'">
	  	           <xslt:sequence select="-2"/>
	 	         </xslt:when>
	 	         <xslt:otherwise use-when="1=1">
	 	            <xslt:sequence select="xs:integer($x)"/>
		          </xslt:otherwise>
		       </xslt:choose>
      </out>
	  </xslt:template>
</xslt:transform>
