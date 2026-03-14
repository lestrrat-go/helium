<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case in which the operand of the test attribute in an xsl:if instruction is a 
  				sequences of node, zero-length strings and numeric zeroes in one instance and a 
  				sequence of a node and two boolean values, in another case. -->

   <xslt:variable name="v" select="''"/>

   <xslt:variable name="var" as="node()">
		    <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
	  </xslt:variable>

   <xslt:variable name="v1">
		    <xslt:sequence select="($var, '', 0, $v)"/>
	  </xslt:variable>

   <xslt:variable name="v2">
		    <xslt:sequence select="($var, false(), true())"/>
	  </xslt:variable>

   <xslt:template match="/">
		    <out>
         <xslt:text>
</xslt:text>
         <xslt:if test="boolean($v1)">
	   			     <a>
               <xslt:text>v1 value </xslt:text>
            </a>
	  		    </xslt:if>
         <xslt:text>
</xslt:text>
         <xslt:if test="boolean($v2)">
		  		      <b>
               <xslt:text>v2 value</xslt:text>
            </b>
	  		    </xslt:if>
         <xslt:text>
</xslt:text>
      </out>
	  </xslt:template>
</xslt:transform>
