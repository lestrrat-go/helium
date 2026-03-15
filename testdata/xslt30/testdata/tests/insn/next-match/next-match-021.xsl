<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test of xsl:next-match in a template where the pattern has a union 
  				operator with Name Test and element(E,T) that match one and the same node. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="doc">
      <out>
         <xslt:apply-templates select="one"/>
      </out>
   </xslt:template>

   <xslt:template match="one|element(one, xs:untyped)">
	     <one xmlns:xsl="http://www.w3.org/1999/XSL/Transform">one time</one>
	     <xslt:next-match/>
   </xslt:template>

   <xslt:template match="node()">
	     <two xmlns:xsl="http://www.w3.org/1999/XSL/Transform">finally</two>
   </xslt:template>
</xslt:transform>
