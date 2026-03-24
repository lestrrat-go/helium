<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Test case in which xsl:otherwise comes before xsl:when instruction. -->

   <xslt:template match="/">
		    <out>
         <xslt:choose>
			         <xslt:otherwise>
				           <xslt:text>No Way </xslt:text>
			         </xslt:otherwise>
			         <xslt:when test="true()">
	   			        <xslt:text>number </xslt:text>
	  		       </xslt:when>
	  		    </xslt:choose>
      </out>
	  </xslt:template>
</xslt:transform>
