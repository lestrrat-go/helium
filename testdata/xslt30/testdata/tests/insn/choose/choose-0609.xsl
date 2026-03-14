<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case to verify that the test attribute of an xsl:if instruction results to 
  				'false' where the operand is an empty sequence. -->

   <xslt:variable name="v">
		    <xslt:sequence select="()"/>
	  </xslt:variable>

   <xslt:variable name="v2" as="xs:untypedAtomic" select="xs:untypedAtomic('')"/>

   <xslt:template match="/">
		    <out>
         <xslt:if test="$v=()">
	   			     <xslt:value-of select="'text'"/>
	  		    </xslt:if>
         <xslt:if test="()">
	   			     <xslt:value-of select="'text'"/>
	  		    </xslt:if>
         <xslt:if test="boolean($v2)">
	   			     <xslt:value-of select="'text'"/>
	  		    </xslt:if>
      </out>
	  </xslt:template>
</xslt:transform>
