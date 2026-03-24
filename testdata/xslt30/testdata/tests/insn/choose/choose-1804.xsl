<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Test case in which there are two xsl:otherwise for an xsl:choose instruction. -->

   <xslt:template match="/">
		    <out>
         <xslt:choose>
			         <xslt:when test="true()">
	   			        <xslt:text>number </xslt:text>
	  		       </xslt:when>
	  		       <xslt:otherwise>
				           <xslt:text>No Way </xslt:text>
			         </xslt:otherwise>
			         <xslt:otherwise>
				           <xslt:text>There is a Way </xslt:text>
			         </xslt:otherwise>
	  		    </xslt:choose>
      </out>
	  </xslt:template>
</xslt:transform>
