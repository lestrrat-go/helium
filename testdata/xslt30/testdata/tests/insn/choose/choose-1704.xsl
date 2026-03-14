<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case in which the operand of the test attribute of an xsl:when instruction 
  				is a singleton value of type xs:string, fn:boolean returns false if the operand 
  				value has zero length; otherwise it returns true. -->

   <xslt:variable name="v2" as="xs:string" select="''"/>

   <xslt:template match="/">
		    <out>
         <xslt:choose>
				        <xslt:when test="$v2">
		   			       <xslt:text>v2 value </xslt:text> 
		  		      </xslt:when>
				        <xslt:otherwise> 
		   			       <xslt:text>value in otherwise</xslt:text>
		  		      </xslt:otherwise>	 
			      </xslt:choose>
      </out>
	  </xslt:template>
</xslt:transform>
