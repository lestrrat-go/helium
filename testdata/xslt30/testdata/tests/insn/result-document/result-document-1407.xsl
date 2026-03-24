<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <xsl:param name="a" select="'a'"/>
   
   <!-- Purpose: Adaptive serialization: no errors -->

   <xsl:template name="xsl:initial-template">
      <xsl:result-document method="ad{$a}ptive" encoding="us-{$a}scii">
         <xsl:map-entry key="'a'" select="1"/>
         <xsl:sequence select="name#1"/>
         <xsl:sequence select="1 to 5"/>
         <xsl:attribute name="att"/>
         <xsl:comment>Non-ascii char &#xaa;</xsl:comment>
      </xsl:result-document>
   </xsl:template>
</xsl:transform>
