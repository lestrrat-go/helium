<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Set a variable with version 2.0 applying to xsl:variable,
    but use the variable under version 1.0 to invoke 1.0 XPath rules. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="values"
                  select="(38.56,1.9345E122,17.0034,5.93784E217)"
                  as="xs:double*"/>

   <xslt:template match="/">
      <out>
         <xslt:for-each select="$values">
    	       <xslt:text>
</xslt:text>
    	       <v>
               <xslt:if test="contains(.,'E')" version="1.0">
    	             <xslt:attribute name="toobig">yes</xslt:attribute>
    	          </xslt:if>
               <xslt:value-of select="."/>
            </v>
         </xslt:for-each>
         <xslt:text>
</xslt:text>
      </out>
   </xslt:template>
</xslt:transform>
