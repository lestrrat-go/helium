<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case in which the test attribute of an xsl:if instruction is a sequence of 
  				nodes and atomic values. -->

   <xslt:variable name="v" as="element()">
		    <xslt:element name="e1">fitst</xslt:element>
	  </xslt:variable>

   <xslt:variable name="seq">
		    <xslt:sequence select="($v, xs:integer('12'), ($v, $v))"/>
	  </xslt:variable>

   <xslt:template match="/">
		    <out>
         <xslt:if test="$seq">
	   			     <xslt:text>text</xslt:text>
	  		    </xslt:if>
      </out>
	  </xslt:template>
</xslt:transform>
